# Observability Operations

Nivora Phase 4.2 provides a lightweight operations foundation for local development and early backend validation.

It does not provide production-grade observability. Prometheus deployment, log aggregation, trace export, alerting, sampling, retention, and incident workflows remain future work.

## HTTP Context

Every HTTP request receives a request ID from the API router. Callers may provide a correlation ID:

```sh
curl -H 'X-Correlation-Id: dev-test-1' http://localhost:8080/api/v1/system/runtime
```

Response headers include:

- `X-Request-Id`
- `X-Correlation-Id`
- `X-Trace-Id` when supplied by `traceparent` or `X-Trace-Id`

PipelineRun, DeploymentRun, and ReleaseExecution records created through HTTP include the correlation ID when one is available.

## Metrics

The `/metrics` endpoint exposes process-local text metrics:

- PipelineRun count
- DeploymentRun count
- runtime failure count
- PipelineRun duration observations
- DeploymentRun duration observations
- runner heartbeat count

These counters reset when the process restarts.

## Runtime Diagnostics

Use:

```sh
curl http://localhost:8080/api/v1/system/runtime
curl http://localhost:8080/api/v1/system/diagnostics
curl http://localhost:8080/metrics
```

`/api/v1/system/diagnostics` includes runtime mode, telemetry configuration, a metrics snapshot, and simple checks that help contributors confirm the backend is responding.

## Tracing

Tracing is a placeholder in Phase 4.2. Nivora accepts trace context headers so future OpenTelemetry integration has a stable shape, but it does not export spans.

## Secret Safety

Operational logs and diagnostics must not include secret values, tokens, kubeconfigs, private keys, or authorization headers. Diagnostics endpoints return runtime metadata only.
