# Quotas And Usage

Phase 7.2 adds quota and usage read models for tenant scopes. These are intended to make limits visible and testable before Nivora grows a distributed quota system.

## View Quota

```bash
go run ./cmd/nivora quota view --scope-type project --scope-id demo
```

API:

```bash
curl "http://localhost:8080/api/v1/tenancy/quota?scopeType=project&scopeId=demo"
```

## Set Quota

```bash
go run ./cmd/nivora quota set --scope-type project --scope-id demo \
  --max-concurrent-pipeline-runs 10 \
  --max-concurrent-deployment-runs 5 \
  --max-runners 20 \
  --max-log-storage-bytes 1073741824
```

API:

```bash
curl -X POST "http://localhost:8080/api/v1/tenancy/quota" \
  -H "Content-Type: application/json" \
  -d '{"scopeType":"project","scopeId":"demo","maxConcurrentPipelineRuns":10,"maxConcurrentDeploymentRuns":5}'
```

Quota updates are metadata-only. They do not carry secrets or credentials. In production-like server mode this endpoint is permission-protected and should be called with a token that has the required project write permission.

## View Usage

```bash
go run ./cmd/nivora usage summary --scope-type project --scope-id demo
```

API:

```bash
curl "http://localhost:8080/api/v1/tenancy/usage?scopeType=project&scopeId=demo"
```

## Current Limits

The foundation includes default limits for:

- max concurrent PipelineRuns
- max concurrent DeploymentRuns
- max runners
- max artifacts tracked
- max log storage bytes
- API token requests per minute
- runner heartbeat requests per minute
- job claim requests per minute
- deployment concurrency
- pipeline concurrency

## Limitations

- Quota state is in-memory in the default local runtime.
- Quota state is persisted when the runtime is configured with the PostgreSQL-backed tenancy store.
- Rate limits are modeled but not enforced by a distributed limiter.
- Production tenant provisioning, persistent usage aggregation, and billing are future work.
- Nivora remains a hardened beta-candidate foundation and is not production-ready.
