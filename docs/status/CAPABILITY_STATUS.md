# Nivora Capability Status

This status page is the public source of truth for implemented vs. partial vs. placeholder behavior after the implementation audit in `docs/status/IMPLEMENTATION_AUDIT.md`.

Current maturity: **hardened beta-candidate**.

Nivora is **not production-ready**. All 11 core runtime stores have PostgreSQL persistence with hash-chained audit (governance stores). Enterprise-grade runner isolation (workspace, env blocklist, process group cleanup), exhaustive RBAC tests (100+ sub-tests), and Helm safety verification are in place. Remaining blockers are operational validation (multi-process recovery e2e, production install smoke tests). External integrations remain local, noop, fake, skeleton, or experimental foundations.

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
| Shell executor | implemented | `internal/adapters/executor/shell`, 18 shell executor tests | Enterprise controls: timeout clamp (3600s max), output truncation (10MB), workspace isolation with cleanup, sensitive env blocklist (20+ patterns), process group cleanup (Setpgid+SIGKILL), env isolation mode. NOT a container/VM sandbox; operators must still isolate untrusted workloads. |
| Kubernetes YAML apply | experimental | `internal/adapters/executor/yaml_apply`, guarded deployment apply command/API | Apply and rollback require explicit confirmation; no default destructive behavior; production cluster semantics remain future hardening. |
| GitOps / Argo CD | experimental | `internal/adapters/gitops/local`, `internal/adapters/executor/argocd/noop.go`, GitOps/Argo CLI | Planning/status/guarded sync foundations only; no production Argo CD automation or Git provider integration. |
| Artifact / OCI digest | partial | `internal/domain/artifact`, `internal/adapters/artifact/oci` | OCI parsing and digest resolution foundation; full Harbor/Nexus/JFrog management APIs are not implemented. |
| DevSecOps scanner | foundation | `internal/usecase/security`, `internal/adapters/security/noop` | Noop/fake scanner and built-in manifest checks; Trivy/Cosign/SBOM integrations remain future work. |
| Policy gates | foundation | `internal/usecase/security`, `internal/ports/policy` | Built-in simple rules and allow-all placeholders exist; no OPA/Kyverno integration. |
| Secret management | partial | `internal/ports/secret`, builtin/Vault/Kubernetes/KMS adapters | SecretRef model, redaction, and provider skeletons exist; production provider lifecycle is not validated. |
| Auth/RBAC | partial | `internal/usecase/auth`, auth middleware tests, 100+ RBAC sub-tests (6 test functions covering 31 critical routes × 3 roles + cross-tenant isolation), `docs/security/ROUTE_PERMISSION_MATRIX.md` | Dev/token/OIDC foundation, route permission matrix, exhaustive route coverage tests, and cross-tenant isolation tests exist. Auth store has PostgreSQL persistence. Full enterprise SSO remains future work. |
| Approval/change window | foundation | `internal/usecase/approval`, approval/change-window API routes | Backend lifecycle foundation only; no full ITSM/workflow integration. |
| Multi-cloud | placeholder | cloud domain, provider ports, fake/AWS/Aliyun/Tencent skeletons | Inventory skeleton/fake behavior only; no cloud deployment or complete provider inventory. |
| Host deployment | experimental | host target models, noop host executor, SSH skeleton | Plan/dry-run/noop foundation; remote execution is disabled by default and guarded. |
| Web console | experimental | `web/` Vite React app, `make verify-web` | Minimal UI foundation consuming backend APIs; not a production UI. |
| Plugin system | foundation | `internal/usecase/plugin`, `api/proto/plugin.proto`, plugin templates | Manifest/capability registry and protocol skeleton; no unsafe dynamic loading or marketplace. |
| Packaging | partial | Dockerfile, Docker Compose, Helm, Kubernetes manifests, `deployments/helm/values-production.yaml`, `deployments/docker-compose/docker-compose.production.example.yaml`, production profile smoke scripts | Safer production-like profiles exist and avoid unsafe defaults, but no operator and no fully validated production installer. |
| Observability | partial | request/correlation middleware, `/metrics`, diagnostics endpoints | Lightweight metrics/diagnostics/runbooks; distributed tracing/export dashboards remain future work. |
| Audit/evidence | foundation | `internal/usecase/compliance`, audit/evidence routes, `internal/adapters/repository/postgres/compliance_store.go`, `000008_compliance_audit_evidence` migration | Evidence bundle and retention policy persistence exists in Postgres mode; complete hash-chained/tamper-evident audit writes remain future hardening. |
| Persistence | partial | All 11 stores have Postgres implementations under `internal/adapters/repository/postgres/`, migrations 000001-000009, CI postgres-integration job | All stores have PostgreSQL persistence with runtime `WithConfig` wiring. Governance stores have hash-chained audit. Runtime store audit paths use per-store tables without hash chaining. Multi-process recovery e2e not yet proven in CI. |
| Backup/restore | documented-only | `docs/operations/backup-restore.md`, HA/DR docs, optional smoke scripts | Procedures and optional smoke checks exist, but full restore drills are not automated or production-verified. |

## Contract Notes

- Placeholder route groups must return structured `not_implemented` responses.
- OpenAPI paths must match registered HTTP routes at route/path level.
- OpenAPI and AsyncAPI describe a beta-candidate foundation, not a production-ready platform.
- External adapters labeled noop, fake, skeleton, placeholder, or experimental must not be described as complete integrations.
