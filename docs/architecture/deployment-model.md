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

Supported or modeled deployment modes include:

- host deployment foundation through `target.type=host`
- Kubernetes YAML deployment beyond dry-run planning
- Helm
- Kustomize
- Argo CD GitOps release
- webhook deployment
- cloud-provider-specific deployment through CloudProvider Adapters

## Phase 2.3 GitOps Foundation

Phase 2.3 adds `argocd` target planning. Nivora can build a GitOpsChangePlan, optionally update a local working tree, model Argo CD application status through a noop provider, and record guarded sync requests. Sync is disabled by default and requires explicit confirmation.

Nivora does not replace Argo CD. Argo CD remains the future reconciliation system for GitOps delivery, while Nivora coordinates release intent, artifact traceability, policy, audit, and timelines around it.

## Rollback

Rollback should be modeled as an auditable operation with a reason, target, prior version or Artifact, status, logs, and verification result.

## Deployment Diff and Health Verification

Phase 2.1 produces a stable desired-state summary and a diff placeholder. Live cluster diff is future work. Rollout verification is modeled through the manifest client port and defaults to a no-op local result.

## Artifact Binding

Phase 2.2 connects DeploymentPlan output to ReleaseArtifacts. Deployment specs may include inline artifact references, and the planner records normalized artifact summaries, immutability warnings, and simple manifest image verification warnings.

This is verify-first behavior. Nivora does not mutate manifests by default, does not resolve registry digests over the network, and does not claim full registry integration.

## Release Orchestration

Phase 2.7 adds ReleasePlan and ReleaseExecution as aggregate release-control-plane records. A ReleasePlan selects an Environment and multiple ReleaseTargets, creates one DeploymentPlan per executable target, records policy results, and preserves target ordering. A ReleaseExecution then runs targets sequentially through DeploymentRuns or safe placeholder targets and aggregates status.

DeploymentRun remains the target-level execution object. ReleaseExecution does not replace target logs, health, snapshots, diff, or rollback plans; it links and summarizes them.

## Phase 3.5 Host Deployment Foundation

Phase 3.5 introduces `host` as a ReleaseTarget type. A host DeploymentRun builds a HostDeploymentPlan with per-host release directories, symlink switch paths, health check metadata, and a non-destructive rollback baseline. The default runtime uses a noop HostExecutor so tests and examples do not mutate local or remote machines.

Remote SSH execution is not enabled by default. A future real adapter must require explicit apply confirmation, `allowRemoteHostDeploy`, and a CredentialRef or SecretRef-backed credential.

## Current State

Phase 3.5 supports YAML deployment planning, dry-run, explicit local no-op apply, resource inventory, lightweight health evaluation, manifest snapshots, desired-state diff summaries, rollback plan baseline, artifact summaries, GitOps planning, guarded Argo CD status/sync modeling, manifest image verification, sequential ReleaseExecution orchestration, and host deployment planning/noop execution only. Production Kubernetes apply semantics, destructive rollback, Helm, Kustomize, production Argo CD sync, cloud provider deployment, real remote SSH deployment, registry integrations, image signing, and scanning remain future work.
