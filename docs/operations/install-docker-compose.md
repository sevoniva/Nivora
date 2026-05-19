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

## Health Checks

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/system/diagnostics
```

## Stop

```sh
docker compose -f deployments/docker-compose/docker-compose.yaml down
```

Add `-v` only when you intentionally want to remove local volumes.
