# Visualization API

Phase 4.0 adds backend read models for future UI surfaces. It does not add frontend code, charting libraries, or new runtime behavior.

The visualization API is a projection layer over existing Control Plane records:

- PipelineRun records become DAG nodes, edges, timelines, and summaries.
- DeploymentRun records become deployment timelines, resource nodes, diff views, and health summaries.
- ReleasePlan and ReleaseExecution records become release overview and target execution views.
- Environment topology is derived from known DeploymentRuns and ReleaseTargets.
- Runner, security, and audit dashboards are aggregate views over existing runtime records.

## Principles

- Visualization endpoints must not invent state that is absent from the backend.
- Responses should be stable DTOs, not internal domain objects leaked wholesale.
- Timeline ordering should be deterministic.
- Frontend-specific rendering choices remain outside the backend.
- The project remains early-stage and not production-ready.

## DTO Families

- `GraphNode` and `GraphEdge` describe DAG-like views.
- `TimelineItem` describes ordered lifecycle, event, and audit records.
- `StatusBadge` gives UI clients a status value and coarse tone.
- `ResourceNode` describes topology and inventory nodes.
- `EnvironmentTopology` groups applications, targets, deployments, resources, and health.
- `DashboardSummary`, `RunnerSummary`, and `SecuritySummary` provide compact dashboard projections.

## Endpoint Groups

- `/api/v1/visualization/pipeline-runs/*`
- `/api/v1/visualization/deployments/*`
- `/api/v1/visualization/releases/*`
- `/api/v1/visualization/environments/*`
- `/api/v1/visualization/runners/summary`
- `/api/v1/visualization/security/summary`
- `/api/v1/visualization/audit/timeline`

These APIs are intended for future frontend and external tooling. They are not a commitment that a Nivora frontend exists in Phase 4.0.
