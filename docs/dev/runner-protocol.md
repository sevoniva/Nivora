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
nivora runner groups create --name prod-runners --project-id project-a --environment-id env-prod --executor shell --max-concurrency 2 --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
nivora runner groups list --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
nivora runner groups get <runner-group-id> --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
nivora runner register --name local-runner --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
nivora runner register --name local-runner --group-id <runner-group-id> --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
export NIVORA_RUNNER_TOKEN='<token returned once by registration or rotation>'
nivora runner heartbeat --name local-runner --server http://localhost:8080 --token-env NIVORA_RUNNER_TOKEN
nivora runner claim --name local-runner --server http://localhost:8080 --token-env NIVORA_RUNNER_TOKEN
nivora runner logs append <job-run-id> --runner-id local-runner --pipeline-run-id <pipeline-run-id> --content "hello" --token-env NIVORA_RUNNER_TOKEN
nivora runner status update <job-run-id> --runner-id local-runner --status Running --token-env NIVORA_RUNNER_TOKEN
nivora runner token rotate local-runner --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
nivora runner token revoke local-runner --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

## API

- `GET /api/v1/runner-groups`
- `POST /api/v1/runner-groups`
- `GET /api/v1/runner-groups/{id}`
- `POST /api/v1/runners/register`
- `POST /api/v1/runners/{id}/token/rotate`
- `POST /api/v1/runners/{id}/token/revoke`
- `POST /api/v1/runners/{id}/heartbeat`
- `POST /api/v1/runners/{id}/jobs/claim`
- `POST /api/v1/runners/{id}/jobs/{job_id}/logs`
- `POST /api/v1/runners/{id}/jobs/{job_id}/status`
- `POST /api/v1/pipeline-runs/{id}/cancel-request`

## Current Limits

- Runner groups constrain registration and job claim by project, environment, executor allow-list, and aggregate concurrency, but they are still control-plane metadata guardrails rather than an OS sandbox.
- Runner tokens are hashed at rest and returned only once, but the broader beta-candidate runtime is still not production-ready.
- The worker loop is polling-based and intentionally simple.
- No Temporal, Tekton, Argo Workflows, NATS, or Redis dependency is introduced.
- Autoscaling and privileged execution are not implemented.
