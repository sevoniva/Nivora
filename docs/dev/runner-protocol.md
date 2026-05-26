# Runner Protocol

Phase 5.3 introduces a token-authenticated HTTP runner protocol foundation. It is intended for local development and future hardening, not production-grade remote execution.

## Flow

```text
runner register
-> store one-time token securely outside the repo
-> heartbeat
-> claim job
-> append logs
-> update job status
-> observe cancel requested
```

## Commands

```sh
nivora runner register --name local-runner --server http://localhost:8080
export NIVORA_RUNNER_TOKEN='<token returned once by registration or rotation>'
nivora runner heartbeat --name local-runner --server http://localhost:8080 --token-env NIVORA_RUNNER_TOKEN
nivora runner claim --name local-runner --server http://localhost:8080 --token-env NIVORA_RUNNER_TOKEN
nivora runner logs append <job-run-id> --runner-id local-runner --pipeline-run-id <pipeline-run-id> --content "hello" --token-env NIVORA_RUNNER_TOKEN
nivora runner status update <job-run-id> --runner-id local-runner --status Running --token-env NIVORA_RUNNER_TOKEN
nivora runner token rotate local-runner --server http://localhost:8080
nivora runner token revoke local-runner --server http://localhost:8080
```

## API

- `POST /api/v1/runners/register`
- `POST /api/v1/runners/{id}/token/rotate`
- `POST /api/v1/runners/{id}/token/revoke`
- `POST /api/v1/runners/{id}/heartbeat`
- `POST /api/v1/runners/{id}/jobs/claim`
- `POST /api/v1/runners/{id}/jobs/{job_id}/logs`
- `POST /api/v1/runners/{id}/jobs/{job_id}/status`
- `POST /api/v1/pipeline-runs/{id}/cancel-request`

## Current Limits

- The default repository remains in-memory.
- Runner tokens are hashed at rest and returned only once, but the broader beta-candidate runtime is still not production-ready.
- The worker loop is polling-based and intentionally simple.
- No Temporal, Tekton, Argo Workflows, NATS, or Redis dependency is introduced.
- Autoscaling and privileged execution are not implemented.
