# Runbook: Stuck PipelineRun

Use this when a PipelineRun remains queued or running longer than expected.

## Signals

- PipelineRun status is `Queued` or `Running`.
- Queue time metrics continue rising.
- Runtime recovery reports queued, stale, or timed-out PipelineRuns.

## Triage

```sh
curl http://localhost:8080/api/v1/system/runtime/recovery
nivora runtime status
```

Check the run:

```sh
nivora pipeline get <pipeline-run-id>
nivora pipeline events <pipeline-run-id>
nivora pipeline logs <pipeline-run-id>
```

## Recovery

1. Confirm a worker is running.
2. Confirm a compatible runner is online.
3. Reconcile runtime state:

```sh
nivora runtime reconcile
```

4. If cancellation was requested, verify the run transitions to `Canceled`.
5. If the run exceeded timeout, verify timeout reconciliation marks it failed or timed out.

## Escalation Notes

Do not manually edit runtime state unless you have a database backup and a maintainer-approved recovery plan.
