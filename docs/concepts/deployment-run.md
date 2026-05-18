# DeploymentRun

A DeploymentRun is an execution of a Release or deployment plan against an Environment or Release Target.

## Why It Exists

DeploymentRuns record what changed, where it changed, how it changed, who approved it, which Policies applied, and what the result was.

## Relationships

- References a Release.
- Targets an Environment and optionally a Release Target.
- May use a Runner and Executor.
- Produces logs, events, audit records, and rollback context.

## Common Confusion

A DeploymentRun is not always Kubernetes rollout state. It can represent host deployment, YAML apply, Helm, Kustomize, Argo CD GitOps, cloud deployment, or webhook deployment.

