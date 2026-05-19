# Persistence Development

Phase 5.1 strengthens the persistence foundation without requiring PostgreSQL in unit tests.

## What To Work On First

Persistence priority:

1. PipelineRun / JobRun / LogChunk / Event / AuditLog / Runner
2. DeploymentRun / Release / Artifact
3. Credential metadata / PolicyResult / approval and notification state

The first priority is implemented as a PostgreSQL PipelineStore under `internal/adapters/repository/postgres`.

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
- claim-state behavior used by the PostgreSQL store
- status update behavior used by the PostgreSQL store

Future integration tests may use Docker or testcontainers only if the project explicitly accepts that dependency and keeps those tests optional for default CI.

## Recovery-Oriented Queries

The PostgreSQL store exposes:

- `ListQueuedPipelineRuns`
- `ListStaleRunningPipelineRuns`
- `ListExpiredJobClaims`
- `ListPendingOutbox`

These support future worker recovery loops and operational diagnostics.
