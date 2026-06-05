package main

import (
	"os"
	"testing"

	dns "github.com/cert-manager/cert-manager/test/acme"
)

var (
	username  = os.Getenv("SOLIDSERVER_USERNAME")
	password  = os.Getenv("SOLIDSERVER_PASSWORD")
	zone      = os.Getenv("TEST_ZONE_NAME")
	dnsServer = os.Getenv("TEST_DNS_SERVER")
)

func TestRunsSuite(t *testing.T) {
	if username == "" || password == "" {
		t.Fatal("SOLIDSERVER_USERNAME and SOLIDSERVER_PASSWORD must be set")
	}

	if dnsServer != "" {
		dnsServer += ":53"
	}

	fixture := dns.NewFixture(&solidserverDNSProviderSolver{},
		dns.SetResolvedZone(zone),
		dns.SetAllowAmbientCredentials(false),
		dns.SetUseAuthoritative(false),
		dns.SetManifestPath("testdata/solidserver"),
		dns.SetDNSServer(dnsServer),
	)
	fixture.RunConformance(t)
}
