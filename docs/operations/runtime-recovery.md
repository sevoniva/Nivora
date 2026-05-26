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

## Multi-Process Recovery Smoke

A controlled end-to-end smoke test verifies server + worker + runner + PostgreSQL can recover state across process restart.

```bash
# Run with a local Postgres instance
DATABASE_URL="postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable" \
  make smoke-multiprocess-recovery

# Skip if Postgres unavailable
SKIP_MULTIPROCESS_RECOVERY=1 make smoke-multiprocess-recovery
```

The script (`scripts/smoke-multiprocess-recovery-postgres.sh`):
1. Starts server, worker, and runner with `runtime_store: postgres`
2. Creates a PipelineRun via the API
3. Stops all processes, restarts them
4. Verifies PipelineRun state, logs, events, and audit records survived
5. Creates a DeploymentRun after restart
6. Performs a second restart and verifies both PipelineRun and DeploymentRun

This test runs in CI as part of the `postgres-integration` job.

## Operational Notes

- The default local runtime remains in-memory.
- PostgreSQL-backed runtime recovery is available when `database.runtime_store: postgres` is configured.
- Optional PostgreSQL integration tests now verify restart-style repository recovery for PipelineRun, DeploymentRun, ReleasePlan, ReleaseExecution, runner claim leases, and event outbox records. See `docs/dev/runtime-recovery-tests.md`.
- The multi-process recovery smoke test proves cross-process restart durability.
- Cancellation is reconciled for queued/running PipelineRuns; executor-level interruption remains limited by the current runner/executor implementation.
- Timeout reconciliation is based on stale update time and lease state.
- Runner offline detection is part of reconciliation and uses heartbeat age.
- DeploymentRun and ReleaseExecution persistence survives repository/service restart in PostgreSQL mode; complete multi-process worker orchestration remains future hardening.

Nivora is a hardened beta-candidate foundation and is not production-ready.
