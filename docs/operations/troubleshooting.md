# Troubleshooting

This guide covers early Nivora backend diagnostics. The project is still early-stage and not production-ready.

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
