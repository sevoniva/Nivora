# Phase 2: Release and Deployment Foundation

## Objective

Add release and deployment workflows suitable for early release modeling without claiming production readiness.

## Scope

- Phase 2.0 YAML deployment planning and dry-run foundation.
- Phase 2.1 controlled Kubernetes YAML dry-run/apply runtime foundation.
- Phase 2.2 artifact and release binding foundation.
- Phase 2.3 GitOps and Argo CD foundation.
- Phase 2.4 Kubernetes resource inventory, health, snapshots, diff, and rollback plan foundation.
- Phase 2.5 OCI artifact digest resolution foundation.
- Phase 2.6 Argo CD status and guarded sync hardening.
- Phase 2.7 release orchestration across targets.
- Static manifest rendering and validation.
- DeploymentPlan, events, audit, logs, timeline, and cancellation basics.
- Artifact reference parsing, immutability warnings, ReleaseArtifact binding, and manifest image verification.
- GitOpsChangePlan, local working tree planning, noop Argo CD status/sync model, and guarded sync semantics.
- Manifest snapshots, desired resource inventory, lightweight health, and non-destructive rollback plans.
- Generic OCI digest resolution, Harbor-compatible registry configuration, and digest-required ReleaseArtifact binding.
- Argo CD application status/resources, guarded sync requests, and limited status watch through a noop provider.
- ReleasePlan and ReleaseExecution foundation for sequential multi-target orchestration.
- Future YAML apply, Helm, and Kustomize rendering design.
- Future Argo CD Adapter design.
- Approval gates.
- Environment locks.
- Deployment diff.
- Rollback records.
- Release audit.

## Non-Goals

- Production Kubernetes apply semantics in Phase 2.
- Argo CD implementation in Phase 2.2.
- Helm or Kustomize execution in Phase 2.2.
- Full Harbor, Nexus, JFrog, or cloud registry integration in Phase 2.2.
- Artifact scanning, signing, or SBOM verification in Phase 2.2.
- Production Argo CD sync automation in Phase 2.3.
- Remote Git provider authentication, commit, and push in Phase 2.3.
- Destructive rollback execution in Phase 2.4.
- Full Kubernetes controller behavior or CRD health in Phase 2.4.
- Harbor management APIs, Nexus/JFrog management APIs, or cloud registry adapters in Phase 2.5.
- Vulnerability scanning, signing, Cosign, and SBOM workflows in Phase 2.5.
- Production-grade Argo CD automation, app management, SSO/RBAC integration, or multi-cluster GitOps in Phase 2.6.
- Heavy workflow engines, production-grade parallel orchestration, host SSH deployment, and cloud target execution in Phase 2.7.
- Multi-cloud provider expansion.
- Full DevSecOps platform.
- Visualization frontend.

## Expected Deliverables

Release and DeploymentRun workflows that can model GitOps and non-GitOps deployment modes. Phase 2.7 specifically includes safe YAML planning/dry-run behavior, explicit no-op local apply, artifact reference parsing, generic OCI digest resolution, ReleaseArtifact binding, manifest image verification, GitOps planning, guarded Argo CD status/sync modeling, resource inventory, lightweight health evaluation, manifest snapshots, desired-state diff summaries, non-destructive rollback plans, and a ReleasePlan/ReleaseExecution orchestration foundation across multiple targets.

## Acceptance Criteria

- GitOps remains one deployment mode.
- YAML deployment dry-run can run without a Kubernetes cluster.
- Apply is explicit and never default. Phase 6.0 adds a guarded apply/rollback hardening path for Kubernetes YAML without enabling destructive defaults.
- ReleaseArtifacts are explicit and auditable.
- Deployment plans surface mutable artifact warnings.
- Digest-required releases fail when a digest cannot be provided or resolved.
- GitOps sync is disabled by default and guarded.
- Argo CD force sync is rejected in the current phase.
- ReleaseExecution aggregates target DeploymentRun status.
- Sequential multi-target release execution works without external services.
- Secret values are never stored in resource inventory or snapshots.
- Rollback plans are non-destructive by default. Guarded manifest-restore rollback requires explicit confirmation and does not prune/delete resources by default.
- DeploymentRun audit is complete enough for rollback analysis.
- Approvals and environment locks are explicit.

## Contribution Opportunities

- Deployment state model.
- Renderer design.
- Artifact reference parsing and registry adapter design.
- Argo CD RFC.
- GitOps working tree tests.
- Approval and lock tests.
