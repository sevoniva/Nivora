# Deployment Model

Nivora treats deployment as a workflow from Release intent to an Environment or Release Target. GitOps is one deployment mode, not the whole product.

## Core Concepts

- Release: versioned delivery intent, usually tied to immutable Artifacts.
- DeploymentRun: one execution of a Release or deployment plan.
- Environment: delivery context such as dev, staging, production, region, or tenant.
- Release Target: concrete target such as a host group, Kubernetes cluster, Argo CD application, cloud target, or webhook target.

## Phase 2.0 YAML Foundation

Phase 2.0 introduces the first CD-side execution foundation. It supports a minimal `Deployment` YAML spec that creates a DeploymentRun, renders static YAML manifest files, validates manifest shape, builds a DeploymentPlan, runs a policy pre-check placeholder, performs non-destructive dry-run validation through the local no-op manifest client, and records logs, events, audit entries, and timeline data.

The Phase 2.0 flow is:

```text
Release intent
-> DeploymentRun created
-> manifest rendering
-> deployment plan
-> policy pre-check
-> dry-run validation
-> status/events/audit/logs
```

Dry-run success means Nivora successfully rendered and validated the desired manifests in its current storage/runtime mode. It does not mean resources were applied to a cluster.

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

Phase 2.0 produces a stable desired-state summary and a diff placeholder. Live cluster diff and health verification are future work. These features should be target-aware and should not assume Kubernetes or Argo CD is always present.

## Current State

Phase 2.0 supports YAML deployment planning and dry-run foundation only. Production Kubernetes apply, Helm, Kustomize, Argo CD, cloud providers, host deployment, and registry integrations remain future work.
