# Service Level Objectives

Nivora defines example SLOs for production-direction operations. These are guidance targets, not contractual commitments. Operators should adjust thresholds based on their deployment profile and risk tolerance.

## SLO Definitions

### API Availability

| Metric | Target | Window |
|---|---|---|
| `/healthz` success rate | 99.9% | 30 days |
| `/readyz` success rate | 99.5% | 30 days |
| API error rate (5xx) | <0.1% | 30 days |

### Pipeline Execution

| Metric | Target | Window |
|---|---|---|
| PipelineRun success rate | >95% | 7 days |
| PipelineRun P50 latency | <30s | 7 days |
| PipelineRun P95 latency | <5min | 7 days |
| Stuck PipelineRuns (queued >10min) | <5 | Real-time |

### Deployment Planning

| Metric | Target | Window |
|---|---|---|
| DeploymentPlan success rate | >95% | 7 days |
| DeploymentPlan P50 latency | <5s | 7 days |
| Failed DeploymentRuns | <10% of total | 7 days |

### Runner Health

| Metric | Target | Window |
|---|---|---|
| Runner heartbeat freshness | <60s since last heartbeat | Real-time |
| Runner online rate | >90% of registered runners | 5 min |
| Job claim latency P50 | <500ms | 7 days |

### Audit Integrity

| Metric | Target | Window |
|---|---|---|
| Audit chain verification success | 100% | Real-time |
| Audit records present | >0 for all 9 scopes | 24 hours |
| Evidence bundles available | For completed runs | 24 hours |

### Recovery

| Metric | Target | Window |
|---|---|---|
| Recovery Time Objective (RTO) | <10 min | Per incident |
| Recovery Point Objective (RPO) | <1 min (PostgreSQL) | Per incident |
| Stale runs reconciled | <5 min after detection | Real-time |

## Measuring SLOs

### Current (Nivora Metrics Endpoint)

```bash
curl http://localhost:8080/metrics
```

Returns:
- `nivora_pipeline_runs_total`
- `nivora_pipeline_run_duration_ms`
- `nivora_deployment_runs_total`
- `nivora_deployment_run_duration_ms`
- `nivora_failures_total`
- `nivora_runner_heartbeats_total`
- `nivora_job_claims_total`
- `nivora_policy_denials_total`

### Future (Prometheus/Grafana)

Operators should deploy Prometheus to scrape `/metrics` and Grafana to visualize. Export the following as Prometheus-compatible gauges/counters:

```prometheus
# HELP nivora_pipeline_runs_total Total PipelineRuns created.
# TYPE nivora_pipeline_runs_total counter
nivora_pipeline_runs_total{status="succeeded"} 150
nivora_pipeline_runs_total{status="failed"} 5

# HELP nivora_runner_heartbeats_total Total runner heartbeats received.
# TYPE nivora_runner_heartbeats_total counter
nivora_runner_heartbeats_total 1200
```

### Health/Diagnostics

```bash
# Runtime recovery status (queued, stale, expired)
curl http://localhost:8080/api/v1/system/runtime/recovery

# System diagnostics (runtime mode, unsafe flags)
curl http://localhost:8080/api/v1/system/diagnostics

# Audit chain verification
curl http://localhost:8080/api/v1/audit/verify?scopeType=pipeline
```

## SLO Burn Rate Alerts

Operators should configure alerts when SLO burn rate exceeds thresholds:

| Burn Rate | Action |
|---|---|
| 1x (normal) | No action |
| 2x | Investigate within business hours |
| 5x | Page on-call |
| 10x | Incident declared |

## Current Limitations

- Metrics are in-memory counters (lost on restart unless Postgres store configured).
- No Prometheus-compatible format for `/metrics` endpoint (plain text summary).
- No histogram/bucket support for latency percentiles.
- No SLO dashboard or automated SLO calculation.
- Error budget tracking is manual.
- Correlation/trace IDs are generated but not exported to a tracing system.
