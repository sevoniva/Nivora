# Runner Fleet Operations

Phase 5.3 adds a secure runner fleet foundation.

## Register A Runner

```bash
nivora runner register --name local-runner --server http://localhost:8080
```

The response includes a raw runner token exactly once. Store it in an environment variable or secret manager outside the repository:

```bash
export NIVORA_RUNNER_TOKEN='<returned runner token>'
```

Nivora stores only a token hash and token metadata.

## Heartbeat And Claim Work

```bash
nivora runner heartbeat --name local-runner --token-env NIVORA_RUNNER_TOKEN
nivora runner claim --name local-runner --token-env NIVORA_RUNNER_TOKEN
```

Runner-owned mutation endpoints require the token as `Authorization: Bearer <token>` or `X-Nivora-Runner-Token`.

## Rotation

```bash
nivora runner token rotate local-runner
```

The new raw token is returned only in the rotation response. Old tokens stop validating.

Revoke a token when retiring a runner:

```bash
nivora runner token revoke local-runner
```

## Offline Detection

The runtime reconciliation loop marks stale online runners offline after missed heartbeat. Operators can also call:

```bash
curl -X POST 'http://localhost:8080/api/v1/runners/offline-detect?timeoutSeconds=60'
```

## Limits

- No autoscaling is implemented.
- No privileged execution is enabled by default.
- Runner isolation, sandboxing, and production credential distribution remain future hardening work.
- Nivora remains early-stage and not production-ready.
