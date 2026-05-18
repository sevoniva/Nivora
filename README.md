# Nivora

Nivora is an open-source DevOps delivery control plane for CI/CD, GitOps, multi-cloud deployment, artifact orchestration, policy gates, runners, release audit, and future visualization APIs.

Phase 0 is a backend skeleton. It defines the module boundaries, APIs, local development shape, and minimal runnable services. It does not implement full CI/CD execution, production GitOps reconciliation, cloud integrations, or a frontend.

## Architecture Summary

Nivora starts as a modular monolith in Go. The control plane and runner are separate binaries, while domain models stay independent from HTTP, databases, queues, cloud SDKs, Kubernetes SDKs, and vendor integrations.

The code follows ports/adapters boundaries:

- `internal/domain`: small domain structs and statuses.
- `internal/usecase`: future business use cases.
- `internal/ports`: interfaces for SCM, artifacts, cloud, executors, workflow, secrets, notifications, policies, events, and object storage.
- `internal/adapters`: placeholder or local implementations.
- `internal/api`: HTTP and gRPC API boundaries.
- `internal/infra`: config, logging, database, migrations, auth, telemetry, and queue infrastructure.

## Components

- `nivora-server`: HTTP control plane with health, readiness, version, system info, and placeholder API groups.
- `nivora-worker`: background runtime placeholder for event consumption and workflow advancement.
- `nivora-runner`: runner runtime placeholder with registration, heartbeat, and a basic shell executor.
- `nivora`: CLI for version, service startup, and config validation.

## Local Development

This repository defaults Go module downloads to a China-friendly proxy:

```sh
export GOPROXY=https://goproxy.cn,direct
```

Start local dependencies and services:

```sh
make dev-up
```

Stop them:

```sh
make dev-down
```

Run checks:

```sh
make tidy
make test
make build
```

## Run Server

```sh
go run ./cmd/nivora server --config configs/server.yaml
```

The server listens on `:8080` by default.

## Run Worker

```sh
go run ./cmd/nivora worker --config configs/worker.yaml
```

## Run Runner

```sh
go run ./cmd/nivora runner --config configs/runner.yaml
```

## Example API Call

```sh
curl http://localhost:8080/api/v1/version
```

Unfinished API groups return a structured Phase 0 `not_implemented` JSON response.

## Roadmap

See [ROADMAP.md](ROADMAP.md).

## License

Nivora is licensed under the Apache License 2.0.

