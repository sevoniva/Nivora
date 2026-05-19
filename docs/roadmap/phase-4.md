# Phase 4: Visualization, Web UI, and Operations Foundations

## Objective

Build visualization-ready backend APIs first, then future frontend surfaces on top of stable backend state.

## Scope

- Pipeline DAG.
- Deployment timeline.
- Environment topology.
- Runner dashboard.
- Audit timeline.
- Security dashboard.
- Release execution overview.
- Resource inventory and health views.

## Non-Goals

- Frontend implementation in Phase 4.0.
- Charting libraries in the backend.
- Replacing backend audit or event models with frontend-only state.
- Claiming that a Nivora frontend exists before it does.

## Expected Deliverables

- Visualization DTOs.
- PipelineRun DAG, timeline, and summary endpoints.
- DeploymentRun timeline, resources, diff, and health endpoints.
- Release overview and ReleaseExecution target/timeline endpoints.
- Environment topology endpoint.
- Runner, security, and audit dashboard endpoints.
- OpenAPI and developer documentation.

## Acceptance Criteria

- Visualization APIs are documented.
- UI does not invent state that is absent from the Control Plane.
- Dashboards reflect PipelineRuns, DeploymentRuns, Runners, Audit, and PolicyResults.
- No frontend code is added in Phase 4.0.
- Normal verification passes.

## Phase 4.1 Web UI Foundation

Phase 4.1 adds a minimal React + TypeScript + Vite app under `web/`.

Scope:

- Dashboard.
- PipelineRun detail.
- Deployment detail.
- Release execution detail.
- Runner summary.
- Security summary.
- Minimal API client for existing visualization endpoints.

Non-goals:

- Complete product UI.
- New backend behavior.
- Heavy design system.
- Production readiness claims.

## Phase 4.2 Observability and Operations Hardening

Phase 4.2 adds lightweight backend operations support without introducing a full observability stack.

Scope:

- Request ID, correlation ID, and trace ID propagation through HTTP.
- Structured access logs with non-secret operational fields.
- Process-local metrics for PipelineRuns, DeploymentRuns, failures, durations, and runner heartbeats.
- `/metrics`, `/api/v1/system/runtime`, and `/api/v1/system/diagnostics` endpoints.
- Tracing configuration placeholder for future OpenTelemetry work.
- Operations documentation for observability and troubleshooting.

Non-goals:

- Prometheus deployment.
- Distributed trace export.
- Log aggregation or retention.
- Frontend observability dashboards.
- Production readiness claims.

## Phase 4.3 Plugin and Extension System Foundation

Phase 4.3 defines plugin metadata, a capability registry, and an external protocol skeleton.

Scope:

- Plugin manifest model.
- Plugin types for SCM, artifact, cloud, executor, secret, notification, policy, scanner, and GitOps extensions.
- Static built-in adapter capability registry.
- Plugin metadata APIs and CLI commands.
- External plugin protocol skeleton in `api/proto/plugin.proto`.
- Adapter authoring and plugin RFC documentation.

Non-goals:

- Go plugin dynamic loading.
- Unsafe external code loading.
- Marketplace behavior.
- Frontend plugin management.
- Production plugin execution.

## Contribution Opportunities

- API design for timelines.
- UX research.
- Frontend prototype after backend contracts stabilize.
- Read-model tests for dashboard summaries and topology projections.
