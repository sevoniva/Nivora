# Nivora Project Charter

Nivora is an open-source DevOps delivery control plane under the `sevoniva` organization. It is backend-first, written in Go, and currently developed as a modular monolith with separate binaries for `nivora-server`, `nivora-worker`, `nivora-runner`, and the `nivora` CLI.

Nivora is currently a beta-candidate foundation. It has backend, runtime, deployment, release, security, governance, visualization, operations, plugin, and packaging foundations, but it is not production-ready. Production Kubernetes deployment semantics, destructive rollback, production Argo CD automation, real cloud deployment, full Git provider and registry integrations, production SSO, external notification delivery, enterprise secret backends, durable audit/evidence, complete persistence, and complete frontend workflows remain future work.

## What Nivora Is

Nivora is a delivery control plane for CI/CD, GitOps, host deployment, Kubernetes deployment, multi-cloud delivery targets, artifact orchestration, policy gates, runners, approvals, release audit, and future visualization APIs.

The project is intended to coordinate delivery across existing tools rather than replace all of them. Git providers, artifact registries, Kubernetes, Argo CD, workflow engines, scanners, notification systems, object stores, and cloud providers should be integrated through stable Ports and Adapters.

## Why Nivora Exists

Modern delivery systems are fragmented. A single release can cross Git webhooks, CI runners, artifact registries, approvals, policy checks, deployment tools, cloud targets, logs, audit records, and incident response workflows. Those systems often have different concepts of state, identity, artifact identity, and audit.

Nivora exists to make delivery state explicit and auditable. It should provide one control surface for PipelineRuns, Releases, DeploymentRuns, Artifacts, Environments, Release Targets, Policies, Runners, and Audit without hiding the underlying tools behind opaque magic.

## Problems Addressed

- Delivery state is spread across many systems.
- Artifact identity is often mutable or implicit.
- Approval and policy gates are inconsistent between teams.
- Host deployment, Kubernetes deployment, and GitOps are often modeled as unrelated workflows.
- Runner behavior and logs are difficult to audit uniformly.
- Multi-cloud delivery targets lack a common inventory and release history.
- Platform teams need extension points without turning the core into a vendor-specific product.

## Target Users

- Platform engineers building internal delivery platforms.
- DevOps engineers standardizing delivery workflows.
- SREs who need release visibility, rollback context, and audit trails.
- Backend teams that need a reliable path from source to environment.
- Infrastructure architects evaluating delivery architecture across teams.
- Security reviewers who need policy gates, secret boundaries, and audit records.
- Open-source contributors building adapters, docs, tests, and core runtime features.
- AI coding agents working under strict architecture guardrails.

## Long-Term Vision

Nivora should become a practical, auditable delivery control plane that can coordinate pipelines and releases across heterogeneous infrastructure. It should support simple local execution for early development, then grow into controlled runner assignment, persisted workflow state, artifact-based releases, policy enforcement, deployment orchestration, audit, and visualization APIs.

The target architecture keeps the Control Plane separate from the Execution Plane. The Control Plane owns API, state, policy, audit, integration configuration, and workflow decisions. Runners execute assigned work through Executors and report logs, heartbeats, and status.

## Non-Goals

Nivora is not:

- a Jenkins clone
- an Argo CD replacement
- a Kubernetes-only tool
- a cloud-provider-specific product
- a frontend-first project
- production-ready in the current phase or tagged as GA without maintainer-approved blockers closed
- a system that hides every underlying tool behind opaque magic

## Architecture Principles

- Backend foundation first.
- Modular monolith first, service extraction later only after stable boundaries.
- Domain models do not depend on HTTP, database, queue, cloud, Kubernetes, Argo CD, or vendor SDKs.
- Ports define capabilities; Adapters implement integrations.
- Control Plane and Execution Plane are separated.
- Artifacts should be immutable.
- Policy and Audit are first-class concerns.
- GitOps is one deployment mode, not the whole product.
- Runners are a trust boundary and should be designed conservatively.

## Open-Source Collaboration Principles

- Keep changes small and reviewable.
- Prefer clear documentation before broad implementation.
- Use ADRs for architecture decisions.
- Use RFCs for large features, protocol changes, workflow runtime changes, database model changes, public API breaking changes, and security-sensitive changes.
- Do not add real integrations before the design is reviewed.
- Do not claim production readiness before the project reaches that phase.
- Keep AI agent instructions canonical in `AGENTS.md`.

## Phase-Based Development

- Phase 0: backend skeleton, module boundaries, binaries, configuration, domain structs, Ports, placeholder Adapters, migrations, docs, CI, local development.
- Phase 0.5: guardrails, architecture verification, secret checks, CI hardening, AI coding rules, Makefile verification.
- Phase 0.6: public planning docs, project charter, product vision, architecture blueprint, concept docs, roadmap docs, contribution model, RFC template.
- Phase 1: minimal PipelineRun execution, runner assignment, shell executor flow, log streaming, status transitions, audit event, minimal persistence.
- Phase 1.5: runtime foundation hardening, explicit state transitions, in-memory runtime repositories, worker advancement path, runner heartbeat, retry, timeout, cancellation, ordered LogChunks, and timeline APIs.
- Phase 1.6: runtime acceptance, developer experience, smoke scripts, example polish, CLI/API inspection, and troubleshooting docs.
- Phase 2: Release, DeploymentRun, Kubernetes YAML, GitOps, artifact binding, and release orchestration foundations.
- Phase 3: DevSecOps, secrets, auth/RBAC, approvals, multi-cloud inventory, host deployment, and durable runner protocol foundations.
- Phase 4: visualization APIs, minimal web UI, observability, plugin registry, and packaging foundations.
- Phase 5: public alpha hardening, release notes, capability matrix, demo path, and known limitations.
- Current truth-alignment status: beta-candidate foundation. Future `v1.0.0` documents are readiness checklists, not evidence that production GA has been achieved.
