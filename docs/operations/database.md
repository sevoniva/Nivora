# Database Operations

Nivora uses PostgreSQL as the target source of truth. The alpha still runs many local demos with in-memory stores, but Phase 5.1 adds the first focused PostgreSQL runtime repository for PipelineRun state.

Nivora is not production-ready.

## Migrations

Migrations live under `internal/infra/migration/`.

Current migration groups:

- `000001_init`: initial relational skeleton for projects, applications, pipelines, releases, deployments, events, audit, logs, and related concepts.
- `000002_runtime_protocol`: event outbox and runner protocol fields on the initial skeleton.
- `000003_persistence_foundation`: alpha runtime tables for PipelineRun snapshots, JobRun claim state, ordered logs, events, audit, runners, outbox, and idempotency keys.
- `000004_runtime_recovery`: PipelineRun and DeploymentRun lease fields plus outbox retry metadata and recovery indexes.
- `000005_runner_fleet`: runner token metadata, capabilities, max concurrency, last seen time, and runner fleet indexes.
- `000006_performance_indexes`: query-shape indexes for list, log, event, audit, runner, lease, and outbox paths.

Run migrations with:

```sh
make migrate-up
```

Roll back the latest migration with:

```sh
make migrate-down
```

Set `DATABASE_URL` explicitly. Do not commit credentials.

Enable the PostgreSQL PipelineRun runtime store with:

```yaml
database:
  runtime_store: postgres
  url: "<set per environment>"
```

Local demos keep `runtime_store: memory` by default.

## Runtime Tables

The Phase 5.1 runtime tables are prefixed with `runtime_` and use text IDs to match existing alpha runtime identifiers:

- `runtime_pipeline_runs`
- `runtime_job_runs`
- `runtime_log_chunks`
- `runtime_events`
- `runtime_audit_logs`
- `runtime_runners`
- `runtime_event_outbox`
- `idempotency_keys`

## Operational Notes

- `runtime_event_outbox` is a persistence foundation. External broker publication remains future work.
- `runtime_pipeline_runs.owner_id`, `lease_expires_at`, `attempt`, and `heartbeat_at` support worker restart recovery.
- `runtime_event_outbox.retry_count`, `next_attempt_at`, and `last_error` support retriable event publication.
- `runtime_job_runs.lease_expires_at` supports recovery of assigned jobs whose runners stop heartbeating.
- `runtime_runners.token_hash` stores runner token hashes only; raw tokens are returned once by registration or rotation.
- `runtime_runners.max_concurrency` and `capabilities` support safer job claim decisions.
- `runtime_pipeline_runs.version`, `runtime_job_runs.version`, and `runtime_runners.version` provide optimistic-locking-friendly fields for later hardening.
- Log chunks are ordered by `(pipeline_run_id, sequence)`.
- Secret values must never be stored in runtime logs, events, audit records, or idempotency request hashes.

## Not Yet Complete

- The default local server still uses in-memory stores. Set `database.runtime_store: postgres` to use the Phase 5.1 PipelineRun store.
- DeploymentRun, Release, Artifact, Credential metadata, and PolicyResult PostgreSQL repositories remain future work.
- Phase 8.2 documents HA, backup, and restore procedures, but Nivora still does not automate them or claim production readiness.
