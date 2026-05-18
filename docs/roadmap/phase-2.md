# Phase 2: GitOps and Production Release Basics

## Objective

Add release and deployment workflows suitable for early production release modeling.

## Scope

- YAML, Helm, and Kustomize rendering design.
- Argo CD Adapter design.
- Approval gates.
- Environment locks.
- Deployment diff.
- Rollback records.
- Release audit.

## Non-Goals

- Multi-cloud provider expansion.
- Full DevSecOps platform.
- Visualization frontend.

## Expected Deliverables

Release and DeploymentRun workflows that can model GitOps and non-GitOps deployment modes.

## Acceptance Criteria

- GitOps remains one deployment mode.
- DeploymentRun audit is complete enough for rollback analysis.
- Approvals and environment locks are explicit.

## Contribution Opportunities

- Deployment state model.
- Renderer design.
- Argo CD RFC.
- Approval and lock tests.

