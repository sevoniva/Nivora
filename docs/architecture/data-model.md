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

## Current State

Phase 0 includes an initial PostgreSQL migration with reasonable but intentionally minimal tables. Persistence use cases are not implemented yet.

