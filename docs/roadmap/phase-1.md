# Phase 1: Minimal Pipeline Execution

## Objective

Implement the smallest useful PipelineRun execution path.

## Scope

- Minimal PipelineRun persistence.
- Runner assignment.
- Shell Executor execution path.
- Log streaming.
- Status transitions.
- Audit event.
- Minimal API support.

## Non-Goals

- Production workflow runtime.
- Real cloud integrations.
- Kubernetes or Argo CD production integration.
- Full policy engine.
- Frontend.

## Expected Deliverables

A basic flow from PipelineRun creation to controlled local runner execution, logs, status, events, and audit.

## Current Phase 1 Implementation

The first Phase 1 implementation supports minimal shell-based PipelineRun execution only:

- parse a small Pipeline YAML definition
- create PipelineRun, StageRun, JobRun, and StepRun records in memory
- assign work to a Phase 1 local Runner
- execute shell steps through the shell Executor
- capture stdout and stderr logs
- transition status to Succeeded or Failed
- emit CloudEvents-style PipelineRun events through the in-memory EventBus
- create in-memory AuditLog records
- expose minimal HTTP and CLI entry points

## Phase 1.5 Runtime Foundation

Phase 1.5 hardens this loop without adding Phase 2 integrations:

- explicit PipelineRun, StageRun, JobRun, and StepRun transition helpers
- queued PipelineRuns that can be advanced by the worker in the current in-memory runtime mode
- minimal runner registration, selection, and heartbeat records
- ordered LogChunk capture for stdout and stderr
- timeline APIs backed by stored runtime events
- minimal cancellation for non-terminal PipelineRuns
- job-level retry count and step/job timeout support

This is still an early runtime foundation. It is not a distributed scheduler, does not provide durable cross-process PostgreSQL runtime persistence yet, and does not implement Kubernetes, Argo CD, Git provider, artifact registry, or cloud provider integrations.

## Phase 1.6 Runtime Acceptance and Developer Experience

Phase 1.6 makes the runtime easier to verify and extend without adding external integrations:

- runtime acceptance matrix
- local and API smoke scripts
- safer example pipeline suite
- server-backed CLI inspection commands
- clearer validation errors
- request IDs in structured API errors
- developer getting started and troubleshooting docs

The scope remains shell-only and self-contained.

## Acceptance Criteria

- PipelineRun state transitions are tested.
- Runner assignment is explicit.
- Shell execution is controlled and cancellable.
- Logs and audit events are persisted or clearly modeled.
- Retry, timeout, cancellation, timeline, and runner heartbeat behavior are tested.

## Contribution Opportunities

- PipelineRun state machine.
- Runner protocol design.
- Log streaming design.
- Persistence tests.
- API schema updates.
