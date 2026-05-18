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

A basic flow from PipelineRun creation to controlled runner execution, logs, status, and audit.

## Acceptance Criteria

- PipelineRun state transitions are tested.
- Runner assignment is explicit.
- Shell execution is controlled and cancellable.
- Logs and audit events are persisted or clearly modeled.

## Contribution Opportunities

- PipelineRun state machine.
- Runner protocol design.
- Log streaming design.
- Persistence tests.
- API schema updates.

