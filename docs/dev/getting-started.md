# Developer Getting Started

Nivora is a hardened beta-candidate foundation and is not production-ready. The current runtime is a shell-only PipelineRun foundation for local development and tests.

## Quickstart (one command)

Start a single-process Nivora instance on `http://127.0.0.1:18091` using memory runtime, dev auth, and the built-in local runner. No postgres or docker required; ready in seconds:

```sh
./scripts/quickstart.sh
```

The script prints ready-to-run CLI examples. Stop with Ctrl-C or `kill $(cat .nivora/quickstart.pid)`. To use a different port: `NIVORA_QUICKSTART_PORT=19000 ./scripts/quickstart.sh`.

The commands below assume the quickstart server is running on `:18091`. Replace the port if you started on a different one.

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

## Store And Run A Pipeline Definition

With a running server, Pipeline definitions can be stored in the catalog and then used to create a PipelineRun:

```sh
go run ./cmd/nivora pipeline definition create --server http://127.0.0.1:18091 --project-id project-a --file examples/pipelines/simple-shell.yaml
go run ./cmd/nivora pipeline definition run <pipeline-id> --server http://127.0.0.1:18091
```

See [Pipeline Definition Catalog](pipeline-definitions.md) for the current API and CLI limits.

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
go run ./cmd/nivora pipeline get <pipeline-run-id> --server http://127.0.0.1:18091 --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora pipeline logs <pipeline-run-id> --server http://127.0.0.1:18091 --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora pipeline events <pipeline-run-id> --server http://127.0.0.1:18091
go run ./cmd/nivora pipeline timeline <pipeline-run-id> --server http://127.0.0.1:18091
```

## Current Limitations

- The default local development config uses memory stores. Use `database.runtime_store: postgres` for production-like persistence testing.
- Local CLI mode executes and prints a run but does not persist it across CLI invocations.
- No production Kubernetes, Argo CD, cloud, Git provider, or full artifact registry integrations are complete.
- The project is not production-ready.
