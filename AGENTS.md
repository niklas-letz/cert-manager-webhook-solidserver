# AGENTS.md

## Project identity

A cert-manager ACME DNS01 webhook for EfficientIP SOLIDserver DDI. Registered as a solver named `"solidserver"` (`main.go:54`).

## Commands

```bash
make test              # conformance test suite (downloads etcd/apiserver/kubectl)
make docker-build      # multi-stage Docker build (linux/amd64)
make docker-build-fast # pre-compile locally, then Docker build with Dockerfile.fast
make render-helm       # helm template -> _out/rendered-manifest.yaml
make clean             # rm -rf _test/ _out/
```

## Test prerequisites

`main_test.go` is a **cert-manager conformance test**, not unit tests. It creates real DNS TXT records against a live SOLIDserver and verifies propagation.

Required env vars (checked at runtime):
- `SOLIDSERVER_USERNAME`
- `SOLIDSERVER_PASSWORD`

Optional:
- `TEST_ZONE_NAME` — DNS zone for the resolved zone
- `TEST_DNS_SERVER` — DNS server IP for propagation checks (port `:53` appended automatically)

Test fixture config loads from `testdata/solidserver/config.json`. Set real connection details there before running.

The test binary chain is: `test` → kubebuilder binaries (etcd, kube-apiserver, kubectl) → `setup-envtest@latest`. First run downloads ~200MB.

## Architecture

```
main.go          — entrypoint, solver registration, config struct, API client
main_test.go     — conformance test fixture
deploy/          — Helm chart (apiVersion v2)
testdata/        — test fixture config.json
```

**Registration** (`main.go:27-35`): `GroupName` comes from env `GROUP_NAME` (panics if unset). The solver `Name()` returns `"solidserver"` — this is the value used in `ClusterIssuer.spec.acme.solvers[].dns01.webhook.solverName`.

**Config struct** (`main.go:41-52`): `Host`, `Port`, `ServerName`, `ViewName`, `ZoneName`, `Username`, `Password`, `UsernameSecretRef`, `PasswordSecretRef`. Port defaults to 443.

**Credential resolution** (`main.go:206-249`): 3-tier fallback:
1. `UsernameSecretRef`/`PasswordSecretRef` → Kubernetes Secret lookup
2. Inline `Username`/`Password` in config
3. Env vars `SOLIDSERVER_USERNAME`/`SOLIDSERVER_PASSWORD`

Secret key defaults are `"username"` and `"password"` (overridable via the `.key` field on each ref).

## Helm chart

- Named template prefix: `solidserver-webhook` (not `example-webhook`)
- Chart install: `helm install --namespace cert-manager solidserver-webhook deploy/cert-manager-webhook-solidserver`
- `groupName` in `values.yaml` must match the `GroupName` env var and the `webhook.groupName` in ClusterIssuer configs
- The chart auto-generates a full PKI chain (selfSigned CA → serving cert) — no external certs needed
- RBAC grants `cert-manager`'s service account permission to `create` resources in the webhook's API group

## Code conventions

- Go 1.25.0, module `github.com/niklas-letz/cert-manager-webhook-solidserver`
- Docker images target `linux/amd64` only (no multi-arch)
- Alpine 3.23 base image in Dockerfiles
- Lint/format: not configured (no linter or formatter in CI or Makefile)
