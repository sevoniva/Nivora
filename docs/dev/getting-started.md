# Developer Getting Started

Nivora is early-stage and not production-ready. The current runtime is a shell-only PipelineRun foundation for local development and tests.

## Prerequisites

- Go
- Make
- curl for API smoke tests

Docker and PostgreSQL are useful for later local development, but they are not required for the Phase 1 / 1.5 runtime tests.

## Verify the Repository

```sh
make verify
```

This runs formatting checks, module tidy checks, `go vet`, tests, binary builds, architecture checks, secret checks, and the local PipelineRun smoke test.

## Run a Local Pipeline

```sh
go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml
```

The command prints the PipelineRun ID, final status, duration, log count, and captured logs.

## Run the API Smoke Test

```sh
make smoke-api
```

The script starts `nivora-server` on a temporary local port, creates a shell PipelineRun, verifies logs and timeline access, and stops the server.

## Start Services Manually

```sh
make run-server
make run-worker
make run-runner
```

The server exposes HTTP APIs. The worker currently advances queued runs in the in-memory runtime mode available to its process. Durable cross-process runtime persistence is future work.

## Inspect Runtime State

Server-backed CLI inspection commands require a running server:

```sh
go run ./cmd/nivora pipeline get <pipeline-run-id> --server http://localhost:8080
go run ./cmd/nivora pipeline logs <pipeline-run-id> --server http://localhost:8080
go run ./cmd/nivora pipeline events <pipeline-run-id> --server http://localhost:8080
go run ./cmd/nivora pipeline timeline <pipeline-run-id> --server http://localhost:8080
```

## Current Limitations

- Runtime state is in memory.
- Local CLI mode executes and prints a run but does not persist it across CLI invocations.
- No Kubernetes, Argo CD, cloud, Git provider, or artifact registry integrations are implemented in this phase.
- The project is not production-ready.
