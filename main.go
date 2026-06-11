package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	sdsclient "github.com/efficientip-labs/solidserver-go-client/sdsclient"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName,
		&solidserverDNSProviderSolver{},
	)
}

type solidserverDNSProviderSolver struct {
	client *kubernetes.Clientset
}

type solidserverDNSProviderConfig struct {
	Host       string `json:"host"`
	Port       int32  `json:"port,omitempty"`
	ServerName string `json:"serverName"`
	ViewName   string `json:"viewName,omitempty"`
	ZoneName   string `json:"zoneName,omitempty"`

	Username          string                   `json:"username,omitempty"`
	Password          string                   `json:"password,omitempty"`
	UsernameSecretRef cmmeta.SecretKeySelector `json:"usernameSecretRef,omitempty"`
	PasswordSecretRef cmmeta.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

func (s *solidserverDNSProviderSolver) Name() string {
	return "solidserver"
}

func (cfg *solidserverDNSProviderConfig) zoneNameOrDefault(resolvedZone string) string {
	if cfg.ZoneName != "" {
		return cfg.ZoneName
	}
	return strings.TrimSuffix(resolvedZone, ".")
}

var sdsHTTPClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 25 * time.Second,
}

func (s *solidserverDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("Present: namespace=%s, zone=%s, fqdn=%s",
		ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	username, password, err := s.getCredentials(cfg, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	zoneName := cfg.zoneNameOrDefault(ch.ResolvedZone)

	addInput := sdsclient.NewDnsRrAddInput()
	addInput.SetServerName(cfg.ServerName)
	addInput.SetRrName(strings.TrimSuffix(ch.ResolvedFQDN, "."))
	addInput.SetRrType("TXT")
	addInput.SetRrValue1(ch.Key)
	addInput.SetZoneName(zoneName)
	if cfg.ViewName != "" {
		addInput.SetViewName(cfg.ViewName)
	}

	klog.Infof("Present: creating TXT record server=%s fqdn=%s zone=%s view=%s value=%s",
		cfg.ServerName, strings.TrimSuffix(ch.ResolvedFQDN, "."), zoneName, cfg.ViewName, ch.Key)

	apiClient, ctx, cancel := newAPIClient(cfg, username, password)
	defer cancel()

	_, resp, err := apiClient.DnsAPI.DnsRrAdd(ctx).DnsRrAddInput(*addInput).Execute()
	if err != nil {
		return fmt.Errorf("adding DNS record: %w", apiError(err, resp))
	}

	klog.Infof("Present: TXT record created successfully")
	return nil
}

func (s *solidserverDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("CleanUp: dnsName=%s zone=%s", ch.ResolvedFQDN, ch.ResolvedZone)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	username, password, err := s.getCredentials(cfg, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	zoneName := cfg.zoneNameOrDefault(ch.ResolvedZone)

	klog.Infof("CleanUp: deleting TXT record server=%s fqdn=%s zone=%s view=%s value=%s",
		cfg.ServerName, strings.TrimSuffix(ch.ResolvedFQDN, "."), zoneName, cfg.ViewName, ch.Key)

	apiClient, ctx, cancel := newAPIClient(cfg, username, password)
	defer cancel()

	req := apiClient.DnsAPI.DnsRrDelete(ctx).
		RrName(strings.TrimSuffix(ch.ResolvedFQDN, ".")).
		RrValue1(ch.Key).
		ServerName(cfg.ServerName)
	if cfg.ViewName != "" {
		req = req.ViewName(cfg.ViewName)
	}

	_, resp, err := req.Execute()
	if err != nil {
		var apiErr *sdsclient.GenericOpenAPIError
		if errors.As(err, &apiErr) && len(apiErr.Body()) > 0 && strings.Contains(string(apiErr.Body()), "does not exist") {
			klog.Infof("CleanUp: record already deleted (does not exist)")
			return nil
		}
		return fmt.Errorf("deleting DNS record: %w", apiError(err, resp))
	}

	klog.Infof("CleanUp: TXT record deleted successfully")
	return nil
}

func newAPIClient(cfg solidserverDNSProviderConfig, username, password string) (*sdsclient.APIClient, context.Context, context.CancelFunc) {
	sdsCfg := sdsclient.NewConfiguration()
	sdsCfg.HTTPClient = sdsHTTPClient
	sdsCfg.Servers[0].Variables["host"] = sdsclient.ServerVariable{DefaultValue: cfg.Host}
	sdsCfg.Servers[0].Variables["port"] = sdsclient.ServerVariable{DefaultValue: fmt.Sprintf("%d", cfg.Port)}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	ctx = context.WithValue(ctx, sdsclient.ContextBasicAuth, sdsclient.BasicAuth{
		UserName: username,
		Password: password,
	})

	return sdsclient.NewAPIClient(sdsCfg), ctx, cancel
}

func (s *solidserverDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	klog.Infof("Initialize: creating Kubernetes client")
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return fmt.Errorf("creating Kubernetes client: %w", err)
	}
	s.client = cl
	klog.Infof("Initialize: Kubernetes client created successfully")
	return nil
}

func loadConfig(cfgJSON *extapi.JSON) (solidserverDNSProviderConfig, error) {
	cfg := solidserverDNSProviderConfig{}
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("decoding solver config: %w", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 443
	}
	if cfg.Username == "" {
		cfg.Username = os.Getenv("SOLIDSERVER_USERNAME")
	}
	if cfg.Password == "" {
		cfg.Password = os.Getenv("SOLIDSERVER_PASSWORD")
	}
	klog.Infof("Config: host=%s port=%d serverName=%s viewName=%s zoneName=%s credSource=%s",
		cfg.Host, cfg.Port, cfg.ServerName, cfg.ViewName, cfg.ZoneName, credSource(cfg))
	return cfg, nil
}

func (s *solidserverDNSProviderSolver) getCredentials(cfg solidserverDNSProviderConfig, namespace string) (string, string, error) {
	username := cfg.Username
	password := cfg.Password

	if cfg.UsernameSecretRef.Name != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		secret, err := s.client.CoreV1().Secrets(namespace).Get(ctx, cfg.UsernameSecretRef.Name, metav1.GetOptions{})
		if err != nil {
			return "", "", fmt.Errorf("fetching username secret %s/%s: %w", namespace, cfg.UsernameSecretRef.Name, err)
		}
		key := cfg.UsernameSecretRef.Key
		if key == "" {
			key = "username"
		}
		username = string(secret.Data[key])
	}

	if cfg.PasswordSecretRef.Name != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		secret, err := s.client.CoreV1().Secrets(namespace).Get(ctx, cfg.PasswordSecretRef.Name, metav1.GetOptions{})
		if err != nil {
			return "", "", fmt.Errorf("fetching password secret %s/%s: %w", namespace, cfg.PasswordSecretRef.Name, err)
		}
		key := cfg.PasswordSecretRef.Key
		if key == "" {
			key = "password"
		}
		password = string(secret.Data[key])
	}

	return username, password, nil
}

func credSource(cfg solidserverDNSProviderConfig) string {
	if cfg.UsernameSecretRef.Name != "" || cfg.PasswordSecretRef.Name != "" {
		return "secret"
	}
	if cfg.Username != "" || cfg.Password != "" {
		return "config"
	}
	return "env"
}

func apiError(err error, resp *http.Response) error {
	var apiErr *sdsclient.GenericOpenAPIError
	if errors.As(err, &apiErr) && len(apiErr.Body()) > 0 {
		return fmt.Errorf("%w (HTTP %d, body: %s)", err, resp.StatusCode, string(apiErr.Body()))
	}
	if resp != nil {
		return fmt.Errorf("%w (HTTP %d)", err, resp.StatusCode)
	}
	return err
}
