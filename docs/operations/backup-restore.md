# Backup and Restore

Nivora Phase 8.2 documents backup and restore procedures for production-direction operations. Nivora is still early-stage and not production-ready; operators remain responsible for backup scheduling, encryption, retention, and restore drills.

## What to Back Up

| Data | Why it matters | Suggested backup |
| --- | --- | --- |
| PostgreSQL | Runtime records, logs, events, audit, runners, outbox, releases, artifacts when persisted. | Native PostgreSQL backups such as `pg_dump`, snapshots, or managed database backup. |
| Object store | Manifest snapshots, evidence bundle artifacts, future large logs or release assets. | Bucket or filesystem snapshot with versioning where available. |
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
