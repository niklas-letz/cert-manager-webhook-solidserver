# AGENTS.md

## Project identity

A cert-manager ACME DNS01 webhook for EfficientIP SOLIDserver DDI. Registered as a solver named `"solidserver"` (`main.go:54`).

## Commands

```bash
go build ./...         # quick compile check
go vet ./...           # static analysis

make test              # conformance test suite (downloads etcd/apiserver/kubectl, ~200MB first run)
make docker-build      # Docker build (linux/amd64 only, not multi-arch)
make docker-build-fast # pre-compile locally, then Docker build with Dockerfile.fast
make render-helm       # helm template -> _out/rendered-manifest.yaml
make clean             # rm -rf _test/ _out/
```

## CI workflow

CI runs `golangci-lint` (via `golangci-lint-action@v9`) on every push/PR. The release workflow also builds multi-arch container images (`linux/amd64,linux/arm64`) and publishes the Helm chart to both GitHub Pages (traditional Helm repo) and GHCR (OCI).

Helm chart install via repo:
```bash
helm repo add solidserver https://niklas-letz.github.io/cert-manager-webhook-solidserver
helm install --namespace cert-manager solidserver-webhook solidserver/cert-manager-webhook-solidserver --version <version>
```

## Dependency version lock

**Do not bump `k8s.io/*` or `sigs.k8s.io/*`** beyond v0.35.x. `cert-manager v1.20.2` depends on `controller-runtime v0.23.1`, which is incompatible with `client-go v0.36.x` (`missing method HasSyncedChecker`). Dependabot is configured to ignore these packages (`.github/dependabot.yml:21-22`).

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

## Test prerequisites

`main_test.go` is a **cert-manager conformance test**, not unit tests. It creates real DNS TXT records against a live SOLIDserver and verifies propagation.

Required env vars:
- `SOLIDSERVER_USERNAME`
- `SOLIDSERVER_PASSWORD`

Optional:
- `TEST_ZONE_NAME` — DNS zone for the resolved zone
- `TEST_DNS_SERVER` — DNS server IP for propagation checks (port `:53` appended automatically)

Test fixture config loads from `testdata/solidserver/config.json`. Set real connection details there before running.

## Code conventions

- Go 1.26.0, module `github.com/niklas-letz/cert-manager-webhook-solidserver`
- Alpine 3.23 base image in Dockerfiles
- Dockerfile uses `--platform=$BUILDPLATFORM` for the builder stage so Go cross-compiles arm64 natively on amd64 runners (avoids slow QEMU emulation)
- No lint/format config in repo — golangci-lint runs with defaults in CI
- CA certificates are copied from the builder image; the final image has no `apk add` steps
