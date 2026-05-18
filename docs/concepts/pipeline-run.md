# PipelineRun

A PipelineRun is one execution of a Pipeline.

## Why It Exists

PipelineRuns preserve execution state: pending, queued, running, paused, succeeded, failed, canceled, or timed out.

## Relationships

- References a Pipeline and PipelineVersion.
- Contains StageRuns, JobRuns, and StepRuns.
- May be assigned to Runners.
- May produce Artifacts.
- Should produce audit and event records.

## Phase 1 Behavior

Phase 1 can create and execute a PipelineRun from a minimal Pipeline definition. The current runtime is in-memory and shell-only. It records status transitions, captured logs, CloudEvents-style events, and AuditLogs.

## Phase 1.5 Behavior

Phase 1.5 persists the minimal runtime state in in-memory repositories for the lifetime of the process. A PipelineRun is created, queued, advanced to running by the runtime service or worker path, and finished as succeeded, failed, canceled, or timed out.

The Phase 1.5 model also stores ordered LogChunks, emits timeline events, records audit entries for important lifecycle actions, supports a minimal job retry count, and supports cancellation for non-terminal runs.

This does not make the runtime production-ready. PostgreSQL runtime persistence, distributed scheduling, remote runner protocols, and deployment integrations are future work.

## Common Confusion

A PipelineRun should not be treated as the same thing as a Release. A PipelineRun may build and validate an Artifact. A Release expresses delivery intent for an Artifact or version.
