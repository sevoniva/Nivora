# Persistence Development

Phase 5.1 strengthens the persistence foundation without requiring PostgreSQL in unit tests.

## What To Work On First

Persistence priority:

1. PipelineRun / JobRun / LogChunk / Event / AuditLog / Runner
2. DeploymentRun / DeploymentPlan / resources / logs / events / audit
3. Release / ReleaseArtifact / ReleasePlan / ReleaseExecution
4. Credential metadata / PolicyResult / approval and notification state

The first three priorities now have PostgreSQL repository foundations under `internal/adapters/repository/postgres`. They store aggregate snapshots as JSONB for compatibility with the current usecase models and maintain normalized query tables for status, logs, events, audit, resources, artifacts, target state, and recovery scans.

## Design Rules

- Keep SQL explicit.
- Keep domain packages free of `pgx`, SQL, transactions, and database tags.
- Keep usecases dependent on repository interfaces.
- Keep in-memory stores for fast unit tests.
- Make migrations reversible.
- Add indexes for status, run IDs, leases, outbox status, and timestamps.
- Prefer repository-level idempotency over HTTP-specific behavior in domain code.

## Testing

Default tests must not require an external database.

Current tests validate:

- migration presence and reversibility for the Phase 5.1 tables
- migration presence and reversibility for DeploymentRun and ReleaseExecution runtime tables
- claim-state behavior used by the PostgreSQL store
- status update behavior used by the PostgreSQL store
- production config rejects `database.runtime_store: memory`

Future integration tests may use Docker or testcontainers only if the project explicitly accepts that dependency and keeps those tests optional for default CI.

## Recovery-Oriented Queries

The PostgreSQL store exposes:

- `ListQueuedPipelineRuns`
- `ListStaleRunningPipelineRuns`
- `ListExpiredJobClaims`
- `ListPendingOutbox`
- `ListNonTerminalDeploymentRuns`
- `ListStaleDeploymentRuns`
- `ListNonTerminalReleaseExecutions`
- `ListStaleReleaseExecutions`

These support future worker recovery loops and operational diagnostics.

## Runtime Store Selection

Local development and unit tests may continue using:

```yaml
database:
  runtime_store: memory
```

Production-like environments should use:

```yaml
database:
  runtime_store: postgres
  url: "<set per environment>"
```

The config validator rejects `runtime_store: memory` when `environment` is `production` or `prod`. This prevents accidental production use of process-local state, but it does not make the runtime production-ready by itself.
