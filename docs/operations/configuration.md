# Configuration

Nivora configuration is YAML-based and loaded by server, worker, runner, and CLI local workflows.

## Files

Local development:

- `configs/server.yaml`
- `configs/worker.yaml`
- `configs/runner.yaml`

Docker Compose:

- `configs/docker-compose.server.yaml`
- `configs/docker-compose.worker.yaml`
- `configs/docker-compose.runner.yaml`

Production-shaped example:

- `configs/production.example.yaml`

The production example is not a production-ready configuration. It shows field shape and safe defaults without secret values.

Production-like package profiles:

- `deployments/helm/values-production.yaml`
- `deployments/docker-compose/docker-compose.production.example.yaml`

## Validate

```sh
go run ./cmd/nivora config validate --file configs/server.yaml
go run ./cmd/nivora config validate --file configs/production.example.yaml
```

For production-like posture checks, use the read-only doctor command:

```sh
go run ./cmd/nivora doctor --file configs/production.example.yaml
go run ./cmd/nivora doctor security --file configs/production.example.yaml
go run ./cmd/nivora doctor runtime --file configs/production.example.yaml
```

`nivora doctor` checks config guardrails and reports live-only areas as `NOT_CHECKED`; it is not a production-readiness certificate.

## Important Fields

- `app.name`: component name.
- `environment`: local, docker-compose, kubernetes, or another deployment label.
- `http.bind_address`: bind address for the process.
- `database.url`: PostgreSQL connection string.
- `database.runtime_store`: `memory` for local/CI mode or `postgres` for production-like runtime stores. Production mode rejects `memory`.
- `event_bus.type`: currently `memory` by default.
- `object_store.type`: currently `local` by default.
- `object_store.path`: local object-store directory.
- `log.level`: info, debug, warn, or error.
- `telemetry.enabled`: tracing/metrics integration switch for future external telemetry.
- `auth`: local dev, token, or OIDC foundation configuration. Token values and OIDC secrets must come from environment variables or secret providers, not committed files.
- `mcp`: local stdio MCP control-plane settings. `request_timeout`, `max_response_bytes`, and `max_requests_per_minute` cap local AI inspection calls; they do not make remote MCP safe.
- `runner`: runner name, group, and heartbeat interval.
- `runtime`: unsafe capability flags. Production mode rejects local shell executor, privileged executor, remote host deploy, Kubernetes apply, Argo sync, and insecure registry when enabled globally.

## Production Validation

When `environment` is `production` or `prod`, validation rejects:

- `database.runtime_store: memory`
- `auth.enabled: false`
- `auth.mode: dev` or `disabled`
- token auth without `auth.static_token_env`
- inline database passwords in `database.url`
- enabled MCP without `mcp.request_timeout`
- enabled MCP without a positive `mcp.max_response_bytes`
- enabled MCP without a positive `mcp.max_requests_per_minute`
- unsafe runtime flags enabled globally

Use `configs/production.example.yaml` and inject secrets through environment variables or a secret provider. Do not put database passwords, auth tokens, kubeconfigs, registry passwords, SSH keys, or webhook secrets directly in committed config.

Profile smoke checks:

```sh
./scripts/smoke-helm-production-profile.sh
./scripts/smoke-compose-production-profile.sh
./scripts/smoke-audit-durability.sh
```

## Secrets

Do not put real secret values in committed config files.

Use environment variables, Kubernetes Secrets, SecretRef/CredentialRef records, or future external secret providers. Config examples may name environment variables such as `NIVORA_AUTH_TOKEN`, but they must not contain the token value.

## Packaging Notes

The Docker image uses the `nivora` CLI as its entrypoint. The process mode is selected by command arguments:

```sh
nivora server --config /etc/nivora/server.yaml
nivora worker --config /etc/nivora/worker.yaml
nivora runner --config /etc/nivora/runner.yaml
```
