# Runner Design

The runner is separate from the control plane. It will register with the control plane, send heartbeats, receive assigned work, execute through executor adapters, and stream logs.

Phase 5.3 hardens the first runner protocol foundation:

- runner registration
- runner group metadata for project, environment, executor, and concurrency constraints
- runner identity and one-time token issuance
- token hash storage; raw tokens are never persisted
- token rotation
- token revocation
- runner heartbeat
- job claim with a lease expiration
- executor capability and label matching
- max concurrency checks
- job log append
- job status update
- cancel-request observation
- offline detection after missed heartbeat
- event outbox records for later reliable publication

This is still not a production remote runner protocol. It keeps HTTP payloads small, avoids exposing the entire domain model to runners, and does not enable privileged execution by default.

## Protocol Shape

```text
GET /api/v1/runner-groups
POST /api/v1/runner-groups
GET /api/v1/runner-groups/{id}
POST /api/v1/runners/register
POST /api/v1/runners/{id}/token/rotate
POST /api/v1/runners/{id}/token/revoke
POST /api/v1/runners/{id}/heartbeat
POST /api/v1/runners/{id}/jobs/claim
POST /api/v1/runners/{id}/jobs/{job_id}/logs
POST /api/v1/runners/{id}/jobs/{job_id}/status
POST /api/v1/pipeline-runs/{id}/cancel-request
```

`claim` returns a compact job lease with PipelineRun ID, StageRun ID, JobRun ID, StepRun IDs, executor name, commands, attempt, lease expiration, and cancel-request state.

Runner groups are control-plane metadata. They can constrain runner registration and claim by project, environment IDs, executor allow-list, and aggregate max concurrency. They are not an operating-system sandbox and do not replace isolated runner hosts.

Runner-owned mutation endpoints require a runner token through `Authorization: Bearer <token>` or `X-Nivora-Runner-Token`. Registration and token rotation are admin/RBAC operations. Privileged execution, autoscaling, container isolation, Kubernetes jobs, and remote host execution are not implemented by this runner protocol.

## Sandbox Limitations

The current runner protocol is not a production sandbox boundary. A runner should be treated as a trusted execution host for the project/environment it serves. Do not run untrusted pipelines on shared privileged runners. Before GA use, maintainers should define runner placement rules, workspace isolation, resource limits, secret-masking behavior, and incident response steps for compromised runner hosts.
