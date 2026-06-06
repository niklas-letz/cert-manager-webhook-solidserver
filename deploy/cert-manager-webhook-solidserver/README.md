# cert-manager-webhook-solidserver

A Helm chart for the cert-manager ACME DNS-01 webhook solver for EfficientIP SOLIDserver DDI.

## Requirements

- [Kubernetes](https://kubernetes.io/) >= v1.35.0
- [cert-manager](https://cert-manager.io/) >= 1.20.0
- [Helm](https://helm.sh/) >= v3.0.0

## Installation

**Note:** The webhook must be deployed in the same namespace as cert-manager.

```bash
helm repo add solidserver-webhook https://niklas-letz.github.io/cert-manager-webhook-solidserver
helm repo update
helm install solidserver-webhook solidserver-webhook/cert-manager-webhook-solidserver \
  --version <version> \
  --namespace cert-manager
```

## Values

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `groupName` | string | `acme.solidserver.niklasletz.dev` | The API group name referenced in the Issuer/ClusterIssuer webhook stanza |
| `certManager.namespace` | string | `cert-manager` | The namespace where cert-manager is deployed |
| `certManager.serviceAccountName` | string | `cert-manager` | The service account name used by cert-manager |
| `image.registry` | string | `ghcr.io` | Container image registry |
| `image.repository` | string | `niklas-letz/cert-manager-webhook-solidserver` | Container image repository |
| `image.tag` | string | `""` | Container image tag (defaults to chart `appVersion`) |
| `image.pullPolicy` | string | `IfNotPresent` | Container image pull policy |
| `nameOverride` | string | `""` | Override the resource name prefix |
| `fullnameOverride` | string | `""` | Override the full resource name |
| `service.type` | string | `ClusterIP` | Kubernetes Service type |
| `service.port` | integer | `443` | Service port |
| `resources` | object | `{}` | Pod resource requests and limits |
| `nodeSelector` | object | `{}` | Node selector for pod assignment |
| `tolerations` | list | `[]` | Tolerations for pod assignment |
| `affinity` | object | `{}` | Affinity for pod assignment |

## Issuer / ClusterIssuer

Create an `Issuer` or `ClusterIssuer` resource with the SOLIDserver webhook solver:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: mail@example.com
    privateKeySecretRef:
      name: letsencrypt-staging
    solvers:
      - dns01:
          webhook:
            groupName: acme.solidserver.niklasletz.dev
            solverName: solidserver
            config:
              host: sds.example.com
              serverName: dns.example.com
              zoneName: test.example.com
              usernameSecretRef:
                name: solidserver-credentials
                key: username
              passwordSecretRef:
                name: solidserver-credentials
                key: password
```

**Note:** The example uses Let's Encrypt's **staging** environment. Once your setup works, switch to production by changing the `server` field to `https://acme-v02.api.letsencrypt.org/directory`.

### Config fields

| Field | Required | Description |
| --- | --- | --- |
| `host` | yes | SOLIDserver hostname or IP |
| `port` | no | API port (defaults to `443`) |
| `serverName` | yes | DNS server name configured in SOLIDserver |
| `zoneName` | no | DNS zone name (defaults to the resolved ACME zone) |
| `viewName` | no | DNS view name |
| `username` / `password` | no | Inline credentials (not recommended for production) |
| `usernameSecretRef` / `passwordSecretRef` | no | Kubernetes Secret references for credentials |

### Credentials

Credentials can be provided inline or via a Kubernetes Secret. Using a Secret is recommended for production:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: solidserver-credentials
  namespace: cert-manager
type: Opaque
data:
  username: <base64-encoded-username>
  password: <base64-encoded-password>
```

The default keys are `username` and `password` — override with the `key` field in `usernameSecretRef` / `passwordSecretRef` if different.

## Certificate

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-cert
  namespace: cert-manager
spec:
  dnsNames:
    - example.com
  issuerRef:
    name: letsencrypt-staging
    kind: ClusterIssuer
  secretName: example-cert
```
