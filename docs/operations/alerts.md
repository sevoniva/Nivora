# Alerting Guidance

Suggested alert rules for Nivora production-direction operations. Operators should adapt thresholds and notification channels to their environment.

## Critical Alerts (Page On-Call)

### Runner Offline

**Condition:** All registered runners have `lastHeartbeatAt > 120s` ago.
**Check:** `make runbook-check-runner` or query runner heartbeat.
**Impact:** No jobs can be executed. PipelineRuns queue indefinitely.
**Runbook:** `docs/operations/runbooks/offline-runner.md`

### Database Unavailable

**Condition:** `/readyz` reports `status: degraded` for database dependency.
**Check:** `make runbook-check-database`
**Impact:** Runtime state cannot be persisted or queried. API partially degraded.
**Runbook:** `docs/operations/runbooks/db-unavailable.md`

### Audit Chain Broken

**Condition:** `GET /api/v1/audit/verify?scopeType=<any>` returns `valid: false`.
**Check:** `make runbook-check-audit`
**Impact:** Possible tampering or data corruption in compliance audit records.
**Runbook:** Investigate `compliance_audit_records` for tampered hashes.

## Warning Alerts (Investigate)

### Stuck PipelineRuns

**Condition:** `GET /api/v1/system/runtime/recovery` shows `queuedRuns > 0` for >10 minutes.
**Check:** `make runbook-check-runtime`
**Impact:** Work not being processed. Worker may be down or overloaded.
**Runbook:** `docs/operations/runbooks/stuck-pipelinerun.md`

### High Failure Rate

**Condition:** PipelineRun failure rate > 10% over 1 hour.
**Check:** `curl /metrics` and review failure counters.
**Impact:** Executor or infrastructure issue causing repeated failures.
**Runbook:** `docs/operations/runbooks/failed-deploymentrun.md`

### Policy Denial Spike

**Condition:** Policy denials increase > 5x baseline over 15 minutes.
**Check:** Review audit search for denied actions.
**Impact:** Possible misconfiguration or unauthorized access attempts.
**Runbook:** `docs/operations/runbooks/policy-gate-denied.md`

### Outbox Backlog

**Condition:** Event outbox pending count > 100 for >5 minutes.
**Check:** Query `runtime_event_outbox` table for `status = 'pending'`.
**Impact:** Event processing delayed. May affect downstream integrations.

### Runner Heartbeat Stale

**Condition:** Any runner has `lastHeartbeatAt > 60s`.
**Check:** `make runbook-check-runner`
**Impact:** Runner may be about to go offline. Jobs may be reassigned.

### Deployment Concurrency Exceeded

**Condition:** Concurrent DeploymentRuns exceed tenant quota.
**Check:** `GET /api/v1/tenancy/usage`
**Impact:** Deployment throttling active. May delay releases.

## Alert Severity Matrix

| Condition | Severity | Urgency | Runbook |
|---|---|---|---|
| All runners offline | Critical | Immediate | runner |
| DB unavailable | Critical | Immediate | db |
| Audit chain broken | Critical | Immediate | audit |
| Stuck runs >10min | Warning | <1 hour | stuck-runs |
| Failure rate >10% | Warning | <1 hour | failed-deploy |
| Policy denial spike | Warning | <4 hours | policy-gate |
| Outbox backlog | Warning | <4 hours | runtime |
| Stale heartbeat | Warning | <1 hour | runner |
| Quota exceeded | Info | <8 hours | tenant |

## Integration

Operators should integrate these alerts with their monitoring stack:

- **Prometheus Alertmanager**: Define alert rules based on SLO burn rates.
- **PagerDuty/Opsgenie**: Route critical alerts to on-call.
- **Slack/Teams**: Route warning alerts to operations channel.
- **Runbook automation**: Link alerts to runbook check scripts (`make runbook-check-*`).

## Current Limitations

- Nivora does not ship alerting rules (PrometheusRule CRD or Alertmanager config).
- Alert thresholds are guidance only; operators must calibrate for their environment.
- No built-in notification for alert conditions (future notification provider integration).
- Metrics are in-memory; restart resets counters unless Postgres store is configured.
