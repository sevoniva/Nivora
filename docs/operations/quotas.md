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

## Limitations

- Quota state is in-memory in the default local runtime.
- Rate limits are modeled but not enforced by a distributed limiter.
- Production tenant provisioning, persistent usage aggregation, and billing are future work.
- Nivora remains early-stage and not production-ready.
