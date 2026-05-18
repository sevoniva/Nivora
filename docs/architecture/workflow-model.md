# Workflow Model

The workflow model describes how delivery state should progress. Phase 0 only defines initial structs, statuses, and package boundaries.

## Pipeline Lifecycle

- Pipeline: reusable definition.
- PipelineVersion: immutable version of a definition.
- PipelineRun: one execution of a PipelineVersion.
- StageRun: stage-level execution record.
- JobRun: job-level execution record.
- StepRun: step-level execution record.

## State Transitions

PipelineRuns may move through pending, queued, running, paused, succeeded, failed, canceled, or timed out states. JobRuns may be pending, assigned, running, succeeded, failed, skipped, retrying, or canceled.

## Retry, Pause, Approval, and Cancellation

Future use cases should model retry, pause, approval, and cancellation as explicit state transitions. They should produce events and audit records.

## WorkflowRuntime Abstraction

The WorkflowRuntime Port allows Nivora to start simple and later integrate a durable workflow engine if needed. Introducing a production workflow runtime requires an RFC.

