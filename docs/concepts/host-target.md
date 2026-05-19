# Host Target

A Host Target represents VM or bare-metal delivery. It is a ReleaseTarget type used when a DeploymentRun should plan delivery to one or more hosts rather than a Kubernetes cluster, Argo CD application, cloud target, or webhook.

## Why It Exists

Many delivery systems still need to deploy JARs, binaries, static packages, or tarballs to long-lived machines. Nivora models that flow explicitly so it can become auditable and reversible instead of hidden inside ad hoc scripts.

## Relationships

- A `HostGroup` belongs to an Environment.
- A `Host` belongs to a HostGroup or inline host list.
- A `DeploymentRun` with `target.type=host` creates a `HostDeploymentPlan`.
- A `HostExecutor` performs prepare/upload/execute/health-check operations through an adapter.
- Rollback is guarded and modeled as a symlink restore; it does not delete release directories by default.

## Common Confusion

Host Target is not cloud host discovery. It is not SSH automation by default. Phase 8.1 supports safe plan generation, noop execution, batch rollout metadata, typed health checks, guarded rollback, and a guarded SSH adapter surface that still requires explicit configuration.
