# Runtime Recovery Integration Tests

Nivora has optional PostgreSQL integration tests that exercise durable runtime recovery paths against a real database. These tests are not part of the default unit test suite because they require PostgreSQL, but they are the preferred evidence for production-like persistence behavior.

## What The Tests Cover

The recovery integration suite applies migrations to a disposable PostgreSQL schema, writes runtime state, discards the repository/service instances, reconnects to the same database, and verifies that the state can be loaded after the simulated restart.

Covered paths:

- PipelineRun, StageRun, JobRun, StepRun, logs, events, audit, queued recovery, stale lease queries, and cancellation state.
- DeploymentRun, DeploymentPlan, resource inventory, manifest snapshot metadata, rollback plan, logs, events, audit, non-terminal queries, and stale run queries.
- Release, ReleaseArtifact, ReleasePlan, ReleaseExecution, target execution records, release execution events, release execution audit, non-terminal queries, and stale execution queries.
- Runner registration, runner job claim lease persistence, expired lease detection, and reclaim behavior after lease expiry.
- Event outbox pending, published, failed, retry, and idempotent state recovery.
- Runtime bootstrap wiring that selects PostgreSQL runtime stores when `database.runtime_store: postgres` is configured.

These tests do not require Kubernetes, Argo CD, Harbor, Nexus, Git providers, cloud providers, external registries, or scanners.

## Running The Tests

Set `DATABASE_URL` to a PostgreSQL database where the test user can create and drop schemas, then opt in with `NIVORA_RUN_POSTGRES_INTEGRATION=true`.

```bash
export DATABASE_URL='postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable'
export NIVORA_RUN_POSTGRES_INTEGRATION=true
make test-postgres-integration
```

Equivalent script:

```bash
NIVORA_RUN_POSTGRES_INTEGRATION=true DATABASE_URL="$DATABASE_URL" ./scripts/smoke-runtime-recovery-postgres.sh
```

Without `NIVORA_RUN_POSTGRES_INTEGRATION=true`, the Makefile target and smoke script skip successfully. This keeps normal unit tests and `make verify` self-contained.

## Test Isolation

Each integration test creates a unique schema and sets PostgreSQL `search_path` through the connection URL. The schema is dropped during cleanup. The tests do not require a developer-specific database name and should not store credentials in the repository.

## Multi-Process Recovery Smoke

A separate smoke test (`scripts/smoke-multiprocess-recovery-postgres.sh`) proves server + worker + runner + PostgreSQL can recover across process restarts:

```bash
DATABASE_URL="postgres://..." make smoke-multiprocess-recovery
```

The script:
1. Starts server, worker, and runner with `runtime_store: postgres`
2. Creates PipelineRun via API, verifies logs/events/audit
3. Stops all processes, restarts them
4. Verifies PipelineRun state survived restart
5. Creates DeploymentRun and ReleaseExecution
6. Stops and restarts again
7. Verifies all three workloads survived

This complements the repository-level integration tests by proving cross-process durability.

## Current Limitations

- The integration suite proves durable repository recovery and bootstrap, not multi-process orchestration (covered by the multi-process smoke).
- Runner claim/lease recovery across server restart is tested at the repository level; cross-process claim behavior is not yet smoke-tested.
- Cancellation recovery verifies persisted state and recovery queries; executor interruption remains best-effort.
- Timeout and lease tests use deterministic timestamps and repository queries; they do not run long sleep-based workers.
- Memory stores remain available for local development only. Production-like runtime validation should use PostgreSQL mode.
