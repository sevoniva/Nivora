# Observability Operations

Nivora Phase 8.3 provides a production-direction observability foundation for local development, staging, and beta operations.

It does not provide a complete managed observability stack. Prometheus deployment, log aggregation, trace export, sampling, retention, and incident workflows must be configured by operators.

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
- runner job claim attempts
- job claim latency observations
- runtime queue time observations
- policy denial count

These counters reset when the process restarts.

Suggested Prometheus scrape:

```yaml
scrape_configs:
  - job_name: nivora-server
    static_configs:
      - targets: ["nivora-server:8080"]
```

## SLO Foundation

These are recommended starting SLOs for beta environments, not GA commitments:

| Area | Suggested SLO | Primary signals |
| --- | --- | --- |
| API availability | 99.5% successful `/readyz` over 30 days | readiness status, HTTP error logs |
| Pipeline completion | 95% non-canceled PipelineRuns complete successfully | `nivora_pipeline_run_total`, `nivora_runtime_failure_total` |
| Deployment completion | 95% dry-run/guarded DeploymentRuns complete successfully | `nivora_deployment_run_total`, failures |
| Runner freshness | 99% active runners heartbeat within expected interval | `nivora_runner_heartbeat_total`, runner offline events |
| Policy gate visibility | 100% policy denials create events/audit | `nivora_policy_denial_total`, audit search |

## Alert Suggestions

- `NivoraReadinessDegraded`: `/readyz` returns non-200 for 5 minutes.
- `NivoraRuntimeFailuresHigh`: `nivora_runtime_failure_total` increases faster than normal baseline.
- `NivoraRunnerHeartbeatMissing`: expected runner heartbeat count stops increasing.
- `NivoraJobClaimLatencyHigh`: job claim latency total/observations indicates sustained slow claims.
- `NivoraPolicyDenials`: `nivora_policy_denial_total` increases in production environments.
- `NivoraQueueTimeGrowing`: queue time observations rise while completion metrics stall.

## Runtime Diagnostics

Use:

```sh
curl http://localhost:8080/api/v1/system/runtime
curl http://localhost:8080/api/v1/system/diagnostics
curl http://localhost:8080/metrics
```

`/api/v1/system/diagnostics` includes runtime mode, telemetry configuration, a metrics snapshot, and dependency checks for database, object store, event bus, outbox recovery, and runner reconnect posture.

## Tracing

Tracing remains a foundation. Nivora accepts W3C `traceparent` and `X-Trace-Id` headers, propagates `trace_id` into diagnostics and structured access logs, and keeps the exporter configuration placeholder. Full OpenTelemetry span export remains future work.

## Runbooks

- [Stuck PipelineRun](runbooks/stuck-pipelinerun.md)
- [Failed DeploymentRun](runbooks/failed-deploymentrun.md)
- [Offline Runner](runbooks/offline-runner.md)
- [Database Unavailable](runbooks/db-unavailable.md)
- [Object Store Unavailable](runbooks/object-store-unavailable.md)
- [Policy Gate Denied](runbooks/policy-gate-denied.md)

## Secret Safety

Operational logs and diagnostics must not include secret values, tokens, kubeconfigs, private keys, or authorization headers. Diagnostics endpoints return runtime metadata only.
