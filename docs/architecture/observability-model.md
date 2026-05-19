# Observability Model

Nivora should make delivery behavior explainable through logs, events, metrics, traces, and timelines.

## Phase 4.2 Runtime Signals

Phase 4.2 adds a small operations foundation without introducing a heavy observability stack. The HTTP API now carries request IDs, correlation IDs, and trace IDs in request context and response headers. Runtime diagnostics and local metrics are exposed for development and contributor debugging.

This is not a production observability platform. External metrics storage, distributed tracing export, log retention, sampling, and alerting remain future work.

## Logs

Logs explain execution behavior. Runner and Executor logs should be correlated with PipelineRuns, JobRuns, StepRuns, DeploymentRuns, and AuditLogs.

Phase 1.5 stores stdout and stderr as ordered LogChunks for each PipelineRun. Log streaming, external log storage, and retention policies are future work.

API access logs are structured and include method, path, status, duration, request ID, correlation ID, and trace ID. They intentionally avoid request bodies, query values, credentials, and secret material.

## Events

Events should describe lifecycle changes such as PipelineRun created, queued, started, completed, failed, canceled, JobRun assigned, JobRun started, JobRun completed, JobRun failed, runner heartbeat, DeploymentRun started, and policy violation detected.

Phase 1.5 stores PipelineRun events in the in-memory runtime and exposes them through events and timeline APIs.

## Metrics and Traces

Phase 4.2 exposes an in-process metrics registry and a lightweight text endpoint at `/metrics`. It tracks PipelineRun count, DeploymentRun count, runtime failure count, observed run durations, and runner heartbeat count.

Tracing is configuration-only in this phase. Nivora accepts trace IDs from `traceparent` or `X-Trace-Id` headers and carries them through HTTP context and diagnostics responses, but it does not export spans yet. OpenTelemetry remains the likely future direction if it can be added without weakening the modular architecture.

## Correlation IDs

Requests, events, logs, and audit records should share correlation IDs where practical. Phase 4.2 records the inbound correlation ID on PipelineRun, DeploymentRun, and ReleaseExecution records created through HTTP routes.

Header behavior:

- `X-Request-Id` is generated or accepted by the API router.
- `X-Correlation-Id` is accepted from callers and defaults to the request ID when omitted.
- `traceparent` and `X-Trace-Id` are accepted as tracing placeholders.

## Diagnostics

Operational endpoints:

- `GET /healthz` reports basic process health.
- `GET /readyz` reports API readiness.
- `GET /metrics` exposes lightweight local counters.
- `GET /api/v1/system/runtime` returns runtime mode and telemetry configuration.
- `GET /api/v1/system/diagnostics` returns runtime context, metrics snapshot, and simple diagnostic checks.

## Timelines

Phase 1.5 exposes a minimal PipelineRun timeline from stored events. Future visualization APIs should support richer pipeline timelines, deployment timelines, runner heartbeat history, and audit timelines.
