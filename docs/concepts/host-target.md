# Host Target

A Host Target represents VM or bare-metal delivery. It is a ReleaseTarget type used when a DeploymentRun should plan delivery to one or more hosts rather than a Kubernetes cluster, Argo CD application, cloud target, or webhook.

## Why It Exists

Many delivery systems still need to deploy JARs, binaries, static packages, or tarballs to long-lived machines. Nivora models that flow explicitly so it can become auditable and reversible instead of hidden inside ad hoc scripts.

## Relationships

- A `HostGroup` belongs to an Environment.
- A `Host` belongs to a HostGroup or inline host list.
- A `DeploymentRun` with `target.type=host` creates a `HostDeploymentPlan`.
- A `HostExecutor` performs prepare/upload/execute/health-check operations through an adapter.
- Rollback is modeled as a plan, not destructive execution, in Phase 3.5.

## Common Confusion

Host Target is not cloud host discovery. It is not SSH automation by default. Phase 3.5 only introduces the backend model, safe plan generation, noop execution, and a disabled SSH adapter skeleton.
