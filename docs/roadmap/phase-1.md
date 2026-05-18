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
