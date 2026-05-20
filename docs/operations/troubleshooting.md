# Troubleshooting

This guide covers Nivora backend diagnostics. Runbook check scripts provide automated health assessments for runtime, runners, database, and audit. Nivora is near-production-candidate (0.9.0-rc.1) and is not production-ready.

## API Is Not Responding

Check basic health:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

If these fail, confirm the server process is running and that `http.bind_address` uses the expected port.

## Request Fails

Capture the request ID and correlation ID from response headers:

```sh
curl -i -H 'X-Correlation-Id: debug-run-1' http://localhost:8080/api/v1/system/diagnostics
```

Use the IDs to match API access logs with PipelineRun, DeploymentRun, or ReleaseExecution records.

## Runtime State Looks Stale

The default Phase 4.2 runtime is in-process and in-memory. Metrics and runtime records reset when the process restarts unless a later persistence mode is explicitly configured.

Check:

```sh
curl http://localhost:8080/api/v1/system/runtime
curl http://localhost:8080/api/v1/system/diagnostics
```

## Metrics Are Missing

The local metrics endpoint is:

```sh
curl http://localhost:8080/metrics
```

Metrics are intentionally minimal and process-local. They do not require Prometheus and are not exported to an external backend in this phase.

## Logs Contain Sensitive Data

Treat this as a bug. Nivora should not log secret values, bearer tokens, kubeconfigs, private keys, or credential payloads. Use SecretRef and CredentialRef metadata instead of raw values.

## Traces Are Not Exported

That is expected in Phase 4.2. Trace IDs can be carried through headers and diagnostics, but OpenTelemetry span export is future work.

## Runbook Check Scripts

Automated read-only diagnostics scripts for common operational scenarios. All scripts query the server API and never mutate state.

### Runtime Health

```sh
make runbook-check-runtime
# or: NIVORA_SERVER_URL=http://localhost:8080 ./scripts/runbook-check-runtime.sh
```

Checks: healthz, readyz, runtime recovery status, queued/stale/expired runs, recent PipelineRuns and DeploymentRuns, failure counts.

**Expected output:** Health OK, 0 stuck runs, recent runs listed.
**If stuck runs:** Run reconciliation or check worker.
**Escalate if:** Stuck runs persist after reconciliation.

### Runner Fleet

```sh
make runbook-check-runner
```

Checks: registered runners, online/offline status, heartbeat timestamps.

**Expected output:** All runners online with recent heartbeats.
**If offline runners:** Run offline detection, check runner process.
**Escalate if:** All runners offline.

### Database

```sh
make runbook-check-database
```

Checks: runtime store mode, database connectivity (via readyz), migration status.

**Expected output:** postgres runtime store, server ready.
**If memory store:** Warns — state will be lost on restart.
**If degraded:** PostgreSQL may be unavailable.

### Audit Integrity

```sh
make runbook-check-audit
```

Checks: hash chain verification for all 9 audit scopes, recent audit entries, evidence bundles, retention policy.

**Expected output:** All available chains valid, audit entries found.
**If chain invalid:** Possible tampering — investigate immediately.
**If no records:** May be expected with memory store.

### Runbook Reference

| Symptom | Runbook |
|---|---|
| Stuck PipelineRun | `docs/operations/runbooks/stuck-pipelinerun.md` |
| Failed DeploymentRun | `docs/operations/runbooks/failed-deploymentrun.md` |
| Offline runner | `docs/operations/runbooks/offline-runner.md` |
| Policy gate denied | `docs/operations/runbooks/policy-gate-denied.md` |
| Database unavailable | `docs/operations/runbooks/db-unavailable.md` |
| Object store unavailable | `docs/operations/runbooks/object-store-unavailable.md` |
