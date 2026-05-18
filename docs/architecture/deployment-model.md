# Deployment Model

Nivora treats deployment as a workflow from Release intent to an Environment or Release Target. GitOps is one deployment mode, not the whole product.

## Core Concepts

- Release: versioned delivery intent, usually tied to immutable Artifacts.
- DeploymentRun: one execution of a Release or deployment plan.
- Environment: delivery context such as dev, staging, production, region, or tenant.
- Release Target: concrete target such as a host group, Kubernetes cluster, Argo CD application, cloud target, or webhook target.

## Phase 2.1 YAML Runtime Foundation

Phase 2.1 extends the CD-side foundation. It supports a minimal `Deployment` YAML spec that creates a DeploymentRun, renders static YAML manifest files, validates manifest shape, builds a DeploymentPlan, runs a policy pre-check placeholder, performs controlled server-side dry-run behavior through a manifest client port, can run explicit no-op local apply when confirmed, and records logs, events, audit entries, resource inventory, rollout result, rollback baseline, and timeline data.

The Phase 2.1 flow is:

```text
Release intent
-> DeploymentRun created
-> manifest rendering
-> deployment plan
-> policy pre-check
-> dry-run validation
-> optional explicit apply
-> optional rollout verification
-> status/events/audit/logs
```

Dry-run success means Nivora successfully rendered and validated the desired manifests in its current storage/runtime mode. It does not mean resources were applied to a cluster. Apply is explicit and never default.

## Deployment Modes

Future deployment modes may include:

- host deployment
- Kubernetes YAML deployment beyond dry-run planning
- Helm
- Kustomize
- Argo CD GitOps release
- webhook deployment
- cloud-provider-specific deployment through CloudProvider Adapters

## Rollback

Rollback should be modeled as an auditable operation with a reason, target, prior version or Artifact, status, logs, and verification result.

## Deployment Diff and Health Verification

Phase 2.1 produces a stable desired-state summary and a diff placeholder. Live cluster diff is future work. Rollout verification is modeled through the manifest client port and defaults to a no-op local result.

## Artifact Binding

Phase 2.2 connects DeploymentPlan output to ReleaseArtifacts. Deployment specs may include inline artifact references, and the planner records normalized artifact summaries, immutability warnings, and simple manifest image verification warnings.

This is verify-first behavior. Nivora does not mutate manifests by default, does not resolve registry digests over the network, and does not claim full registry integration.

## Current State

Phase 2.2 supports YAML deployment planning, dry-run, explicit local no-op apply, resource inventory, rollout result modeling, rollback baseline, artifact summaries, and manifest image verification only. Production Kubernetes apply, Helm, Kustomize, Argo CD, cloud providers, host deployment, registry integrations, image signing, and scanning remain future work.
