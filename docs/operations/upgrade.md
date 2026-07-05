# Upgrade Guide

This guide defines the release-candidate upgrade review path. It is not an automated upgrader and does not claim zero-downtime upgrades.

## Before Upgrading

1. Read `CHANGELOG.md`.
2. Read the target release checklist under `docs/releases/`.
3. Back up PostgreSQL, object-store data, and sanitized config.
4. Back up secret values through the configured secret provider.
5. Confirm migration up/down files exist for every new migration.
6. Run `make verify` on the candidate revision.
7. Review OpenAPI and AsyncAPI changes for breaking behavior.

## Versioning

`VERSION` remains the source for the built binary version string. Do not update it to an RC value until the maintainer is cutting the RC.

For `v1.0.0-rc.1`, the release cut should include:

```sh
printf '1.0.0-rc.1\n' > VERSION
```

Only do this as part of the release commit or tag flow, not during ordinary hardening work.

## Database Migration Flow

Use an isolated database first:

```sh
DATABASE_URL='postgres://...' make migrate-up
DATABASE_URL='postgres://...' make migrate-down
DATABASE_URL='postgres://...' make migrate-up
```

For the release-to-release compatibility smoke used by the CI PostgreSQL profile:

```sh
DATABASE_URL='postgres://...' make smoke-upgrade-migration-compatibility
```

This smoke check validates `VERSION` to Helm `appVersion` alignment, reversible migration file pairs, migration up/down execution, Postgres runtime bootstrap, and representative PipelineRun, DeploymentRun, ReleaseExecution, and evidence-bundle recovery paths. It skips with a clear reason when PostgreSQL client tools or a disposable database are not available locally.

For non-disposable data:

1. Take a backup.
2. Stop workers before schema changes.
3. Run migrations.
4. Start the server.
5. Check `/readyz`.
6. Start workers.
7. Run runtime reconciliation if needed.
8. Start runners and verify heartbeat.

## Runtime Recovery After Upgrade

Check recovery state:

```sh
nivora runtime status --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
curl http://localhost:8080/api/v1/system/runtime/recovery
```

Run one reconciliation pass after an upgrade if queued, stale, canceled, or timeout-candidate work is present:

```sh
nivora runtime reconcile --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

## Rollback Direction

Rollback means restoring the previous application version and compatible data state. Because database down migrations can lose data if used carelessly, follow this order:

1. Stop workers and runners.
2. Restore application binaries or images.
3. Restore database from backup when schema rollback is not safe.
4. Restore object-store data if needed.
5. Start server and verify diagnostics.
6. Start workers and runners.

Do not rely on down migrations as the only rollback mechanism for non-disposable data.

## Compatibility Review

Before declaring an upgrade path acceptable for RC:

- APIs changed intentionally and are documented.
- Events changed intentionally and are documented.
- Config defaults are safe.
- Secrets remain externalized.
- Guarded operations remain opt-in.
- Existing examples still validate.
- `make smoke-upgrade-migration-compatibility` has passed against a disposable PostgreSQL database or a skip reason is recorded for local-only review.
