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
- PipelineRun and DeploymentRun list views.
- PipelineRun detail.
- Deployment detail.
- Release and release execution views.
- Release execution detail.
- Runner summary.
- Security summary.
- Audit timeline.
- Environment topology.
- Minimal API client for existing visualization endpoints.

Phase 6.4 extends this into a web console foundation that also calls existing runtime list APIs. It remains a backend-driven console and does not add production UI claims.

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

## Phase 4.3 / 7.4 Plugin and Extension System Foundation

Phase 4.3 defined plugin metadata, a capability registry, and an external protocol skeleton. Phase 7.4 stabilizes the plugin API version, compatibility checks, validate-config lifecycle, and adapter templates.

Scope:

- Plugin manifest model.
- Plugin types for SCM, artifact, cloud, executor, secret, notification, policy, scanner, and GitOps extensions.
- Static built-in adapter capability registry.
- Plugin metadata APIs and CLI commands.
- Plugin API version and compatibility validation.
- Adapter manifest templates.
- External plugin protocol skeleton in `api/proto/plugin.proto`.
- Adapter authoring and plugin RFC documentation.

Non-goals:

- Go plugin dynamic loading.
- Unsafe external code loading.
- Marketplace behavior.
- Frontend plugin management.
- Production plugin execution.

## Phase 4.4 Packaging and Deployment Foundation

Phase 4.4 makes local and Kubernetes installation easier to verify.

Scope:

- Hardened multi-binary Docker image with non-root runtime.
- Docker Compose stack for server, worker, runner, PostgreSQL, and MinIO local validation.
- Helm chart for server, worker, runner, ConfigMap, Secret placeholder, Service, optional Ingress, and optional migration Job.
- Minimal raw Kubernetes manifests.
- Configuration examples for local, compose, and production-shaped installs.
- Makefile targets for Docker and Helm validation.
- Operations installation and configuration docs.

Non-goals:

- Kubernetes operator.
- Cloud-provider-specific deployment.
- Committed secrets.
- Production readiness claims.

## Phase 5.0 Alpha Release Hardening

Phase 5.0 stabilizes the public alpha surface without adding large new features.

Scope:

- Capability matrix.
- Alpha demo path.
- Changelog, release template, and version alignment.
- Release checklist and known limitations.
- CI and Makefile verification hardening.
- Documentation consistency review.

Non-goals:

- Production GA release.
- New external integrations.
- Architecture rewrite.
- Hiding limitations.

## Contribution Opportunities

- API design for timelines.
- UX research.
- Frontend prototype after backend contracts stabilize.
- Read-model tests for dashboard summaries and topology projections.
