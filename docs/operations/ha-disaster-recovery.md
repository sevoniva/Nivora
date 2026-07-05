# HA and Disaster Recovery Operations

This guide explains how to operate the Phase 8.2 HA/DR foundation. It is not a production SLA or automated disaster recovery product.

## Health and Diagnostics

Use:

```sh
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/system/diagnostics
curl http://localhost:8080/api/v1/system/runtime/recovery
```

Readiness returns a status plus dependency checks. A critical degraded dependency returns HTTP 503. Diagnostics includes runtime context, metrics, and the same dependency checks.

## Server Restart

1. Stop traffic or drain the instance.
2. Restart `nivora-server`.
3. Check `/healthz`, `/readyz`, and `/api/v1/system/diagnostics`.
4. Confirm new requests include request/correlation IDs.

If the runtime store is `memory`, state from that server process is not recoverable.

## Worker Restart

1. Restart `nivora-worker`.
2. Inspect recoverable work:

```sh
nivora runtime status --token-env NIVORA_AUTH_TOKEN
```

3. Run reconciliation:

```sh
nivora runtime reconcile --token-env NIVORA_AUTH_TOKEN
```

This checks queued work, expired leases, cancel requests, timeouts, and outbox records where the current runtime supports them.

## Runner Disconnect

Runners heartbeat to the server. If a runner disconnects:

1. Check runner status.
2. Restart the runner with its configured token.
3. Mark stale runners offline when needed:

```sh
curl -X POST 'http://localhost:8080/api/v1/runners/offline-detect?timeoutSeconds=60'
```

Assigned jobs with expired leases can be recovered by the worker/runtime reconciliation path where implemented.

## DB Unavailable

The server currently reports database configuration posture. For production-direction deployments:

- use PostgreSQL runtime store
- run migrations before starting workers
- back up before schema changes
- restore DB before object-store-dependent reconciliation

For a running server, use `nivora doctor live --server <url> --token-env NIVORA_AUTH_TOKEN` to read the current diagnostics, runtime recovery summary, event outbox counts, and audit hash-chain verification result. This is a read-only live check; it does not run migrations, repair the database, or prove production readiness.

## Object Store Unavailable

If object storage is unavailable, workflows that depend on stored snapshots, manifests, or evidence may be incomplete. Restore the object store before replaying runtime or compliance workflows.

## Event Publish Failure

Do not delete pending or failed outbox records. Restore event transport dependencies, then run runtime reconciliation. The outbox is the recovery surface for future durable event publication.

## Helm and Docker Compose

Helm and Docker Compose examples remain foundation assets. For HA direction:

- run more than one server replica behind a load balancer only after configuring shared persistent stores
- run workers separately from servers
- keep runners independently restartable
- back up PostgreSQL and object store outside the application pods
- avoid embedding secret values in Helm values or compose files
