# Runbook: Offline Runner

Use this when a runner stops heartbeating or cannot claim jobs.

## Signals

- Runner status is offline or stale.
- `nivora_runner_heartbeat_total` stops increasing.
- Job claims fail or queue length grows.

## Triage

```sh
nivora runner list --token-env NIVORA_AUTH_TOKEN
curl http://localhost:8080/api/v1/system/runtime/recovery
```

Check runner process logs for token, network, or executor errors. Do not print runner tokens.

## Recovery

1. Restart the runner process.
2. Confirm it heartbeats:

```sh
nivora runner heartbeat --name <runner-name> --token-env NIVORA_RUNNER_TOKEN
```

3. Mark stale runners offline if needed:

```sh
curl -X POST 'http://localhost:8080/api/v1/runners/offline-detect?timeoutSeconds=60'
```

4. Run runtime reconciliation so expired job leases can be recovered.

## Escalation Notes

Rotate runner tokens if token exposure is suspected. Raw tokens are returned only at creation/rotation time.
