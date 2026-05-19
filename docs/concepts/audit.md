# Audit

Audit is the durable record of important delivery actions.

## Why It Exists

Delivery systems affect production systems and sensitive credentials. Nivora must preserve who did what, when, to which target, with which Artifact, under which Policy and approval context.

## Relationships

- Records important PipelineRun, Release, DeploymentRun, Policy, approval, credential, and runner actions.
- Must not contain secret values.
- Should correlate with events and logs.

## Phase 1.5 Behavior

Phase 1.5 creates in-memory AuditLog records for important PipelineRun lifecycle actions: created, queued, started, completed, failed, and canceled. Heartbeats are emitted as events but are not treated as high-value audit records by default because they may become high-frequency.

Phase 7.3 adds structured audit fields for actor, action, subject, scope, reason, request ID, correlation ID, and safe metadata. It also adds audit search, evidence bundle export, and retention policy metadata.

Durable external audit storage and retention enforcement jobs are future work.

## Common Confusion

Audit is not optional logging. Logs explain behavior; audit records preserve accountability.
