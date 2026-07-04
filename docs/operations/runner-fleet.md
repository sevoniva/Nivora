# Runner Fleet Operations

Phase 5.3 adds a secure runner fleet foundation.

## Register A Runner

```bash
nivora runner register --name local-runner --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

The response includes a raw runner token exactly once. Store it in an environment variable or secret manager outside the repository:

```bash
export NIVORA_RUNNER_TOKEN='<returned runner token>'
```

Nivora stores only a token hash and token metadata.

## Runner Groups

Create runner groups to constrain which project/environment work a runner fleet can claim and which executors it may advertise:

```bash
nivora runner groups create --name prod-runners --project-id project-a --environment-id env-prod --executor shell --max-concurrency 2 --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
nivora runner register --name prod-runner-1 --group-id prod-runners --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
```

Group constraints are enforced during registration and job claim. They are metadata guardrails, not a sandbox.

## Heartbeat And Claim Work

```bash
nivora runner heartbeat --name local-runner --token-env NIVORA_RUNNER_TOKEN
nivora runner claim --name local-runner --token-env NIVORA_RUNNER_TOKEN
```

Runner-owned mutation endpoints require the token as `Authorization: Bearer <token>` or `X-Nivora-Runner-Token`.

## Rotation

```bash
nivora runner token rotate local-runner --token-env NIVORA_AUTH_TOKEN
```

The new raw token is returned only in the rotation response. Old tokens stop validating.

Revoke a token when retiring a runner:

```bash
nivora runner token revoke local-runner --token-env NIVORA_AUTH_TOKEN
```

## Offline Detection

The runtime reconciliation loop marks stale online runners offline after missed heartbeat. Operators can also call:

```bash
nivora runner offline-detect --timeout-seconds 60 --token-env NIVORA_AUTH_TOKEN
```

## Limits

- No autoscaling is implemented.
- No privileged execution is enabled by default.
- Runner groups reduce accidental cross-scope claims, but runner isolation, sandboxing, and production credential distribution remain future hardening work.
- Nivora remains a hardened beta-candidate foundation and is not production-ready.
