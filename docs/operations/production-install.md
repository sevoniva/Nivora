# Production-Direction Install Guide

This guide describes the install review path for a `v1.0.0-rc.1` candidate. It is intentionally cautious: Nivora should not be described as production-ready until the RC checklist is completed and maintainers explicitly approve that claim.

## Install Modes To Validate

| Mode | Purpose | Required check |
| --- | --- | --- |
| Local binaries | Verify the built server, worker, runner, and CLI can start with reviewed config. | `go build ./cmd/nivora-server ./cmd/nivora-worker ./cmd/nivora-runner ./cmd/nivora` |
| Docker Compose | Validate local packaging with PostgreSQL, object store, server, worker, and runner profile. | `make docker-build` and `./scripts/dev-up.sh` in a disposable environment |
| Helm | Validate Kubernetes manifests render and lint. | `make helm-template` and `make helm-lint` |
| Raw manifests | Keep minimal examples available for review only. | Inspect `deployments/k8s/`; do not treat them as hardened production manifests |

## Preflight Checklist

1. Run `make verify`.
2. Confirm `./scripts/verify-no-secrets.sh` passes.
3. Review `configs/production.example.yaml` and replace placeholder values only in a private operator-controlled config.
4. Confirm `database.runtime_store` is set intentionally.
5. Confirm PostgreSQL and object-store backup procedures exist before persistence validation.
6. Confirm auth, runner tokens, registry credentials, cloud credentials, kubeconfigs, and webhook secrets are supplied through environment variables, Kubernetes Secrets, or an external secret provider.
7. Confirm guarded operations remain disabled by default: Kubernetes apply, Argo CD sync, host remote deploy, Git push, destructive rollback, and pruning.

## Local Binary Review

Build all binaries:

```sh
go build ./cmd/nivora-server ./cmd/nivora-worker ./cmd/nivora-runner ./cmd/nivora
```

Validate config before starting:

```sh
go run ./cmd/nivora config validate --file configs/server.yaml
go run ./cmd/nivora config validate --file configs/production.example.yaml
```

Start components with explicit config:

```sh
nivora server --config configs/server.yaml
nivora worker --config configs/worker.yaml
nivora runner --config configs/runner.yaml
```

Check health:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/system/diagnostics
```

## Docker Compose Review

Docker Compose is a local validation path, not the recommended production deployment mode.

```sh
make docker-build
./scripts/dev-up.sh
curl http://localhost:8080/readyz
./scripts/dev-down.sh
```

Do not reuse the local trust-auth database posture or placeholder object-store credentials outside disposable environments.

## Helm Review

Render and lint:

```sh
make helm-template
make helm-lint
```

Install only into an explicitly selected disposable context:

```sh
NIVORA_ALLOW_KIND_INSTALL=true make kind-install
```

Before any non-disposable Helm validation, review:

- database connection and migration posture
- object-store persistence
- auth mode and token/secret source
- runner token distribution
- ingress/TLS configuration
- resource requests and limits
- backup and restore process

## Current Limitations

- The chart is a packaging foundation, not an operator.
- HA, backup, restore, secret-provider integration, and migration execution require operator-controlled procedures.
- Some runtime repositories still have partial persistence coverage.
- External systems such as Kubernetes clusters, Argo CD, registries, Vault, and cloud providers remain optional and guarded.
