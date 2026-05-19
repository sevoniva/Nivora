# Beta Capability Matrix

Target: `v0.5.0-beta.1` readiness baseline

This matrix is the Phase 9.0 beta-freeze view of Nivora. It freezes feature expansion for review and separates beta-ready foundations from partial capabilities and future work. Nivora remains early-stage and is not production-ready or GA.

## Capability Status

| Area | Beta Freeze Status | What Is In Scope | Explicitly Not In Scope |
|---|---|---|---|
| PipelineRun | Beta foundation | Shell-based PipelineRun execution, YAML parsing, logs, events, audit, timeline, retry, timeout, cancellation, worker recovery, runner claim protocol, pagination, benchmarks | Advanced DSL, distributed workflow engine, privileged execution by default |
| Runner Fleet | Beta foundation | runner identity, hashed tokens, heartbeat, job claim, leases, log append, status update, offline detection, concurrency/label checks | Autoscaling, remote privilege management, untrusted sandboxing guarantees |
| DeploymentRun | Beta foundation | Kubernetes YAML planning, validation, dry-run, explicit guarded apply, logs/events/audit/timeline, resource inventory, health, diff, snapshot, rollback plan | Default apply, destructive prune/delete by default, GA Kubernetes controller behavior |
| Kubernetes YAML | Partial beta | static YAML render, resource inventory, health checks for common resources, rollout model, guarded rollback foundation | Helm, Kustomize, CRD health, production-grade rollback automation |
| Release Orchestration | Beta foundation | Release, ReleaseArtifact, ReleasePlan, ReleaseExecution, sequential multi-target orchestration, aggregate status | Release trains, promotion workflows, full approval workflow engine |
| Artifacts / OCI | Beta foundation | OCI parsing, digest resolution foundation, digest-bound releases, mutable-tag warnings, SecretRef-compatible credentials, explicit insecure registry config | Harbor/Nexus/JFrog admin APIs, signing, scanning, registry lifecycle management |
| GitOps / Argo CD | Partial beta | GitOps plan/diff/local working tree, Argo CD status model, guarded sync request path, sync disabled by default | Production sync automation, app creation, repo/cluster registration, full Git provider integration |
| DevSecOps | Partial beta | SecurityScan/Finding model, fake/noop scanners, built-in policy rules, policy gate decisions, audit/events | Required Trivy/Cosign/SBOM, OPA/Kyverno/Gatekeeper, vulnerability database operations |
| Secrets / Credentials | Partial beta | SecretRef/Credential metadata, builtin dev provider, rotation model, redaction, Vault/Kubernetes/KMS provider skeletons | Production Vault/KMS hardening, secret value APIs, enterprise credential lifecycle |
| Auth / RBAC | Partial beta | dev auth, static token mode, OIDC provider foundation, service accounts, API tokens, permission checks for critical routes | Browser login UI, full enterprise SSO lifecycle, final RBAC coverage guarantees |
| Governance | Partial beta | approvals, approval decisions, change windows, noop/log/webhook notification foundations, audit trail | ITSM integration, external delivery guarantees, approval UI |
| Multi-Tenancy / Quotas | Partial beta | tenant scope model, quota/usage views, runner/credential scope checks, concurrency foundations | billing, hard distributed quota enforcement, production rate limiting |
| Compliance Audit | Partial beta | structured audit model, audit search, evidence bundles, retention policy metadata, JSON/Markdown export | immutable external audit store, retention enforcement jobs, compliance certification |
| Cloud Inventory | Foundation | CloudAccount model, provider ports, fake inventory, AWS/Aliyun/Tencent skeletons, CredentialRef model | cloud deployment, required real cloud credentials, full provider inventory coverage |
| Host Deployment | Foundation | host target/group model, plan/dry-run/noop execution, guarded SSH adapter surface, health and rollback models | remote SSH by default, destructive cleanup, beta-grade host rollout execution |
| Visualization / Web | Foundation | backend visualization APIs and minimal web console foundation | complete frontend product, production UI claims |
| Observability / Performance | Beta foundation | request/correlation IDs, diagnostics, process metrics, SLO/runbook docs, benchmarks, load scripts, API pagination/limits | distributed tracing export, bundled dashboards, production SRE stack |
| Packaging / Operations | Beta foundation | Dockerfile, Compose, Helm chart, Kubernetes manifests, config examples, backup/restore/HA docs | operator, managed cloud install, production HA guarantee |
| Plugin System | Beta foundation | manifest model, capability registry, plugin API version, compatibility checks, protocol skeleton, templates | marketplace, unsafe dynamic loading, external plugin supervisor |

## Freeze Rules

- No new major feature areas should be added during the beta freeze.
- Changes should focus on consistency, tests, docs, examples, migration safety, and security review.
- Public docs must keep current capability and target architecture separate.
- Any new API/event must update OpenAPI or AsyncAPI.
- Any integration requiring credentials must use SecretRef/CredentialRef and must not require external services in baseline tests.

## Readiness Interpretation

- **Beta foundation** means the local backend workflow is testable and documented, but not GA production support.
- **Partial beta** means the model and guarded path exist, but production hardening remains.
- **Foundation** means useful for contributor validation and future adapter work, not production rollout.
