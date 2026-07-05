# Data Model

Nivora persists delivery state so workflow, audit, policy, and operational history survive process restarts and runner failures.

## Core Entities

- Org and User
- Project
- Application and Service
- Environment and ReleaseTarget
- Repository
- ArtifactRegistry
- Credential and SecretRef
- Pipeline, PipelineVersion, PipelineRun
- StageRun, JobRun, StepRun
- Release and release artifacts
- DeploymentRun and deployment steps
- Runner and RunnerGroup
- Approval
- Policy and PolicyResult
- EnvironmentLock
- Event
- AuditLog
- LogChunk
- Notification

## Lifecycle Records

PipelineRuns and DeploymentRuns are lifecycle records. They should preserve status, timestamps, related definitions, related targets, and execution context.

## Audit and Event Records

Audit records preserve accountability. Event records preserve lifecycle facts for internal workflows and future integrations.

## Artifact Records

Artifact records should prefer immutable digests. Release history should not depend on mutable tags.

## Runtime Persistence Foundation

Phase 5.1 started with a focused PostgreSQL persistence foundation for the PipelineRun runtime path. The current runtime hardening pass extends that direction to DeploymentRun, release artifact binding, ReleasePlan, and ReleaseExecution state while preserving the in-memory stores for local development and unit tests.

The runtime persistence tables use text IDs because runtime IDs are domain-generated values such as `prun-*`, `job-*`, and `runner-*`. The earlier relational skeleton remains useful for future normalized models, while the `runtime_*` tables provide a reversible foundation for current runtime state.

Catalog metadata now follows the same explicit-store approach. When `database.runtime_store: postgres` is configured, server route wiring uses PostgreSQL-backed stores for orgs, projects, applications, environments, repositories, release targets, Pipeline definitions, artifact registries, policies, and policy attachments. Local memory stores remain available for fast tests and development.

Persisted in Phase 5.1:

- PipelineRun snapshots in `runtime_pipeline_runs`
- flattened JobRun claim state in `runtime_job_runs`
- ordered LogChunks in `runtime_log_chunks`
- PipelineRun events in `runtime_events`
- audit records in `runtime_audit_logs`
- runner registration and heartbeat state in `runtime_runners`
- outbox records in `runtime_event_outbox`
- idempotency keys in `idempotency_keys`

Persisted by the deployment/release runtime hardening pass:

- DeploymentRun snapshots in `runtime_deployment_runs`
- DeploymentRun logs, events, and audit in `runtime_deployment_logs`, `runtime_deployment_events`, and `runtime_deployment_audit_logs`
- Deployment resource inventory in `runtime_deployment_resources`
- manifest snapshot metadata in `runtime_manifest_snapshots`
- rollback plan metadata in `runtime_rollback_plans`
- Release records and ReleaseArtifact bindings in `runtime_releases` and `runtime_release_artifacts`
- ReleasePlan records in `runtime_release_plans`
- ReleaseExecution records and target states in `runtime_release_executions` and `runtime_release_execution_targets`
- ReleaseExecution events and audit in `runtime_release_execution_events` and `runtime_release_execution_audit_logs`

Persisted by the catalog hardening pass:

- org metadata in `catalog_orgs`
- project metadata in `catalog_projects`
- application metadata in `catalog_applications`
- environment metadata in `catalog_environments`
- repository metadata and CredentialRef linkage in `catalog_repositories`
- release target metadata, safety flags, and CredentialRef linkage in `catalog_release_targets`
- repository snapshot records in `repository_snapshots`
- repository static intelligence records in `repository_intelligence`
- Pipeline definitions and version/hash metadata in `pipeline_definitions`
- artifact registry metadata and CredentialRef linkage in `catalog_artifact_registries`
- policy definitions in `catalog_policies`
- policy attachment scope links in `catalog_policy_attachments`

## Transaction Boundaries

The PostgreSQL PipelineStore uses explicit transactions for:

- saving a PipelineRun snapshot with flattened JobRun state
- claiming a job through `FOR UPDATE SKIP LOCKED`
- appending logs while preserving sequence order
- appending events and audit records while updating the run snapshot
- updating job status and cancel-request state

This is a foundation, not a complete distributed scheduler.

## Idempotency

The `idempotency_keys` table records request scope, key, resource type, resource ID, and request hash. Phase 5.1 exposes repository-level helpers so create paths can adopt idempotency without leaking HTTP details into domain models.

## Recovery Queries

The persistence adapter exposes recovery-friendly queries for:

- queued PipelineRuns
- stale running PipelineRuns
- assigned JobRuns whose leases expired
- pending outbox events

These queries are intended for worker recovery loops and future operational tooling.

## Remaining Work

DeploymentRun, ReleaseExecution, catalog metadata, Pipeline definitions, artifact registry metadata, and policy catalog state now have PostgreSQL-backed repository foundations when `database.runtime_store: postgres` is configured. Remaining persistence priorities include notification state, richer idempotency use at API boundaries, and broader integration tests against a real PostgreSQL instance.
