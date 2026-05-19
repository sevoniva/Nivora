# Runtime Recovery Operations

Phase 5.2 adds recovery endpoints and CLI commands for the internal runtime foundation.

## Inspect Recovery State

```bash
nivora runtime status --server http://localhost:8080
curl http://localhost:8080/api/v1/system/runtime/recovery
```

The response reports queued PipelineRuns, stale running runs, expired job claims, cancel requests, timeout candidates, and outbox retry state.

## Run Reconciliation

```bash
nivora runtime reconcile --server http://localhost:8080
curl -X POST http://localhost:8080/api/v1/system/runtime/reconcile
```

The worker also runs reconciliation as its runtime advancement step.

## Operational Notes

- The default local runtime remains in-memory.
- PostgreSQL-backed runtime recovery is available when `database.runtime_store: postgres` is configured.
- Optional PostgreSQL integration tests now verify restart-style repository recovery for PipelineRun, DeploymentRun, ReleasePlan, ReleaseExecution, runner claim leases, and event outbox records. See `docs/dev/runtime-recovery-tests.md`.
- Cancellation is reconciled for queued/running PipelineRuns; executor-level interruption remains limited by the current runner/executor implementation.
- Timeout reconciliation is based on stale update time and lease state.
- Runner offline detection is part of reconciliation and uses heartbeat age.
- DeploymentRun and ReleaseExecution persistence survives repository/service restart in PostgreSQL mode; complete multi-process worker orchestration remains future hardening.

Nivora is still early-stage and not production-ready.
