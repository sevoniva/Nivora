# Nivora Documentation

This index helps new contributors understand where to start. Nivora has completed the backend skeleton, guardrails, public planning docs, and an initial shell-only PipelineRun runtime foundation. It is still early-stage and not production-ready.

## Start Here

- [Project Charter](../PROJECT_CHARTER.md): what Nivora is, why it exists, and how it develops.
- [README](../README.md): public project overview, architecture diagrams, runtime model, roadmap, and local commands.
- [Roadmap](../ROADMAP.md): phase summary.
- [Contribution Guide](../CONTRIBUTING.md): development setup and contribution rules.
- [AI Agent Rules](../AGENTS.md): canonical coding instructions for AI agents.

## What To Read First

For a quick path through the project:

1. Read the [README](../README.md) for value, architecture, runtime, and roadmap.
2. Read the [Project Charter](../PROJECT_CHARTER.md) for project purpose and non-goals.
3. Read [Architecture Contract](architecture/architecture-contract.md) and [Module Boundaries](architecture/module-boundaries.md) before changing code.
4. Read [Phase 1](roadmap/phase-1.md) for the current shell-based runtime foundation.
5. Read the [RFC Process](rfcs/README.md) before proposing architecture-sensitive changes.

## Product Docs

- [Vision](product/vision.md)
- [Problem Statement](product/problem-statement.md)
- [Personas](product/personas.md)
- [Use Cases](product/use-cases.md)
- [Non-Goals](product/non-goals.md)
- [Product Principles](product/product-principles.md)

## Concept Docs

- [Concept Overview](concepts/overview.md)
- [Glossary](concepts/glossary.md)
- [Application](concepts/application.md)
- [Environment](concepts/environment.md)
- [Release Target](concepts/release-target.md)
- [Pipeline](concepts/pipeline.md)
- [PipelineRun](concepts/pipeline-run.md)
- [Release](concepts/release.md)
- [DeploymentRun](concepts/deployment-run.md)
- [Runner](concepts/runner.md)
- [Executor](concepts/executor.md)
- [Artifact](concepts/artifact.md)
- [GitOps](concepts/gitops.md)
- [Host Target](concepts/host-target.md)
- [Policy](concepts/policy.md)
- [Approval](concepts/approval.md)
- [Change Window](concepts/change-window.md)
- [Cloud Provider](concepts/cloud-provider.md)
- [RBAC](concepts/rbac.md)
- [Secret](concepts/secret.md)
- [Credential](concepts/credential.md)
- [Audit](concepts/audit.md)

## Architecture Docs

- [Architecture Contract](architecture/architecture-contract.md)
- [Module Boundaries](architecture/module-boundaries.md)
- [Target Architecture](architecture/target-architecture.md)
- [System Context](architecture/system-context.md)
- [Control Plane](architecture/control-plane.md)
- [Runner and Executor](architecture/runner-and-executor.md)
- [Workflow Model](architecture/workflow-model.md)
- [Deployment Model](architecture/deployment-model.md)
- [Release Orchestration](architecture/release-orchestration.md)
- [Artifact Model](architecture/artifact-model.md)
- [GitOps Model](architecture/gitops-model.md)
- [Host Deployment Model](architecture/host-deployment-model.md)
- [Kubernetes Resource Model](architecture/kubernetes-resource-model.md)
- [Integration Model](architecture/integration-model.md)
- [Security Model](architecture/security-model.md)
- [Auth Model](architecture/auth-model.md)
- [Approval Model](architecture/approval-model.md)
- [Cloud Provider Model](architecture/cloud-provider-model.md)
- [Secret Model](architecture/secret-model.md)
- [Policy Gates](architecture/policy-gates.md)
- [Observability Model](architecture/observability-model.md)
- [Data Model](architecture/data-model.md)
- [Extensibility Model](architecture/extensibility-model.md)

## Roadmap Docs

- [Roadmap Overview](roadmap/overview.md)
- [Phase 0](roadmap/phase-0.md)
- [Phase 0.5](roadmap/phase-0.5.md)
- [Phase 0.6](roadmap/phase-0.6.md)
- [Phase 1](roadmap/phase-1.md)
- [Phase 2](roadmap/phase-2.md)
- [Phase 3](roadmap/phase-3.md)
- [Phase 4](roadmap/phase-4.md)

## Community and Governance

- [Governance](community/governance.md)
- [Contribution Areas](community/contribution-areas.md)
- [Decision Making](community/decision-making.md)
- [Maintainer Guide](community/maintainer-guide.md)

## RFCs and ADRs

- [RFC Process](rfcs/README.md)
- [RFC Template](rfcs/0000-template.md)
- [ADR Directory](adr/)

Use RFCs for large proposals before implementation. Use ADRs to record accepted architecture decisions.

## Engineering Guardrails

- [Developer Getting Started](dev/getting-started.md)
- [Local PipelineRun Development](dev/local-pipeline-run.md)
- [YAML Deployment Dry-Run](dev/deployment-yaml-dry-run.md)
- [Kubernetes YAML Runtime](dev/kubernetes-yaml-runtime.md)
- [Local YAML Apply](dev/deployment-yaml-apply-local.md)
- [Artifact and Release Binding](dev/artifact-release-binding.md)
- [GitOps Planning](dev/gitops-plan.md)
- [Argo CD Local Validation](dev/argocd-local-validation.md)
- [Argo CD Status](dev/argocd-status.md)
- [Argo CD Guarded Sync](dev/argocd-guarded-sync.md)
- [Kubernetes Resource Inventory](dev/kubernetes-resource-inventory.md)
- [Kubernetes Health](dev/kubernetes-health.md)
- [Rollback Plan](dev/rollback-plan.md)
- [OCI Digest Resolution](dev/oci-digest-resolution.md)
- [Local Harbor Validation](dev/local-harbor-validation.md)
- [Multi-Target Release Development](dev/multi-target-release.md)
- [Runtime Acceptance Matrix](dev/runtime-acceptance.md)
- [Runtime Troubleshooting](dev/troubleshooting.md)
- [Security Scans](dev/security-scans.md)
- [Policy Gates](dev/policy-gates.md)
- [Secret Management](dev/secret-management.md)
- [Local Auth Mode](dev/auth-local-mode.md)
- [Approvals](dev/approvals.md)
- [Change Windows](dev/change-windows.md)
- [Notifications](dev/notifications.md)
- [Cloud Inventory](dev/cloud-inventory.md)
- [Host Deployment Dry-Run](dev/host-deployment-dry-run.md)
- [AI Change Policy](engineering/ai-change-policy.md)
- [Dependency Policy](engineering/dependency-policy.md)
- [Testing Policy](engineering/testing-policy.md)
- [Security Baseline](engineering/security-baseline.md)
- [Release Scope](engineering/release-scope.md)
- [Documentation Style Guide](STYLE_GUIDE.md)
- [Documentation Inventory](DOCS_INVENTORY.md)
- [Optional Local Development Environment](dev/local-dev-environment.md)
