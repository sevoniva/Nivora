# Runner Protocol

Phase 3.6 introduces a small HTTP runner protocol foundation. It is intended for local development and future hardening, not production-grade remote execution.

## Flow

```text
runner register
-> heartbeat
-> claim job
-> append logs
-> update job status
-> observe cancel requested
```

## Commands

```sh
nivora runner register --name local-runner --server http://localhost:8080
nivora runner heartbeat --name local-runner --server http://localhost:8080
nivora runner claim --name local-runner --server http://localhost:8080
nivora runner logs append <job-run-id> --pipeline-run-id <pipeline-run-id> --content "hello"
nivora runner status update <job-run-id> --status Running
```

## API

- `POST /api/v1/runners/register`
- `POST /api/v1/runners/{id}/heartbeat`
- `POST /api/v1/runners/{id}/jobs/claim`
- `POST /api/v1/jobs/{id}/logs`
- `POST /api/v1/jobs/{id}/status`
- `POST /api/v1/pipeline-runs/{id}/cancel-request`

## Current Limits

- The default repository remains in-memory.
- PostgreSQL migrations define the future durable shape, but full DB repositories are not wired by default.
- Runner auth is still local/dev oriented.
- The worker loop is polling-based and intentionally simple.
- No Temporal, Tekton, Argo Workflows, NATS, or Redis dependency is introduced.
