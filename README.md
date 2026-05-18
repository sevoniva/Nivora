# Nivora

Nivora is an open-source DevOps delivery control plane for CI/CD, GitOps, multi-cloud deployment, artifact orchestration, policy gates, runners, release audit, and future visualization APIs.

Nivora is currently in Phase 0 / Phase 0.5 / Phase 0.6. Current focus is the backend skeleton, architecture guardrails, public planning, and contribution foundation.

Nivora is not production-ready. It does not implement full CI/CD execution, production GitOps reconciliation, cloud integrations, Kubernetes or Argo CD execution, vendor integrations, or a frontend yet.

For the project charter, see [PROJECT_CHARTER.md](PROJECT_CHARTER.md). For the full documentation index, see [docs/README.md](docs/README.md).

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

This repository uses a neutral default Go proxy in local tooling:

```sh
GOPROXY=https://proxy.golang.org,direct
```

Developers in China can override it per command:

```sh
GOPROXY=https://goproxy.cn,direct make build
```

Or export it for the shell:

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
make verify
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

## Verification

Run the full local verification suite:

```sh
make verify
```

This checks formatting, module tidiness, `go vet`, tests, binary builds, architecture boundaries, and high-confidence secret patterns.

## Roadmap

See [ROADMAP.md](ROADMAP.md).

Detailed roadmap docs live under [docs/roadmap/](docs/roadmap/overview.md). Large future proposals should use the [RFC process](docs/rfcs/README.md).

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). AI coding agents must follow [AGENTS.md](AGENTS.md), which is the canonical AI instruction file.

## License

Nivora is licensed under the Apache License 2.0.
