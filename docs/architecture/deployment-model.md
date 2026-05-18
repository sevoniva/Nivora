# Deployment Model

Nivora treats deployment as a workflow from Release intent to an Environment or Release Target. GitOps is one deployment mode, not the whole product.

## Core Concepts

- Release: versioned delivery intent, usually tied to immutable Artifacts.
- DeploymentRun: one execution of a Release or deployment plan.
- Environment: delivery context such as dev, staging, production, region, or tenant.
- Release Target: concrete target such as a host group, Kubernetes cluster, Argo CD application, cloud target, or webhook target.

## Deployment Modes

Future deployment modes may include:

- host deployment
- Kubernetes YAML deployment
- Helm
- Kustomize
- Argo CD GitOps release
- webhook deployment
- cloud-provider-specific deployment through CloudProvider Adapters

## Rollback

Rollback should be modeled as an auditable operation with a reason, target, prior version or Artifact, status, logs, and verification result.

## Deployment Diff and Health Verification

Future phases should expose deployment diff and health verification when the target system supports it. These features should be target-aware and should not assume Kubernetes or Argo CD is always present.

## Current State

Phase 0 defines deployment statuses and migration tables only. No production deployment logic is implemented.
