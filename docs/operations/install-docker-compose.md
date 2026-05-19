# Install with Docker Compose

This Docker Compose setup is for local development and packaging validation. It is not a production deployment guide.

## Build and Start

```sh
make docker-build
docker compose -f deployments/docker-compose/docker-compose.yaml up
```

Or use the helper:

```sh
./scripts/dev-up.sh
```

The compose file starts:

- PostgreSQL
- MinIO
- Nivora server
- Nivora worker
- optional Nivora runner profile

`deployments/docker-compose/docker-compose.yaml` is intentionally a development profile. It uses local configs, trust authentication for PostgreSQL, and development-safe defaults. Do not reuse it as a production baseline.

To start the runner profile:

```sh
docker compose -f deployments/docker-compose/docker-compose.yaml --profile runner up
```

## Configuration

Compose uses:

- `configs/docker-compose.server.yaml`
- `configs/docker-compose.worker.yaml`
- `configs/docker-compose.runner.yaml`

The local PostgreSQL service uses trust authentication to avoid committing a password into the repository. Do not reuse this setup for production.

MinIO values are local-only placeholders and can be overridden with:

```sh
export NIVORA_MINIO_ROOT_USER='local-user'
export NIVORA_MINIO_ROOT_PASSWORD='set-a-local-value'
```

Do not commit real credentials.

## Production-Like Example

`deployments/docker-compose/docker-compose.production.example.yaml` is a safer profile for operator review. It does not embed secrets. It expects:

```sh
export NIVORA_POSTGRES_PASSWORD='<set outside git>'
export NIVORA_AUTH_TOKEN='<set outside git>'
export NIVORA_PRODUCTION_CONFIG=/absolute/path/to/production.yaml
```

The mounted production config should be based on `configs/production.example.yaml`, use `database.runtime_store: postgres`, keep auth enabled, and keep unsafe runtime flags disabled. Validate the profile without starting services:

```sh
./scripts/smoke-compose-production-profile.sh
```

This profile is still not a production-ready deployment; it is a packaging and restore-drill aid.

## Health Checks

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/system/diagnostics
```

`/readyz` includes dependency checks for database, object store, event bus, outbox recovery, and runner reconnect posture. Docker Compose remains local-only; back up its PostgreSQL and MinIO volumes before testing restore flows.

## Backup Notes

- PostgreSQL data is in the compose database volume.
- MinIO/object data is in the compose object-store volume.
- Config files live under `configs/`.
- Secret values should come from environment variables or an external secret provider, not committed compose files.
- For production-like drills, keep PostgreSQL credentials out of committed YAML and inject them through the operator's environment or secret system.

See [Backup and Restore](backup-restore.md) and [HA and Disaster Recovery](ha-disaster-recovery.md).

## Stop

```sh
docker compose -f deployments/docker-compose/docker-compose.yaml down
```

Add `-v` only when you intentionally want to remove local volumes.
