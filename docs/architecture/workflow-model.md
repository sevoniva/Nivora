# Workflow Model

The workflow model describes how delivery state should progress. Phase 1 adds the first minimal in-memory PipelineRun execution loop for shell steps, and Phase 1.5 hardens that loop with explicit transitions, queued processing, retry, timeout, cancellation, ordered logs, events, and audit records.

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

Retry and cancellation now exist in minimal Phase 1.5 form for shell-based PipelineRuns. Pause and approval remain future use cases. State changes should produce events and audit records when they represent important lifecycle actions.

## WorkflowRuntime Abstraction

The WorkflowRuntime Port allows Nivora to start simple and later integrate a durable workflow engine if needed. Introducing a production workflow runtime requires an RFC.

## Phase 1 / 1.5 Local Runtime

Phase 1 accepts a minimal Pipeline definition, creates PipelineRun, StageRun, JobRun, and StepRun records, assigns the job to a local Runner, executes shell steps through the shell Executor, captures logs, emits events, and records audit entries.

Phase 1.5 adds a simple queued processing path used by the worker, minimal runner selection and heartbeat records, retry count support, timeout handling, cancellation for non-terminal runs, and timeline queries.

## Phase 3.6 Durable Runtime Foundation

Phase 3.6 keeps the default runtime in-memory but adds the protocol and state shapes needed for cross-process execution:

- JobRun claim leases to reduce double-claim risk.
- runner-owned log append and status update APIs.
- cancel-request state so workers and runners can observe cancellation before hard terminal transition.
- event outbox records for durable publication in a future database-backed runtime.
- a worker step that processes queued work and publishes pending outbox events.

PostgreSQL migrations now include the minimal columns and `event_outbox` table shape needed by future durable repositories. The default tests do not require PostgreSQL.

## Phase 5.2 Durable Workflow Recovery

Phase 5.2 makes the internal runtime recoverable without introducing Temporal, Tekton, Argo Workflows, Redis, or NATS. The worker now runs one reconciliation pass that can:

- acquire leases for queued PipelineRuns before execution;
- detect running PipelineRuns whose owner lease or update heartbeat has expired;
- return recoverable stale PipelineRuns and expired JobRun claims to `Queued`;
- reconcile cancel-requested PipelineRuns into `Canceled`;
- reconcile long-running stale PipelineRuns into `Timeout`;
- publish pending outbox events and record retry metadata for failed publish attempts.

The current durable recovery implementation is strongest for PipelineRun state because that is the first runtime path backed by the Phase 5.1 PostgreSQL repository. DeploymentRun and ReleaseExecution include lease fields so their state model is ready for the same pattern, but their full database-backed recovery loops remain future hardening work.
