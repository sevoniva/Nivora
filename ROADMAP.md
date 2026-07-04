# Roadmap

This is the concise project roadmap. Detailed phase docs live in [docs/roadmap/overview.md](docs/roadmap/overview.md).

Current maturity: **hardened beta-candidate foundation**. Nivora is **not production-ready**. Future `v1.0.0` documents are readiness checklists and should not be read as evidence that GA has been achieved.

## Phase 0: Skeleton

- Establish Go module structure.
- Define control plane, worker, runner, and CLI binaries.
- Add domain models, ports, placeholder adapters, API specs, migrations, docs, CI, and local development setup.

## Phase 0.5: Guardrails and Validation

- Harden automated-change and contributor guardrails.
- Keep GOPROXY configurable for global contributors and China-based development.
- Enforce formatting, tidy, vet, tests, builds, architecture checks, and secret checks in CI.
- Polish local development verification scripts and Makefile targets.
- Keep placeholder APIs honest and structured without adding business logic.

## Phase 0.6: Planning and Collaboration Docs

- Add public planning docs, project charter, product vision, architecture blueprint, concept docs, roadmap docs, community docs, and RFC template.
- Keep current implementation clearly separated from target architecture.
- Improve contribution paths without adding Phase 1 business logic.

## Phase 1: Minimal Pipeline Execution

- Parse minimal Pipeline definitions and create PipelineRuns.
- Execute controlled shell steps.
- Capture logs, events, audit records, and run status.

## Phase 1.5: Durable Runtime Foundation

- Add explicit status transition helpers for PipelineRun, StageRun, JobRun, and StepRun.
- Add in-memory runtime repositories, ordered LogChunks, timeline APIs, minimal cancellation, retry, timeout, runner selection, and runner heartbeat.
- Keep runtime shell-only and avoid Phase 2 deployment integrations.

## Phase 1.6: Runtime Acceptance and Developer Experience

- Add runtime acceptance docs, smoke scripts, safer examples, CLI/API polish, and developer troubleshooting docs.
- Keep all tests self-contained without Kubernetes, Argo CD, cloud, Git provider, or registry dependencies.

## Phase 2: Release and Deployment Foundation

- Add release and deployment workflows.
- Phase 2.0 adds YAML deployment planning and non-destructive dry-run foundation.
- Phase 2.1 adds controlled Kubernetes YAML dry-run/apply runtime foundation with explicit local apply, resource inventory, rollout result modeling, and rollback baseline.
- Phase 2.2 through Phase 2.6 add artifact/release binding, GitOps planning, Kubernetes inventory/health, OCI digest resolution, and guarded Argo CD status/sync foundations.
- Phase 2.7 adds ReleasePlan and ReleaseExecution orchestration across multiple targets with sequential local execution.
- Future Phase 2 work adds durable approvals, environment locks, production health verification, and rollback execution.
- Keep production Kubernetes apply semantics, Helm, Kustomize, production Argo CD automation, cloud targets, and host SSH deployment out of Phase 2.7.

## Phase 3: Multi-Cloud and DevSecOps

- Phase 3.0 adds SecurityScan, SecurityFinding, noop/fake scanner adapters, and built-in policy gate foundations.
- Phase 3.1 adds SecretRef, Credential metadata, development secret provider, and redaction foundations.
- Phase 3.2 adds local AuthN/AuthZ and RBAC foundations while keeping OIDC/Keycloak future.
- Phase 3.3 adds ApprovalRequest, ApprovalDecision, ChangeWindow, NotificationProvider, and audit/event foundations for human governance.
- Phase 3.4 adds CloudAccount metadata, provider configuration, fake inventory adapters, and AWS/Aliyun/Tencent skeletons.
- Phase 3.5 adds HostTarget/HostGroup models, host deployment planning, noop execution, guarded SSH skeleton, and non-destructive rollback baselines.
- Phase 3.6 adds durable runtime and runner protocol foundations: job claims, leases, runner log/status APIs, cancel requests, worker outbox publishing, and event outbox schema.
- Add cloud provider adapters.
- Add artifact scanning and policy evaluation.
- Add secret backends and notification providers.
- Harden authn, authz, audit, and telemetry.
- Keep real notification integrations, ITSM, Trivy, Cosign, SBOM generation, cloud deployments, cloud scanning, production SSO, real remote host deployment, and production security automation optional and future until RFC-backed.

## Phase 4: Visualization Frontend

- Phase 4.0 adds visualization-ready backend APIs for PipelineRun DAGs, timelines, DeploymentRun resources/health/diff, release overviews, environment topology, runner/security summaries, and audit timelines.
- Phase 4.1 adds a minimal React + TypeScript + Vite web UI foundation that consumes existing visualization APIs.
- Phase 4.2 adds lightweight observability, diagnostics, and operations docs.
- Phase 4.3 adds plugin manifest and capability registry foundations.
- Phase 4.4 adds Docker Compose, Helm, Kubernetes manifests, configuration examples, and install docs.
- Future Phase 4 work can harden frontend surfaces on top of these backend contracts.

## Phase 5: Alpha Release Hardening

- Phase 5.0 prepares `v0.1.0-alpha.1` with a capability matrix, alpha demo guide, changelog, release checklist, version alignment, CI hardening, and documentation consistency review.
- Phase 5.1 hardens the PostgreSQL persistence foundation for PipelineRun runtime state, ordered logs, events, audit records, runners, outbox records, idempotency keys, recovery queries, deployment/release runtime state, catalog metadata, Pipeline definitions, artifact registry catalogs, and policy catalogs.
- The alpha is for public architecture review, contributor onboarding, and local demos.
- It is not a production GA release.

## Release Truth Alignment and Future GA Readiness

- Keep public maturity language aligned with the implementation audit.
- Treat future `v1.0.0` materials as readiness checklists until all blockers are closed by maintainers.
- Keep external integrations labeled as foundation, experimental, fake, noop, skeleton, or placeholder unless real behavior is implemented, tested, documented, and verifiable.
- Do not claim production readiness while persistence, recovery, RBAC coverage, runner isolation, audit durability, and production install validation remain incomplete.
