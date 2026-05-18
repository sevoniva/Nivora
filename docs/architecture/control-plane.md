# Control Plane

The Control Plane owns delivery intent and state. It should not directly execute jobs.

## Responsibilities

- HTTP and future gRPC APIs.
- Authentication and authorization wiring.
- Integration configuration.
- Workflow and state transitions.
- PipelineRun, Release, DeploymentRun, PolicyResult, Event, and AuditLog persistence.
- Policy gate coordination.
- Approval coordination.
- Runner registration and job assignment in future phases.

## Why the Server Should Not Execute Jobs

Execution can involve untrusted inputs, target credentials, network access, cancellation, logs, and environment-specific behavior. Keeping execution in Runners creates a clearer trust boundary and allows the Control Plane to remain focused on state, policy, and audit.

## Why State Belongs to the Control Plane

PipelineRuns, Releases, DeploymentRuns, approvals, policies, audit records, and events must be queryable and durable. Runners may disconnect or fail. The Control Plane is responsible for preserving authoritative state.

## Current State

Phase 0 includes health, readiness, version, system info, and structured `not_implemented` placeholder routes. Real orchestration is future work.

