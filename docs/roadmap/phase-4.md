# Phase 4: Visualization Frontend

## Objective

Build visualization surfaces on top of stable backend APIs.

## Scope

- Pipeline DAG.
- Deployment timeline.
- Environment topology.
- Runner dashboard.
- Audit timeline.
- Security dashboard.

## Non-Goals

- Frontend before backend APIs are stable.
- Replacing backend audit or event models with frontend-only state.

## Expected Deliverables

User-facing visualization built on backend APIs and durable delivery state.

## Acceptance Criteria

- Visualization APIs are documented.
- UI does not invent state that is absent from the Control Plane.
- Dashboards reflect PipelineRuns, DeploymentRuns, Runners, Audit, and PolicyResults.

## Contribution Opportunities

- API design for timelines.
- UX research.
- Frontend prototype after backend contracts stabilize.

