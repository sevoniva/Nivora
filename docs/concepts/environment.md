# Environment

An Environment is a delivery context such as dev, staging, production, regional production, or a tenant-specific context.

## Why It Exists

Environments allow policies, approvals, locks, audit rules, and Release Targets to be scoped by delivery context.

## Relationships

- Belongs to a Project.
- Contains Release Targets.
- Can be locked to prevent unsafe DeploymentRuns.
- May have stricter Policies and approvals.

## Release Target

A Release Target may be a host group, Kubernetes cluster, Argo CD application, cloud target, or webhook target.

In Phase 2.0, the implemented target type is `kubernetes-yaml` for static manifest planning and dry-run validation only. The target namespace is explicit in the deployment spec, but no kubeconfig or cluster endpoint is required for normal tests.

## Common Confusion

An Environment is not only a Kubernetes namespace. Kubernetes namespaces may be one implementation detail of a Release Target.
