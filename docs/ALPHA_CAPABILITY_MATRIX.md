# Alpha Capability Matrix

Version target: `v0.1.0-alpha.1`

Nivora is in public alpha. This matrix separates implemented foundations from partial work, planned work, and intentionally unsupported capabilities. It should be read as an honest release-readiness guide, not as a production support statement.

| Area | Alpha Status | What Exists | Not Yet Included |
|---|---|---|---|
| PipelineRun | Implemented foundation | Minimal YAML spec, local shell executor, status transitions, logs, events, audit, timeline, retry, timeout, cancellation, API/CLI smoke path | Durable distributed scheduling, remote runner isolation, advanced DSL |
| DeploymentRun | Implemented foundation | YAML deployment planning, dry-run, explicit guarded apply model, logs, events, audit, timeline | Production apply semantics, destructive rollback, environment locking |
| Release | Implemented foundation | Release, ReleaseArtifact, ReleasePlan, ReleaseExecution, sequential multi-target orchestration | Promotion workflows, durable approvals across environments, release trains |
| Artifact | Implemented beta foundation | OCI reference parsing, digest pin recognition, mutable tag warnings, generic OCI/Harbor-compatible digest resolution, registry credential refs, explicit insecure registry config, release digest binding | Full Harbor/Nexus/JFrog management APIs, signing, scanning, registry CRUD |
| GitOps | Partial | GitOps plan, diff, local working tree adapter, Argo CD target/status/guarded sync model | Production Git provider integration, repository credentials workflow, production sync automation |
| Kubernetes YAML | Partial | Static YAML render/validation, resource inventory, health summaries, manifest snapshots, rollback plan baseline | Helm, Kustomize, CRD health, production rollback execution |
| DevSecOps | Partial | SecurityScan/Finding model, noop/fake scanner, simple policy gate rules, examples | Required Trivy/Cosign/SBOM integrations, OPA/Kyverno/Gatekeeper integration |
| Secret/Credential | Partial | SecretRef/Credential metadata, development secret provider, rotation API/CLI, provider validation, external provider skeletons, redaction utilities, value-env CLI pattern | Production Vault/KMS storage, enterprise credential governance |
| Auth/RBAC | Partial | Local dev auth, static token mode, service accounts, hashed API tokens, OIDC provider/config foundation, RBAC model/evaluator, middleware foundations | Password login, frontend SSO flow, provider lifecycle management, full production identity operations |
| Multi-tenancy/Quotas | Partial | Scoped subjects, credential scope filtering, runner group scope checks, quota and usage read models | Persistent distributed quota counters, billing, full tenant provisioning, production rate limiting |
| Compliance Audit/Evidence | Partial | Structured AuditLog fields, audit search, evidence bundle JSON/Markdown export, retention policy metadata | Immutable external audit store, retention enforcement jobs, compliance attestations |
| Approval | Partial | ApprovalRequest/Decision, change window, notification provider foundations | ITSM integration, external notification delivery guarantees, UI approvals |
| Cloud Inventory | Partial | CloudAccount model, CloudProvider port, fake AWS/Aliyun/Tencent inventory, skeleton adapters | Real cloud inventory APIs, cloud deployment, cloud security scanning |
| Host Deployment | Partial | HostTarget/HostGroup model, host plan, noop/local execution, guarded SSH skeleton | Default remote SSH execution, service managers, remote rollback execution |
| Visualization | Partial | Backend visualization APIs and minimal Vite/React web foundation | Complete product UI, charts, auth-integrated dashboards |
| Packaging | Implemented foundation | Multi-binary Dockerfile, Docker Compose, Helm chart, raw Kubernetes examples, config docs | Operator, production HA chart, managed cloud install paths |
| Plugin System | Implemented foundation | Plugin manifest model, built-in capability registry, external protocol skeleton | Marketplace, unsafe dynamic loading, external plugin process manager |

## Release Readiness Interpretation

- **Implemented foundation** means the feature has local tests, examples, and a contributor-facing workflow.
- **Partial** means the model and safe local behavior exist, but production-grade integration or operation remains future work.
- **Planned** means the concept is documented but should not be relied on yet.
- **Not implemented** means no runtime behavior should be assumed.

The alpha is credible for local demos and architecture review. It is not a production deployment platform.
