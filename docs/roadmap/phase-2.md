# Phase 2: Release and Deployment Foundation

## Objective

Add release and deployment workflows suitable for early release modeling without claiming production readiness.

## Scope

- Phase 2.0 YAML deployment planning and dry-run foundation.
- Phase 2.1 controlled Kubernetes YAML dry-run/apply runtime foundation.
- Static manifest rendering and validation.
- DeploymentPlan, events, audit, logs, timeline, and cancellation basics.
- Future YAML apply, Helm, and Kustomize rendering design.
- Future Argo CD Adapter design.
- Approval gates.
- Environment locks.
- Deployment diff.
- Rollback records.
- Release audit.

## Non-Goals

- Production Kubernetes apply semantics in Phase 2.1.
- Argo CD implementation in Phase 2.1.
- Helm or Kustomize execution in Phase 2.1.
- Multi-cloud provider expansion.
- Full DevSecOps platform.
- Visualization frontend.

## Expected Deliverables

Release and DeploymentRun workflows that can model GitOps and non-GitOps deployment modes. Phase 2.1 specifically delivers safe YAML planning/dry-run behavior, explicit no-op local apply, resource inventory, rollout result modeling, and rollback baseline.

## Acceptance Criteria

- GitOps remains one deployment mode.
- YAML deployment dry-run can run without a Kubernetes cluster.
- Apply is explicit and never default.
- DeploymentRun audit is complete enough for rollback analysis.
- Approvals and environment locks are explicit.

## Contribution Opportunities

- Deployment state model.
- Renderer design.
- Argo CD RFC.
- Approval and lock tests.
