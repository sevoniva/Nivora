# Multi-Tenancy Model

Phase 7.2 adds the first multi-tenancy, quota, and isolation foundation for Nivora.

## Tenant Scopes

Nivora uses explicit scopes:

- `org`
- `project`
- `application`
- `environment`
- `global`

Organizations contain projects. Projects contain applications and environments. Credentials, runners, releases, deployments, artifacts, audit records, and future integrations should carry the narrowest useful scope.

## Isolation Rules

- Scoped API tokens carry `scopeType` and `scopeId`.
- Permission checks deny access when a scoped subject targets a different scoped resource.
- Credential list/get operations respect the caller scope.
- Runner groups carry project and environment boundaries for scheduling and administration.
- Audit records should preserve enough scope metadata for future tenant-specific audit views.

This is a backend foundation, not a complete enterprise tenant system.

## Quotas

The quota model tracks limits for:

- concurrent PipelineRuns
- concurrent DeploymentRuns
- runners
- artifacts tracked
- log storage bytes
- API token request rate
- runner heartbeat rate
- job claim rate

The current implementation exposes quota get/set operations and usage summaries through API/CLI and provides use-case checks for quota enforcement. PostgreSQL-backed runtime mode can persist quota metadata. Distributed counters, production rate limiting, and billing remain future work.

## Non-Goals

- No billing model.
- No hard multi-cluster isolation.
- No distributed rate limiter.
- No production tenant provisioning workflow.
- No claim of production readiness.
