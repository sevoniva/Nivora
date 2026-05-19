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

Phase 5.1 adds a focused PostgreSQL persistence foundation for the PipelineRun runtime path. It intentionally strengthens the most recovery-sensitive part of the system first instead of rewriting every model at once.

The runtime persistence tables use text IDs because alpha runtime IDs are domain-generated values such as `prun-*`, `job-*`, and `runner-*`. The earlier relational skeleton remains useful for future normalized models, while the `runtime_*` tables provide a reversible foundation for current runtime state.

Persisted in Phase 5.1:

- PipelineRun snapshots in `runtime_pipeline_runs`
- flattened JobRun claim state in `runtime_job_runs`
- ordered LogChunks in `runtime_log_chunks`
- PipelineRun events in `runtime_events`
- audit records in `runtime_audit_logs`
- runner registration and heartbeat state in `runtime_runners`
- outbox records in `runtime_event_outbox`
- idempotency keys in `idempotency_keys`

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

DeploymentRun, Release, Artifact, Credential metadata, and PolicyResult already have domain/usecase foundations, but PostgreSQL repositories for those areas remain a priority after the PipelineRun runtime store is wired into application bootstrap.
