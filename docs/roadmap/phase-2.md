# Phase 2: Release and Deployment Foundation

## Objective

Add release and deployment workflows suitable for early release modeling without claiming production readiness.

## Scope

- Phase 2.0 YAML deployment planning and dry-run foundation.
- Phase 2.1 controlled Kubernetes YAML dry-run/apply runtime foundation.
- Phase 2.2 artifact and release binding foundation.
- Static manifest rendering and validation.
- DeploymentPlan, events, audit, logs, timeline, and cancellation basics.
- Artifact reference parsing, immutability warnings, ReleaseArtifact binding, and manifest image verification.
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
- Multi-cloud provider expansion.
- Full DevSecOps platform.
- Visualization frontend.

## Expected Deliverables

Release and DeploymentRun workflows that can model GitOps and non-GitOps deployment modes. Phase 2.2 specifically delivers safe YAML planning/dry-run behavior, explicit no-op local apply, resource inventory, rollout result modeling, rollback baseline, artifact reference parsing, ReleaseArtifact binding, and manifest image verification.

## Acceptance Criteria

- GitOps remains one deployment mode.
- YAML deployment dry-run can run without a Kubernetes cluster.
- Apply is explicit and never default.
- ReleaseArtifacts are explicit and auditable.
- Deployment plans surface mutable artifact warnings.
- DeploymentRun audit is complete enough for rollback analysis.
- Approvals and environment locks are explicit.

## Contribution Opportunities

- Deployment state model.
- Renderer design.
- Artifact reference parsing and registry adapter design.
- Argo CD RFC.
- Approval and lock tests.
