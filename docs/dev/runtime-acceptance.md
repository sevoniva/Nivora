# Runtime Acceptance Matrix

This matrix documents the Phase 1 / 1.5 shell-based PipelineRun runtime. It is a contributor checklist, not a production certification.

| Scenario | Verifies | Expected Status | Expected Logs / Events / Audit | Automated | Example or Command |
|---|---|---|---|---|---|
| Successful shell pipeline | Pipeline spec parsing, run creation, local runner execution, stdout capture | `Succeeded` | stdout LogChunk, created/queued/started/completed events, lifecycle audit records | Yes | `make smoke-local`, `examples/pipelines/simple-shell.yaml` |
| Failing shell pipeline | non-zero executor result and failure reason | `Failed` | stderr LogChunk, failed event, failed audit record | Yes | `examples/pipelines/failing-shell.yaml` |
| Retry pipeline | job retry transition and second attempt | `Succeeded` when retry succeeds | retrying job event and multiple attempts | Yes | `examples/pipelines/retry-shell.yaml` |
| Timeout pipeline | timeout propagation from shell executor | `Timeout` | timeout failure reason and failed PipelineRun event | Yes | `examples/pipelines/timeout-shell.yaml` |
| Cancellation before running | queued run cancellation | `Canceled` | canceled event and audit record | Yes | usecase test |
| Cancellation while running | future distributed cancellation path | Not fully supported | current local runtime cannot interrupt cross-process work | Partial | `examples/pipelines/cancel-shell.yaml` documents intent |
| stdout log capture | ordered stdout LogChunk | `Succeeded` | `stream=stdout`, increasing sequence | Yes | `examples/pipelines/simple-shell.yaml` |
| stderr log capture | ordered stderr LogChunk | `Succeeded` or `Failed` depending command | `stream=stderr`, increasing sequence | Yes | `examples/pipelines/stderr-shell.yaml` |
| Timeline ordering | event query sorted by time | n/a | created before completed | Yes | `GET /api/v1/pipeline-runs/{id}/timeline` |
| Audit record creation | lifecycle accountability | n/a | created/queued/started/final audit records | Yes | usecase test |
| Event emission | CloudEvents-style runtime events | n/a | pipeline and job lifecycle events | Yes | usecase and API route tests |
| Runner registration | runner records and defaults | n/a | runner registered event | Yes | `POST /api/v1/runners/register` |
| Runner heartbeat | heartbeat timestamp update | n/a | runner heartbeat event | Yes | `nivora runner heartbeat --name local-runner --server http://localhost:8080` |
| Invalid pipeline spec | validation rejects bad definitions | n/a | clear validation error | Yes | spec validation tests |
| Invalid state transition | transition helpers reject impossible movement | n/a | explicit error | Yes | state transition tests |

## Notes

- Normal tests do not require Kubernetes, Argo CD, Harbor, Nexus, GitLab, Gitea, or external network access.
- The current runtime uses in-memory repositories. Restarting a process clears runs, logs, events, audit records, and runner records.
- Phase 1.6 does not add real deployment integrations.
