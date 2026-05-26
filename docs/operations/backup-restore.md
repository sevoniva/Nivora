# Backup and Restore

Nivora Phase 8.2 documents backup and restore procedures for production-direction operations. Nivora is a hardened beta-candidate foundation and is not production-ready; operators remain responsible for backup scheduling, encryption, retention, and restore drills.

## What to Back Up

| Data | Why it matters | Suggested backup |
| --- | --- | --- |
| PostgreSQL | Runtime records, logs, events, audit, runners, outbox, releases, artifacts when persisted. | Native PostgreSQL backups such as `pg_dump`, snapshots, or managed database backup. |
| Object store | Manifest snapshots, future large evidence artifacts, future large logs or release assets. | Bucket or filesystem snapshot with versioning where available. |
| Config files | Server, worker, runner, Helm values, Docker Compose environment. | Git-controlled sanitized config plus secure external secret storage. |
| Secret metadata | Credential and SecretRef metadata. | Database backup; secret values stay in the configured secret provider. |
| Secret values | External credentials and tokens. | Back up through the secret provider, not Nivora APIs. |

## Database

Example logical backup:

```sh
pg_dump "$NIVORA_DATABASE_URL" > nivora-$(date +%Y%m%d%H%M%S).sql
```

Example restore into an empty database:

```sh
psql "$NIVORA_DATABASE_URL" < nivora-backup.sql
```

After restore:

1. Run migrations.
2. Start the server.
3. Check `/readyz`.
4. Check `/api/v1/system/runtime/recovery`.
5. Run one reconciliation pass if needed.

Runtime audit records for all 9 stores are stored in PostgreSQL when `database.runtime_store: postgres`. Compliance evidence bundle and retention policy persistence have PostgreSQL foundations through `compliance_evidence_bundles` and `compliance_retention_policies`. Tamper-evident SHA-256 hash-chain audit writes are implemented across all audit-producing stores with verification via `GET /api/v1/audit/verify`.

## Object Store

For local object store paths, snapshot the configured `object_store.path`:

```sh
tar -czf nivora-objectstore.tgz .nivora/objectstore
```

Restore before replaying deployment or evidence workflows that reference object store entries.

For S3-compatible stores, use the provider's bucket replication, lifecycle, and object lock features. Do not store access keys in backup scripts.

## Config

Back up:

- `configs/server.yaml`
- `configs/worker.yaml`
- `configs/runner.yaml`
- deployment-specific Helm values
- Docker Compose overrides

Do not back up raw secret values in config files. Use environment variables, SecretRef, CredentialRef, or the external secret provider.

## Restore Checklist

1. Restore sanitized config.
2. Restore secret provider values using that provider's process.
3. Restore PostgreSQL.
4. Restore object store data.
5. Run migrations.
6. Start `nivora-server`.
7. Confirm `/readyz` and diagnostics.
8. Start workers.
9. Reconcile pending outbox/runtime state.
10. Start runners and verify heartbeat.

## Backup/Restore Smoke Test

A backup/restore drill smoke test validates the pg_dump → restore pipeline:

```bash
DATABASE_URL="postgres://..." make smoke-backup-restore
```

The script (`scripts/smoke-backup-restore-postgres.sh`):
1. Validates migration pairs are reversible
2. Starts a server, inserts test PipelineRun
3. Stops the server
4. Runs pg_dump (if available)
5. Restarts the server
6. Verifies PipelineRun and audit records survived

Skip with `SKIP_BACKUP_RESTORE=1` or if PostgreSQL is unavailable.

## Migration Drill

For a disposable database or staging environment:

```sh
NIVORA_RUN_POSTGRES_INTEGRATION=true DATABASE_URL="$NIVORA_DATABASE_URL" make test-postgres-integration
./scripts/smoke-audit-durability.sh
make smoke-backup-restore
```

The baseline unit test suite does not require PostgreSQL. Real database migration, recovery, and backup/restore checks are opt-in so local CI remains self-contained.

## Event Outbox Recovery

Pending or failed event outbox records should be preserved during backup. After dependencies recover, run:

```sh
nivora runtime reconcile
```

or:

```sh
curl -X POST http://localhost:8080/api/v1/system/runtime/reconcile
```

This phase provides the foundation; external event transport replay remains future hardening.

## Tested vs Untested

Tested in baseline verification:

- config validation rejects unsafe production defaults
- evidence bundle and retention policy store behavior
- production profile static safety checks

Tested only when optional PostgreSQL integration is enabled:

- migration up/down against a real PostgreSQL schema
- PipelineRun, DeploymentRun, ReleaseExecution, runner claim, outbox, and compliance evidence recovery after reconnect

Not yet production-proven:

- automated restore from a production backup in a live environment
- object-store restore with large evidence payloads
- backup/restore drill automated in CI (requires PostgreSQL; runs as opt-in locally)
