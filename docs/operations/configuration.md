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

## Validate

```sh
go run ./cmd/nivora config validate --file configs/server.yaml
go run ./cmd/nivora config validate --file configs/production.example.yaml
```

## Important Fields

- `app.name`: component name.
- `environment`: local, docker-compose, kubernetes, or another deployment label.
- `http.bind_address`: bind address for the process.
- `database.url`: PostgreSQL connection string.
- `event_bus.type`: currently `memory` by default.
- `object_store.type`: currently `local` by default.
- `object_store.path`: local object-store directory.
- `log.level`: info, debug, warn, or error.
- `telemetry.enabled`: tracing/metrics integration switch for future external telemetry.
- `auth`: local dev or token mode configuration.
- `runner`: runner name, group, and heartbeat interval.

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
