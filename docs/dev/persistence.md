# Persistence Development

Phase 5.1 strengthens the persistence foundation without requiring PostgreSQL in unit tests.

## What To Work On First

Persistence priority:

1. PipelineRun / JobRun / LogChunk / Event / AuditLog / Runner
2. DeploymentRun / DeploymentPlan / resources / logs / events / audit
3. Release / ReleaseArtifact / ReleasePlan / ReleaseExecution
4. Org / project / application / environment / repository / release target catalog metadata and Pipeline definitions
5. Credential metadata / PolicyResult / approval and notification state

The first four priorities now have PostgreSQL repository foundations under `internal/adapters/repository/postgres`. Runtime stores keep aggregate snapshots as JSONB for compatibility with the current usecase models and maintain normalized query tables for status, logs, events, audit, resources, artifacts, target state, and recovery scans. Catalog stores persist operational metadata directly in catalog tables so server restarts do not discard org, project, environment, release target, repository, or Pipeline definition records when `database.runtime_store: postgres` is configured.

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
- migration presence and reversibility for catalog metadata and Pipeline definition tables
- claim-state behavior used by the PostgreSQL store
- status update behavior used by the PostgreSQL store
- optional PostgreSQL restart-style recovery for catalog metadata and Pipeline definitions
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

## Catalog Persistence

When `database.runtime_store: postgres` is selected, the server wires:

- `catalogusecase.Service` to `postgres.CatalogStore`
- `pipelineusecase.DefinitionCatalog` to `postgres.PipelineDefinitionStore`

The migration `000010_catalog_persistence` adds tables for orgs, projects, applications, environments, repositories, release targets, and Pipeline definitions. This closes a restart-loss gap for control-plane metadata. It does not make external Git provider integrations, artifact registry catalog persistence, policy catalog persistence, or tenant lifecycle automation complete.

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
