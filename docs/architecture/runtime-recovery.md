# Runtime Recovery

Nivora uses a small internal durable runtime foundation rather than a workflow engine. The Phase 5.2 design keeps recovery explicit and database-friendly.

## Recovery Loop

The worker performs a reconciliation pass:

1. cancel non-terminal PipelineRuns with `cancelRequested=true`;
2. find stale running PipelineRuns by expired lease or old update time;
3. mark runs past the timeout window as `Timeout`;
4. return recoverable stale runs and expired JobRun claims to `Queued`;
5. acquire leases for queued PipelineRuns and execute them;
6. publish pending event outbox records and schedule retry metadata for failures.

## Lease Fields

PipelineRun, DeploymentRun, and ReleaseExecution records include:

- `ownerId`
- `leaseExpiresAt`
- `attempt`
- `heartbeatAt`

PipelineRun persistence stores these fields in PostgreSQL. DeploymentRun and ReleaseExecution use the same shape as a foundation for future durable recovery.

## Outbox

Event outbox records support:

- `pending`
- `published`
- `failed`
- `retryCount`
- `nextAttemptAt`
- `lastError`

Outbox publication is retried by reconciliation. External broker delivery remains future work.

## Limitations

Phase 5.2 does not add a distributed workflow engine. It does not guarantee exactly-once execution. It reduces restart risk for the current PipelineRun runtime and establishes the model for DeploymentRun and ReleaseExecution recovery.
