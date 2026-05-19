# Changelog

All notable changes to Nivora are tracked here.

Nivora is early-stage software. Releases before `1.0.0` may change APIs, configuration, and runtime behavior as the architecture hardens.

## v0.1.0-alpha.1 - Unreleased

This is the first public alpha foundation release. It is intended for contributors and platform engineers evaluating the architecture, not for production operation.

### Added

- Modular Go backend with `nivora-server`, `nivora-worker`, `nivora-runner`, and `nivora` CLI binaries.
- Architecture guardrails for domain, usecase, adapter, API, and infra boundaries.
- Minimal shell-based PipelineRun runtime with logs, events, audit records, timeline access, retries, timeout, and cancellation foundations.
- DeploymentRun foundations for Kubernetes YAML planning, dry-run, explicit guarded apply, resource inventory, health summaries, manifest snapshots, diff summaries, and rollback plan baselines.
- Artifact parsing, OCI digest resolution foundation, Release and ReleaseArtifact binding, and multi-target ReleasePlan / ReleaseExecution orchestration.
- GitOps and Argo CD planning/status/guarded sync foundations with sync disabled by default.
- DevSecOps policy gate foundation with noop/fake scanners, policy decisions, and security examples.
- SecretRef/Credential metadata, local development auth/RBAC, approval/change-window/notification foundations, cloud inventory skeletons, and host deployment planning/noop execution.
- Visualization backend APIs, minimal web UI foundation, observability diagnostics, plugin capability registry, Docker Compose, Helm chart, and Kubernetes packaging examples.
- Phase 5.1 PostgreSQL PipelineRun persistence foundation with runtime tables, ordered LogChunks, events, audit records, runner state, outbox records, idempotency keys, recovery queries, and explicit `database.runtime_store` configuration.

### Known Limitations

- Not production-ready.
- Persistence, scheduling, runner protocol, and worker coordination are still early foundations.
- Real Kubernetes production deployment semantics, destructive rollback, production Argo CD automation, cloud deployments, host SSH deployment, full registry integrations, Git provider integrations, SSO, Vault/KMS, external notifications, Trivy/Cosign/SBOM integrations, and ITSM integrations remain future work.
- Docker builds depend on external base image registries being reachable.

### Verification Target

The alpha release target is `make verify` plus optional packaging checks such as `make helm-template`, `make helm-lint`, and `make docker-build` when local tooling and registry access are available.
