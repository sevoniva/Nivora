# Kubernetes YAML Runtime

Phase 2.1 extends the YAML deployment foundation into a controlled Kubernetes YAML runtime model.

## Current Behavior

- Static YAML manifests are rendered and validated.
- Resource inventory is extracted from `apiVersion`, `kind`, `metadata.name`, namespace, labels, and annotations.
- DeploymentPlan records target type, namespace, dry-run/apply flags, wait flag, timeout, actions, warnings, and desired resources.
- Server-side dry-run is represented through the `KubernetesManifestClient` port.
- The default local runtime uses a no-op manifest client, so tests and local smoke checks do not require Kubernetes.
- Apply is explicit and never default. Phase 6.0 requires confirmation for apply and keeps real Kubernetes execution behind the manifest client adapter boundary.
- Rollout verification is modeled through the same port and is no-op by default.
- Rollback has a guarded manifest-restore foundation in Phase 6.0. It requires confirmation and does not prune/delete resources by default.

## What Is Not Implemented

- Production Kubernetes apply semantics.
- Kubernetes SDK integration.
- Argo CD.
- Helm.
- Kustomize.
- Cloud providers.
- Host SSH deployment.
- Registry integration.

Nivora remains early-stage and not production-ready.

## Safety Rules

- Keep `options.dryRun: true` and `options.apply: false` in default examples.
- Use `nivora deployment apply --confirm` only for explicit local testing.
- Do not commit kubeconfigs or cluster endpoints.
- Do not make kind, Harbor, Nexus, GitLab, Gitea, or Argo CD required for CI.
