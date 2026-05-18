# DeploymentRun

A DeploymentRun is an execution of a Release or deployment plan against an Environment or Release Target.

## Why It Exists

DeploymentRuns record what changed, where it changed, how it changed, who approved it, which Policies applied, and what the result was.

## Relationships

- References a Release.
- Targets an Environment and optionally a Release Target.
- May use a Runner and Executor.
- Produces logs, events, audit records, and rollback context.

## Current Implementation

Phase 2.1 creates DeploymentRuns from a minimal YAML deployment spec. The runtime renders static Kubernetes-style YAML manifests, validates required manifest fields, builds a DeploymentPlan, evaluates an allow-all policy placeholder by default, performs a non-destructive dry-run through a manifest client port, can run explicit no-op local apply when confirmed, and records logs, events, audit entries, resources, rollout result, rollback baseline, and timeline data.

The current implementation does not apply manifests to a Kubernetes cluster by default. A `Succeeded` DeploymentRun in the default Phase 2.1 local runtime means the dry-run or confirmed no-op apply path succeeded.

## Common Confusion

A DeploymentRun is not always Kubernetes rollout state. It can represent host deployment, YAML apply, Helm, Kustomize, Argo CD GitOps, cloud deployment, or webhook deployment.
