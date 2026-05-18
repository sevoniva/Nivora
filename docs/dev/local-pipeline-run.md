# Local PipelineRun Development

Phase 1 / 1.5 supports minimal shell-based PipelineRun execution. The goal is to make runtime behavior easy to inspect before adding durable persistence or remote runner protocols.

## Run Examples

```sh
go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml
go run ./cmd/nivora pipeline run --local examples/pipelines/failing-shell.yaml
go run ./cmd/nivora pipeline run --local examples/pipelines/retry-shell.yaml
go run ./cmd/nivora pipeline run --local examples/pipelines/timeout-shell.yaml
go run ./cmd/nivora pipeline run --local examples/pipelines/stderr-shell.yaml
```

The examples are safe and do not require credentials or external services.

## Smoke Test

```sh
make smoke-local
```

This runs the simple shell example and checks that the CLI reports a successful PipelineRun and printed logs.

## API Workflow

Start a server:

```sh
make run-server
```

Create a PipelineRun:

```sh
curl -X POST http://localhost:8080/api/v1/pipeline-runs \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Pipeline",
    "metadata": {"name": "hello-shell"},
    "spec": {
      "stages": [{
        "name": "build",
        "jobs": [{
          "name": "echo",
          "executor": "shell",
          "steps": [{"name": "say", "run": "printf hello"}]
        }]
      }]
    }
  }'
```

Inspect the run:

```sh
go run ./cmd/nivora pipeline get <pipeline-run-id> --server http://localhost:8080
go run ./cmd/nivora pipeline logs <pipeline-run-id> --server http://localhost:8080
go run ./cmd/nivora pipeline timeline <pipeline-run-id> --server http://localhost:8080
```

## Runner Commands

```sh
go run ./cmd/nivora runner list --server http://localhost:8080
go run ./cmd/nivora runner heartbeat --name local-runner --server http://localhost:8080
```

These commands use the server's in-memory runner records.
