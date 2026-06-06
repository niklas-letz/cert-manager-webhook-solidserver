# Cert-manager ACME webhook for EfficientIP SOLIDserver DDI

A cert-manager ACME DNS-01 webhook solver that manages the full lifecycle of TXT validation records via the EfficientIP SOLIDserver REST API.

Based on [cert-manager/webhook-example](https://github.com/cert-manager/webhook-example).

API documentation: [solidserver-go-client](https://github.com/EfficientIP-Labs/solidserver-go-client/tree/main)

## Requirements

- [Go](https://golang.org/) >= 1.26.0
- [Helm](https://helm.sh/) >= v3.0.0
- [Kubernetes](https://kubernetes.io/) >= v1.35.0
- [cert-manager](https://cert-manager.io/) >= 1.20.0

## Installation

### cert-manager

Follow the [instructions](https://cert-manager.io/docs/installation/) in the cert-manager documentation to install it within your cluster.

### Webhook

The Helm chart is published on GitHub Pages and GHCR. You can install it via Helm repo, OCI, or directly from a local checkout.

#### From Helm repo (recommended)

```bash
helm repo add solidserver-webhook https://niklas-letz.github.io/cert-manager-webhook-solidserver
helm repo update
helm install solidserver-webhook solidserver-webhook/cert-manager-webhook-solidserver \
  --version <version> \
  --namespace cert-manager
```

#### From OCI registry (GHCR)

```bash
helm install solidserver-webhook \
  oci://ghcr.io/niklas-letz/charts/cert-manager-webhook-solidserver \
  --version <version> \
  --namespace cert-manager
```

#### From local checkout

```bash
helm install --namespace cert-manager solidserver-webhook deploy/cert-manager-webhook-solidserver
```

**Note**: The webhook must be deployed in the same namespace as cert-manager.

To uninstall:

```bash
helm uninstall --namespace cert-manager solidserver-webhook
```

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
            groupName: solidserver-webhook.niklasletz.com
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

## Development

### Running the test suite

See [testdata/solidserver/README.md](testdata/solidserver/README.md) for setup instructions.

```bash
export SOLIDSERVER_USERNAME=<username>
export SOLIDSERVER_PASSWORD=<password>

TEST_DNS_SERVER=<dns-server-ip> TEST_ZONE_NAME=example.com. make test
```

### Building the container image

```bash
make docker-build        # Docker build (linux/amd64)
make podman-build        # Podman build (linux/amd64)
make docker-build-fast   # Pre-compile locally, then Docker build with Dockerfile.fast
make podman-build-fast   # Pre-compile locally, then Podman build with Dockerfile.fast
make docker-push         # Push image to registry using Docker
make podman-push         # Push image to registry using Podman
```

### Rendering the Kubernetes manifest

```bash
make render-helm         # Generate a single Kubernetes manifest from the Helm chart
```

### Cleaning up

```bash
make clean               # Remove _test/ and _out/ directories
```
