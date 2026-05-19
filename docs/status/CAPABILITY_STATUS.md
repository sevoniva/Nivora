# Nivora Capability Status

This status page is the public source of truth for implemented vs. partial vs. placeholder behavior after the implementation audit in `docs/status/IMPLEMENTATION_AUDIT.md`.

Current maturity: **beta-candidate foundation**.

Nivora is **not production-ready**. Production readiness work is still in progress. Many external integrations are local, noop, fake, skeleton, or experimental foundations.

## Status Legend

| Status | Meaning |
|---|---|
| implemented | Implemented, tested, and verifiable for the documented local/foundation scope. |
| partial | Useful implementation exists, but important durability, safety, persistence, scale, or integration gaps remain. |
| foundation | Model, ports, adapters, examples, and tests exist to support future hardening, but the capability is not complete. |
| placeholder | Route, package, or document exists mainly to reserve shape and return structured not_implemented or skeleton behavior. |
| experimental | Available only for guarded evaluation or local validation; not production automation. |
| documented-only | Described in docs but not materially implemented. |
| missing | Not implemented and not usable. |

## Capability Matrix

| Capability | Status | Evidence | Notes / Limitations |
|---|---|---|---|
| PipelineRun runtime | implemented | `internal/usecase/pipeline`, `internal/adapters/executor/shell`, `scripts/smoke-pipelinerun-local.sh` | Shell-based local/foundation runtime; not a full distributed workflow engine. |
| DeploymentRun runtime | partial | `internal/usecase/deployment`, deployment API routes, smoke dry-run scripts, `internal/adapters/repository/postgres/deployment_store.go`, optional PostgreSQL recovery tests in `internal/adapters/repository/postgres/runtime_recovery_integration_test.go` | PostgreSQL persistence and restart-style repository recovery are covered when `NIVORA_RUN_POSTGRES_INTEGRATION=true`; production worker orchestration and full CD hardening remain incomplete. |
| ReleaseExecution | partial | `internal/usecase/releaseorchestration`, release API routes, `make verify-release`, `internal/adapters/repository/postgres/release_orchestration_store.go`, optional PostgreSQL recovery tests in `internal/adapters/repository/postgres/runtime_recovery_integration_test.go` | Sequential orchestration, PostgreSQL persistence, target state recovery, and stale execution queries exist; advanced recovery orchestration and approvals remain incomplete. |
| Runner protocol | partial | runner routes in `internal/api/http/routes/routes.go`, PipelineStore runner methods, runner-token scope tests | Token/heartbeat/claim/log/status foundations exist and runner tokens are scoped to protocol endpoints; sandboxing and fleet-scale behavior are not production-grade. |
| Shell executor | implemented | `internal/adapters/executor/shell`, shell executor tests | Executes local shell commands; operators must isolate untrusted workloads. |
| Kubernetes YAML apply | experimental | `internal/adapters/executor/yaml_apply`, guarded deployment apply command/API | Apply and rollback require explicit confirmation; no default destructive behavior; production cluster semantics remain future hardening. |
| GitOps / Argo CD | experimental | `internal/adapters/gitops/local`, `internal/adapters/executor/argocd/noop.go`, GitOps/Argo CLI | Planning/status/guarded sync foundations only; no production Argo CD automation or Git provider integration. |
| Artifact / OCI digest | partial | `internal/domain/artifact`, `internal/adapters/artifact/oci` | OCI parsing and digest resolution foundation; full Harbor/Nexus/JFrog management APIs are not implemented. |
| DevSecOps scanner | foundation | `internal/usecase/security`, `internal/adapters/security/noop` | Noop/fake scanner and built-in manifest checks; Trivy/Cosign/SBOM integrations remain future work. |
| Policy gates | foundation | `internal/usecase/security`, `internal/ports/policy` | Built-in simple rules and allow-all placeholders exist; no OPA/Kyverno integration. |
| Secret management | partial | `internal/ports/secret`, builtin/Vault/Kubernetes/KMS adapters | SecretRef model, redaction, and provider skeletons exist; production provider lifecycle is not validated. |
| Auth/RBAC | partial | `internal/usecase/auth`, auth middleware tests, `docs/security/ROUTE_PERMISSION_MATRIX.md` | Dev/token/OIDC foundation, selected permission checks, and route matrix exist; full enterprise SSO and exhaustive route coverage remain incomplete. |
| Approval/change window | foundation | `internal/usecase/approval`, approval/change-window API routes | Backend lifecycle foundation only; no full ITSM/workflow integration. |
| Multi-cloud | placeholder | cloud domain, provider ports, fake/AWS/Aliyun/Tencent skeletons | Inventory skeleton/fake behavior only; no cloud deployment or complete provider inventory. |
| Host deployment | experimental | host target models, noop host executor, SSH skeleton | Plan/dry-run/noop foundation; remote execution is disabled by default and guarded. |
| Web console | experimental | `web/` Vite React app, `make verify-web` | Minimal UI foundation consuming backend APIs; not a production UI. |
| Plugin system | foundation | `internal/usecase/plugin`, `api/proto/plugin.proto`, plugin templates | Manifest/capability registry and protocol skeleton; no unsafe dynamic loading or marketplace. |
| Packaging | partial | Dockerfile, Docker Compose, Helm, Kubernetes manifests | Templates and local install assets exist; no operator and no validated production installer. |
| Observability | partial | request/correlation middleware, `/metrics`, diagnostics endpoints | Lightweight metrics/diagnostics/runbooks; distributed tracing/export dashboards remain future work. |
| Audit/evidence | foundation | `internal/usecase/compliance`, audit/evidence routes | Evidence bundle/search model exists; durable tamper-evident audit storage is not complete. |
| Persistence | partial | Postgres PipelineStore, deployment/release runtime stores, migrations under `internal/infra/migration`, `make test-postgres-integration` | PipelineRun/runner/outbox plus DeploymentRun/Release/ReleaseExecution persistence foundations have optional real-PostgreSQL recovery tests; several governance/security stores remain memory-backed. |
| Backup/restore | documented-only | `docs/operations/backup-restore.md`, HA/DR docs | Procedures are documented but restore drills are not automated or verified. |

## Contract Notes

- Placeholder route groups must return structured `not_implemented` responses.
- OpenAPI paths must match registered HTTP routes at route/path level.
- OpenAPI and AsyncAPI describe a beta-candidate foundation, not a production-ready platform.
- External adapters labeled noop, fake, skeleton, placeholder, or experimental must not be described as complete integrations.
