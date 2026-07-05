# Production Install Smoke Tests

Validate that production install profiles are safe and correctly configured before deployment.

## Quick Start

```bash
make smoke-production-install
```

This runs both Helm and Docker Compose production profile validation.

## Helm Production Validation

The Helm production smoke validates `deployments/helm/values-production.yaml`:

- `runtimeStore: postgres` (not memory)
- `auth.enabled: true` with `mode: token`
- All `runtime.allow*` flags are `false` (no unsafe executors)
- No inline secrets in rendered templates
- Environment is `production`
- Optional `helm lint` check

## Docker Compose Production Validation

The Compose production smoke validates `deployments/docker-compose/docker-compose.production.example.yaml`:

- `POSTGRES_PASSWORD` uses `${NIVORA_POSTGRES_PASSWORD:?}` env placeholder (not inline)
- `NIVORA_AUTH_TOKEN` uses `${NIVORA_AUTH_TOKEN:?}` env placeholder (not inline)
- External production config mounted via `${NIVORA_PRODUCTION_CONFIG:?}` placeholder
- No `POSTGRES_HOST_AUTH_METHOD: trust`
- Image tag is not `1.0.0` (must match current beta-candidate version)

## CI Integration

CI runs Helm rendering/linting, Helm safety checks, web/build checks, and the PostgreSQL backup/restore smoke in the `postgres-integration` job. `make smoke-production-install` remains a local operator check that combines the Helm and Docker Compose production profile scripts.

## Operational Notes

- This is a **static validation** — it validates config files and rendered templates, not a live deployment.
- A live deployment smoke test requires a Kubernetes cluster or Docker daemon, which is environment-dependent.
- The production compose file uses `profiles: ["runner"]` — the runner starts only when explicitly enabled.
- Always review production configs before deploying to real environments.
- Image tags should stay aligned with the current VERSION file.
