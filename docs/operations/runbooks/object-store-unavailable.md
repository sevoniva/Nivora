# Runbook: Object Store Unavailable

Use this when manifest snapshots, evidence bundles, or future large log/artifact references cannot be read.

## Signals

- `/api/v1/system/diagnostics` reports degraded object store config.
- Deployment snapshot or evidence export paths fail.
- Object store errors appear in structured logs.

## Triage

```sh
curl http://localhost:8080/api/v1/system/diagnostics
```

Check the configured object store type and path or bucket. Do not print access keys.

## Recovery

1. Restore object store availability.
2. Restore object store data from backup when needed.
3. Confirm diagnostics return an `object_store` check.
4. Retry read-only inspection before replaying deployment or evidence workflows.

## Notes

Database restore alone is not enough for workflows that reference object store content.
