# Runner Design

The runner is separate from the control plane. It will register with the control plane, send heartbeats, receive assigned work, execute through executor adapters, and stream logs.

Phase 3.6 adds the first runner protocol foundation:

- runner registration
- runner heartbeat
- job claim with a lease expiration
- job log append
- job status update
- cancel-request observation
- event outbox records for later reliable publication

This is still not a production remote runner protocol. It keeps HTTP payloads small and avoids exposing the entire domain model to runners.

## Protocol Shape

```text
POST /api/v1/runners/register
POST /api/v1/runners/{id}/heartbeat
POST /api/v1/runners/{id}/jobs/claim
POST /api/v1/jobs/{id}/logs
POST /api/v1/jobs/{id}/status
POST /api/v1/pipeline-runs/{id}/cancel-request
```

`claim` returns a compact job lease with PipelineRun ID, StageRun ID, JobRun ID, StepRun IDs, executor name, commands, attempt, lease expiration, and cancel-request state.

Privileged execution, container isolation, Kubernetes jobs, remote host execution, and production-grade remote runner authentication are not implemented yet.
