# Runbook: Database Unavailable

Use this when PostgreSQL cannot be reached or database configuration is degraded.

## Signals

- `/readyz` returns degraded dependency checks.
- Repository operations fail.
- Runtime recovery cannot list queued or stale work.

## Triage

```sh
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/system/diagnostics
```

Check database connectivity from the deployment environment. Do not print database passwords or full connection strings in incident notes.

## Recovery

1. Stop or pause workers to avoid repeated failures.
2. Restore database connectivity.
3. Verify migrations are at the expected version.
4. Start server and check readiness.
5. Start workers.
6. Run runtime reconciliation.

## Restore

Follow [Backup and Restore](../backup-restore.md). Restore PostgreSQL before replaying outbox or runtime recovery.
