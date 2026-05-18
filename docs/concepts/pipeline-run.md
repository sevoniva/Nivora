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

## Common Confusion

A PipelineRun should not be treated as the same thing as a Release. A PipelineRun may build and validate an Artifact. A Release expresses delivery intent for an Artifact or version.
