# Nivora Documentation

This index helps new contributors understand where to start. Nivora is currently a **hardened beta-candidate foundation**, not a production-ready platform. Future `v1.0.0` documents are readiness checklists only until maintainers close the release blockers.

## Start Here

- [Project Charter](../PROJECT_CHARTER.md): what Nivora is, why it exists, and how it develops.
- [README](../README.md): public project overview, architecture diagrams, runtime model, roadmap, and local commands.
- [User Guide](user/README.md): pipeline, deployment, release, artifact, approval, and security workflows.
- [Operator Guide](operator/README.md): install, config, auth, secrets, backup, observability, and troubleshooting.
- [Developer Guide](developer/README.md): architecture, adapters, plugins, and tests.
- [Tutorials](tutorials/README.md): first pipeline, first deployment dry-run, first release, GitOps plan, and policy gate.
- [Roadmap](../ROADMAP.md): phase summary.
- [Alpha Capability Matrix](ALPHA_CAPABILITY_MATRIX.md): what is implemented, partial, planned, or unsupported in the alpha.
- [Beta Capability Matrix](BETA_CAPABILITY_MATRIX.md): beta-freeze capability boundaries.
- [Implementation Audit](status/IMPLEMENTATION_AUDIT.md): evidence-based maturity audit.
- [Capability Status](status/CAPABILITY_STATUS.md): implemented, partial, foundation, placeholder, experimental, documented-only, and missing capability labels.
- [MCP Control Plane Review](status/MCP_CONTROL_PLANE_REVIEW.md): safe AI-control-plane capability review and remote MCP go/no-go.
- [API Inventory](API_INVENTORY.md): implemented, partial, and placeholder HTTP API groups.
- [Alpha Demo Guide](demo/alpha-demo.md): self-contained demo path that does not require external services.
- [Contribution Guide](../CONTRIBUTING.md): development setup and contribution rules.
- [Contributor Automation Rules](../AGENTS.md): canonical coding instructions for automated and human-assisted changes.

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
- [Control Plane Catalog](concepts/control-plane-catalog.md)
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
- [Runner Design](architecture/runner-design.md)
- [Workflow Model](architecture/workflow-model.md)
- [Runtime Recovery](architecture/runtime-recovery.md)
- [Deployment Model](architecture/deployment-model.md)
- [Release Orchestration](architecture/release-orchestration.md)
- [Artifact Model](architecture/artifact-model.md)
- [GitOps Model](architecture/gitops-model.md)
- [Host Deployment Model](architecture/host-deployment-model.md)
- [Kubernetes Resource Model](architecture/kubernetes-resource-model.md)
- [Integration Model](architecture/integration-model.md)
- [Security Model](architecture/security-model.md)
- [Auth Model](architecture/auth-model.md)
- [Multi-Tenancy Model](architecture/multi-tenancy.md)
- [Approval Model](architecture/approval-model.md)
- [Cloud Provider Model](architecture/cloud-provider-model.md)
- [Secret Model](architecture/secret-model.md)
- [Plugin System](architecture/plugin-system.md)
- [MCP Control Plane](architecture/mcp-control-plane.md)
- [Visualization API](architecture/visualization-api.md)
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
- [v0.1.0-alpha.1 Checklist](releases/v0.1.0-alpha.1-checklist.md)
- [v0.5.0-beta Checklist](releases/v0.5.0-beta-checklist.md)
- [v0.5.0-beta Release Notes Draft](releases/v0.5.0-beta-release-notes-draft.md)
- [v1.0.0-rc.1 Checklist](releases/v1.0.0-rc.1-checklist.md)
- [Future v1.0.0 GA Capability Matrix](releases/v1.0.0-ga-capability-matrix.md)
- [Future v1.0.0 GA Checklist](releases/v1.0.0-ga-checklist.md)
- [Future v1.0.0 Release Notes Draft](releases/v1.0.0-release-notes.md)
- [Release Playbook](releases/release-playbook.md)

## Role-Based Guides

- [User Guide](user/README.md)
- [Operator Guide](operator/README.md)
- [Developer Guide](developer/README.md)
- [Tutorials](tutorials/README.md)

## Developer Checks

- [API Contract Checks](dev/api-contract.md)
- [Pipeline Definition Catalog](dev/pipeline-definitions.md)
- [MCP Tools](dev/mcp-tools.md)
- [Runtime Recovery Tests](dev/runtime-recovery-tests.md)
- [Testing Strategy](dev/testing-strategy.md)

## Operations Docs

- [Configuration](operations/configuration.md)
- [Production-Direction Install](operations/production-install.md)
- [Upgrade Guide](operations/upgrade.md)
- [Release Automation](operations/release-automation.md)
- [Docker Compose Install](operations/install-docker-compose.md)
- [Kubernetes Install](operations/install-kubernetes.md)
- [Cloud Providers](operations/cloud-providers.md)
- [Host Deployment](operations/host-deployment.md)
- [Observability Operations](operations/observability.md)
- [Performance and Load Testing](operations/performance.md)
- [Runbooks](operations/runbooks/stuck-pipelinerun.md)
- [OIDC Auth](operations/auth-oidc.md)
- [RBAC Operations](operations/rbac.md)
- [Quotas and Usage](operations/quotas.md)
- [Audit Evidence and Retention](operations/audit-evidence.md)
- [Production Doctor](operations/production-doctor.md)
- [Database Operations](operations/database.md)
- [Backup and Restore](operations/backup-restore.md)
- [HA and Disaster Recovery](operations/ha-disaster-recovery.md)
- [Vault Secret Provider](operations/secrets-vault.md)
- [KMS and External Secret Providers](operations/secrets-kms.md)
- [Artifact Registries](operations/artifact-registries.md)
- [Runtime Recovery Operations](operations/runtime-recovery.md)
- [Runner Fleet Operations](operations/runner-fleet.md)
- [Kubernetes Deployment Operations](operations/kubernetes-deployment.md)
- [GitOps and Argo CD Operations](operations/gitops-argocd.md)
- [Change Management](operations/change-management.md)
- [Troubleshooting](operations/troubleshooting.md)

## Security Review

- [Threat Model](security/threat-model.md)
- [Security Review Checklist](security/security-review-checklist.md)
- [MCP Permission Matrix](security/MCP_PERMISSION_MATRIX.md)
- [MCP Security](security/mcp-security.md)
- [Security Baseline](engineering/security-baseline.md)

## Community and Governance

- [Governance](community/governance.md)
- [Contribution Areas](community/contribution-areas.md)
- [Decision Making](community/decision-making.md)
- [Maintainer Guide](community/maintainer-guide.md)

## RFCs and ADRs

- [RFC Process](rfcs/README.md)
- [RFC Template](rfcs/0000-template.md)
- [Plugin RFC Template](rfcs/plugin-template.md)
- [Plugin API RFC](rfcs/plugin-api.md)
- [ADR Directory](adr/)

Use RFCs for large proposals before implementation. Use ADRs to record accepted architecture decisions.

## Engineering Guardrails

- [Developer Getting Started](dev/getting-started.md)
- [Testing Strategy](dev/testing-strategy.md)
- [Acceptance Tests](dev/acceptance-tests.md)
- [Quality Dashboard](dev/quality-dashboard.md)
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
- [Runner Protocol](dev/runner-protocol.md)
- [Runtime Troubleshooting](dev/troubleshooting.md)
- [Persistence Development](dev/persistence.md)
- [Security Scans](dev/security-scans.md)
- [Policy Gates](dev/policy-gates.md)
- [Secret Management](dev/secret-management.md)
- [Local Auth Mode](dev/auth-local-mode.md)
- [Approvals](dev/approvals.md)
- [Change Windows](dev/change-windows.md)
- [Notifications](dev/notifications.md)
- [Cloud Inventory](dev/cloud-inventory.md)
- [Host Deployment Dry-Run](dev/host-deployment-dry-run.md)
- [Visualization APIs](dev/visualization-apis.md)
- [Web UI](dev/web-ui.md)
- [Web Console](dev/web-console.md)
- [Writing Adapters](dev/writing-adapters.md)
- [Plugin Authoring](dev/plugin-authoring.md)
- [Dependency Policy](engineering/dependency-policy.md)
- [Testing Policy](engineering/testing-policy.md)
- [Security Baseline](engineering/security-baseline.md)
- [Release Scope](engineering/release-scope.md)
- [Documentation Style Guide](STYLE_GUIDE.md)
- [Documentation Inventory](DOCS_INVENTORY.md)
- [Optional Local Development Environment](dev/local-dev-environment.md)
