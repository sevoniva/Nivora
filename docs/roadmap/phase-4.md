# Phase 4: Visualization Backend and Future Frontend

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

## Contribution Opportunities

- API design for timelines.
- UX research.
- Frontend prototype after backend contracts stabilize.
- Read-model tests for dashboard summaries and topology projections.
