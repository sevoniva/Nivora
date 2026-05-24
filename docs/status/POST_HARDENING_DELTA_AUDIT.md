# Nivora Post-Hardening Delta Audit

## Subsequent Hardening Note

As of 2026-05-19, production install and audit durability hardening added `deployments/helm/values-production.yaml`, `deployments/docker-compose/docker-compose.production.example.yaml`, optional production-profile smoke scripts, stricter production config validation, and a PostgreSQL-backed compliance evidence/retention store foundation. This improves the packaging/audit blockers called out below, but Nivora remains **not production-ready** because full restore drills, exhaustive governance/security persistence, and uniform tamper-evident audit writes are still incomplete.

## 1. Executive Summary

The three recent hardening goals materially improved Nivora's production direction, but they did not make the project production-ready.

Resolved or mostly resolved blockers:

- Public maturity language is now mostly aligned around **beta-candidate foundation** and **not production-ready** in `README.md`, `PROJECT_CHARTER.md`, `ROADMAP.md`, `api/openapi/openapi.yaml`, `api/asyncapi/asyncapi.yaml`, and `docs/status/CAPABILITY_STATUS.md`.
- OpenAPI route coverage and placeholder route behavior now have automated tests in `internal/api/http/routes/openapi_contract_test.go`.
- DeploymentRun, DeploymentPlan, deployment resources, snapshots, rollback plans, deployment logs/events/audit, Release, ReleaseArtifact, ReleasePlan, and ReleaseExecution now have PostgreSQL repository foundations and migration coverage.
- Route permission intent is documented in `docs/security/ROUTE_PERMISSION_MATRIX.md`.
- Critical RBAC, runner token scope, token/hash response safety, and production unsafe-default validation tests exist.
- Runner trust boundary and operational security docs exist in `docs/security/runner-trust-boundary.md` and `docs/operations/runner-security.md`.

Remaining blockers:

- Several governance, security, compliance, approval, credential, cloud, notification, and tenancy stores still default to memory-backed foundations.
- Postgres persistence tests are mostly interface/migration/unit-oriented; real database up/down and restart-recovery integration tests are still limited.
- RBAC coverage is improved but not exhaustive across every route, scope, tenant, and future integration path.
- Runner workload isolation is documented but not enforced as a sandbox.
- Audit/evidence is not yet uniformly durable or tamper-evident.
- Helm/package metadata still carries `1.0.0` scaffolding, so public release messaging needs continued care.

Current honest maturity label: **beta-candidate**.

Recommendation: continue hardening before broad feature expansion. The next work should focus on DB-backed recovery tests, exhaustive RBAC/tenant coverage, runner isolation policy, and durable audit/evidence rather than new integrations.

## 2. Verification Results

| Check | Result | Notes |
|---|---|---|
| `git status --short` | passed | Clean before creating this delta audit. |
| `git log --oneline -n 20` | passed | Recent history includes `d78b673 chore: harden security and runner boundaries`, `1c658f5 feat: persist deployment and release runtime`, and `9b80503 docs: align maturity claims and api contract`. |
| `go mod tidy` | passed | Ran successfully; no module diff after verification. |
| `make fmt-check` | passed | Used as the available gofmt check. |
| `go test ./...` | passed | All Go packages passed. Many skeleton/domain packages still have no tests. |
| `go vet ./...` | passed | No vet findings. |
| `go build ./cmd/nivora-server` | passed | Built successfully. |
| `go build ./cmd/nivora-worker` | passed | Built successfully. |
| `go build ./cmd/nivora-runner` | passed | Built successfully. |
| `go build ./cmd/nivora` | passed | Built successfully. |
| `./scripts/verify-architecture.sh` | passed | Architecture boundary checks passed. |
| `./scripts/verify-no-secrets.sh` | passed | No high-confidence secret patterns found. |
| `make verify` | passed | Ran tidy check, vet, tests, builds, architecture/secret checks, examples validation, API/CLI smoke, deployment/release/security/host smoke, web build, Helm template/lint, and release-doc checks. |
| `git diff --check` | passed | Passed before report creation. |

## 3. Previous Blocker Resolution Matrix

| Blocker | Status | Evidence | Remaining Gap | Recommendation |
|---|---|---|---|---|
| GA / maturity language overclaim | mostly resolved | `README.md`, `PROJECT_CHARTER.md`, `ROADMAP.md`, `docs/status/CAPABILITY_STATUS.md` all state beta-candidate / not production-ready. | Helm chart/package/version scaffolding still uses `1.0.0`; future GA docs still require careful interpretation. | Keep release docs explicit that GA is future readiness, not achieved state. |
| OpenAPI / AsyncAPI / README status drift | resolved | `api/openapi/openapi.yaml` and `api/asyncapi/asyncapi.yaml` use current pre-GA maturity wording and not-production-ready descriptions. | Versioned package artifacts can still confuse readers. | Keep specs aligned during every API change. |
| OpenAPI route coverage drift | mostly resolved | `internal/api/http/routes/openapi_contract_test.go` compares registered chi routes with OpenAPI paths/methods. | Semantic schema/permission validation is not checked. | Add deeper operation/security contract checks later. |
| Placeholder route clarity | resolved | `TestOpenAPIPlaceholderRouteLabelsMatchRouter` and `TestAllPlaceholderRoutesReturnStructuredNotImplemented`. | Future placeholders must be added to the placeholder group list. | Keep placeholder tests mandatory in CI. |
| DeploymentRun persistence | mostly resolved | `internal/adapters/repository/postgres/deployment_store.go`, `internal/infra/migration/000007_deployment_release_runtime.up.sql`. | Limited real DB integration/restart tests. | Add disposable Postgres repository and recovery tests. |
| DeploymentPlan / resources / logs / events / audit persistence | mostly resolved | `runtime_deployment_resources`, `runtime_manifest_snapshots`, `runtime_rollback_plans`, `runtime_deployment_logs`, `runtime_deployment_events`, `runtime_deployment_audit_logs`; append/get methods in deployment store. | Store foundation exists, but production recovery paths are not fully exercised. | Add end-to-end DB-backed deployment smoke tests. |
| Release persistence | mostly resolved | `internal/adapters/repository/postgres/release_store.go`, `runtime_releases`. | Release API idempotency and DB-backed integration tests remain shallow. | Add release create/get/list integration tests. |
| ReleaseArtifact persistence | mostly resolved | `runtime_release_artifacts`, release store writes bindings. | No external registry test dependency by design. | Add fake registry digest-binding persistence tests. |
| ReleasePlan persistence | mostly resolved | `internal/adapters/repository/postgres/release_orchestration_store.go`, `runtime_release_plans`. | Recovery from persisted plan is not proven end-to-end. | Add ReleasePlan recovery tests. |
| ReleaseExecution persistence | mostly resolved | `runtime_release_executions`, targets/events/audit tables, `ListNonTerminalReleaseExecutions`, `ListStaleReleaseExecutions`. | Restart reconciliation remains limited. | Add worker recovery loop tests for release execution. |
| Runtime wiring uses Postgres mode when configured | mostly resolved | `internal/app/runtime/runtime.go` creates Postgres stores for pipeline, deployment, artifact/release, and release orchestration when `database.runtime_store: postgres`. | Security/governance/compliance stores still memory-backed. | Extend persistence only where production use requires it. |
| Production config rejects unsafe memory/dev defaults | mostly resolved | `internal/infra/config/config.go`, `TestProductionRejectsMemoryRuntimeStore`, `TestProductionRejectsUnsafeSecurityDefaults`. | Helm values still render memory runtime store for chart defaults. | Add production Helm values and validation docs/tests. |
| RBAC route permission matrix | resolved | `docs/security/ROUTE_PERMISSION_MATRIX.md`. | Matrix can drift from code unless maintained. | Add matrix review to route-change checklist. |
| Critical route RBAC tests | mostly resolved | `TestCriticalRoutesRequirePermissionInOIDCMode`, `TestRouteRBACAllowsSufficientPermission`, `TestAuditorCanReadAuditButCannotMutate`. | Not every route/role/scope pair is tested. | Add table-driven permission tests generated from the matrix. |
| Runner token scope | mostly resolved | `TestRunnerTokenScopeInTokenAuthMode`, runner protocol middleware, runner token docs. | Deep job ownership and group/label policy still need broader tests. | Add negative tests for mismatched runner id/job id/group. |
| Runner cannot access admin APIs | resolved for tested path | `TestRunnerTokenScopeInTokenAuthMode` asserts runner bearer token cannot access `GET /api/v1/runners`. | Only one admin path tested. | Expand negative tests to credentials, audit, deployments, releases. |
| Runner cannot mutate unrelated jobs | partially resolved | `docs/security/ROUTE_PERMISSION_MATRIX.md` states ownership requirement; runner service routes validate token and job ownership in foundation paths. | Delta evidence did not prove a dedicated unrelated-job negative test. | Add explicit tests for log/status update against unclaimed job. |
| Secret values are not returned by normal APIs | mostly resolved | `TestCredentialRoutesDoNotReturnSecretValue`, `TestCredentialRoutesDoNotReturnCredentialValues`, secret refs model. | Needs broader coverage for all secret-provider and audit/event paths. | Add response redaction tests across all credential/secret endpoints. |
| Token hashes are not returned | mostly resolved | `TestServiceAccountAndAPITokenRoutesDoNotLeakHashes`, `TestRunnerRoutes`, `TestRunnerTokenScopeInTokenAuthMode`, `pipeline_runs_test.go` checks. | Future token routes must preserve one-time-only behavior. | Keep hash leakage tests close to token handlers. |
| Runner trust boundary docs | resolved | `docs/security/runner-trust-boundary.md`, `docs/operations/runner-security.md`, threat model updates. | Documentation is not enforcement. | Add operator hardening examples and sandbox policy design. |
| Unsafe executor / apply / sync / host deploy production flags | mostly resolved | `RuntimeConfig` flags and production validation tests reject local shell, privileged executor, remote host deploy, Kubernetes apply, Argo sync, and insecure registry. | Config flags are validation gates, not full runtime enforcement everywhere. | Wire flags into all production deployment paths before beta/RC. |
| Audit of security-sensitive actions | partially resolved | Existing audit/event foundations; security docs mention token actions and denied permissions. | Permission denied, unsafe override, and secret-use audit are not uniformly proven. | Add security-sensitive audit tests and durable audit persistence. |

## 4. Persistence Audit

| Entity | Storage Mode | Repository Evidence | Migration Evidence | Tests | Risk |
|---|---|---|---|---|---|
| PipelineRun | mixed | `internal/adapters/repository/postgres/pipeline_store.go`; memory fallback in usecase/runtime. | `000003_persistence_foundation`, `000004_runtime_recovery`. | `pipeline_store_test.go`, pipeline service tests. | Medium: Postgres path exists, cross-process e2e is limited. |
| StageRun | mixed | Embedded in PipelineRun record JSON and job rows. | Pipeline runtime migrations. | Pipeline store/service tests. | Medium: nested persistence, not independent table. |
| JobRun | mixed | `runtime_job_runs` operations in `pipeline_store.go`. | Runtime/persistence migrations. | Claim/status tests. | Medium: concurrent claim needs more DB isolation testing. |
| StepRun | mixed | Embedded in pipeline record JSON. | Pipeline runtime migrations. | Pipeline state/service tests. | Medium: not independently queryable. |
| DeploymentRun | mixed | `internal/adapters/repository/postgres/deployment_store.go`; memory fallback. | `runtime_deployment_runs` in `000007_deployment_release_runtime.up.sql`. | Interface/migration tests, deployment usecase tests. | Medium-high: real DB recovery tests limited. |
| DeploymentPlan | mixed | Stored in DeploymentRun raw record/plan fields through deployment store. | `runtime_deployment_runs` raw JSON plus related tables. | Deployment store interface/migration tests. | Medium: plan retrieval persistence needs deeper integration tests. |
| DeploymentResource | mixed | `runtime_deployment_resources`, resource save logic. | `runtime_deployment_resources` and indexes in migration 000007. | Migration test checks table/index. | Medium: live-state reconciliation not proven. |
| ManifestSnapshot | mixed | `saveSnapshot` in deployment store. | `runtime_manifest_snapshots`. | Migration test checks table. | Medium: content/object-store persistence remains foundation. |
| RollbackPlan | mixed | `saveRollbackPlan` in deployment store. | `runtime_rollback_plans`. | Migration test checks table. | High: rollback execution remains guarded/foundation. |
| Deployment logs | mixed | `AppendLog`, `Logs` in deployment store. | `runtime_deployment_logs`, sequence index. | Migration/interface tests. | Medium: retention/storage pressure not fully handled. |
| Deployment events | mixed | `AppendEvent`, `Events` in deployment store. | `runtime_deployment_events`. | Migration/interface tests. | Medium: outbox not uniform across subsystems. |
| Deployment audit | mixed | `AppendAudit`, `Audits` in deployment store. | `runtime_deployment_audit_logs`. | Migration/interface tests. | Medium-high: not tamper-evident. |
| Release | mixed | `internal/adapters/repository/postgres/release_store.go`; memory fallback. | `runtime_releases`. | Interface/migration tests. | Medium: integration coverage shallow. |
| ReleaseArtifact | mixed | `release_store.go` persists bindings. | `runtime_release_artifacts`, index. | Migration test checks table/index. | Medium: registry resolution remains optional. |
| ReleasePlan | mixed | `ReleaseOrchestrationStore.Save` and `GetPlan`. | `runtime_release_plans`. | Interface/migration tests. | Medium: restart planning flow not e2e-tested. |
| ReleaseExecution | mixed | `GetExecution`, `ListExecutions`, stale/non-terminal queries. | `runtime_release_executions`. | Interface/migration tests. | Medium-high: recovery loop incomplete. |
| ReleaseExecution targets | mixed | target upsert logic in release orchestration store. | `runtime_release_execution_targets`. | Migration test checks table. | Medium: target replay/idempotency needs tests. |
| ReleaseExecution events | mixed | `AppendEvent`, `Events`. | `runtime_release_execution_events`. | Migration test checks table. | Medium: outbox consistency incomplete. |
| ReleaseExecution audit | mixed | `AppendAudit`, `Audits`. | `runtime_release_execution_audit_logs`. | Migration test checks table. | Medium-high: not tamper-evident. |
| Runner | mixed | `pipeline_store.go` runner methods and token hash fields. | `runtime_runners` migrations. | runner route/store tests. | Medium-high: sandbox and fleet races remain. |
| EventOutbox | mixed | `PipelineStore.AppendOutbox`, `ListPendingOutbox`. | runtime outbox migrations. | pipeline store tests. | Medium: mostly pipeline-focused. |

## 5. Security / RBAC / Runner Audit

| Security Area | Status | Evidence | Test Coverage | Remaining Risk |
|---|---|---|---|---|
| Route permission matrix exists | mostly resolved | `docs/security/ROUTE_PERMISSION_MATRIX.md`. | Indirect via route tests. | Matrix can drift without generated checks. |
| Critical routes mapped to permissions | mostly resolved | `internal/api/http/routes/routes.go` wraps many mutations with `RequirePermission`. | Critical OIDC tests. | Some read/foundation routes remain authenticated-only rather than fine-grained. |
| Auth required where appropriate | mostly resolved | auth middleware and route wrappers. | unauthenticated/forbidden tests in route and middleware suites. | Dev mode intentionally bypasses auth locally. |
| Runner token authentication | mostly resolved | `X-Nivora-Runner-Token` accepted only for runner protocol paths; token validation in runner handlers/usecase. | `TestRunnerRoutes`, `TestRunnerTokenScopeInTokenAuthMode`. | More mismatched-runner negative tests needed. |
| Runner token scope | mostly resolved | runner protocol middleware creates runner-scoped subject; usecase validates token. | heartbeat and admin-denial tests. | Full group/environment/label policy is not production-proven. |
| Service account behavior | partially resolved | service account/API token usecase and scope checks. | cross-project denial and token no-hash tests. | Full service-account permission matrix incomplete. |
| Viewer vs developer vs maintainer vs admin behavior | partially resolved | OIDC route test provider checks viewer denial, developer allow, auditor restrictions. | selected route tests. | Maintainer/admin matrix is not exhaustive. |
| Credential/secret route protection | mostly resolved | credential/secret routes require `credential.manage`. | credential no-value response tests. | Secret-use audit and provider paths need more tests. |
| Audit route protection | mostly resolved | `/api/v1/audit/search` requires `audit.read`. | auditor read / mutate-denied test. | Evidence/retention paths need broader role tests. |
| Runner mutation route protection | mostly resolved | runner admin routes require `runner.manage`; runner protocol routes require runner token. | runner route tests. | Unrelated job mutation negative test missing. |
| Secret/token response safety | mostly resolved | tests for credential value absence, API token hashes, runner token hashes. | route tests and pipeline route tests. | Broader audit/event/log leakage tests still needed. |
| Redaction coverage | usable | `internal/infra/crypto/redact.go`, redaction tests, secret scan. | crypto tests and no-secrets script. | Must ensure every adapter/log path uses redaction. |
| Production unsafe config validation | mostly resolved | `RuntimeConfig`, config validation tests. | `TestProductionRejectsUnsafeSecurityDefaults`. | Helm default values still need a production profile review. |

## 6. API Contract Audit

| Contract Area | Status | Evidence | Gap | Recommendation |
|---|---|---|---|---|
| Registered routes vs OpenAPI paths | mostly resolved | `TestOpenAPIPathsMatchRegisteredRoutes`. | Path/method only, not request/response schemas. | Add schema conformance tests for critical APIs. |
| OpenAPI paths vs registered routes | mostly resolved | same route contract test checks both directions. | Does not validate middleware/security semantics. | Add security scheme checks for protected groups. |
| Not implemented route responses | resolved | `TestAllPlaceholderRoutesReturnStructuredNotImplemented`. | Placeholder list must stay current. | Keep tests mandatory. |
| Implemented route maturity labels | mostly resolved | OpenAPI metadata says beta-candidate / foundation-only / experimental. | Per-operation maturity is not exhaustive. | Add tags or extension fields for implemented/placeholder status later. |
| Security schemes in OpenAPI | mostly resolved | `BearerAuth` and `RunnerTokenAuth`; runner protocol endpoints marked. | Most protected routes rely on top-level bearer security rather than per-operation permission details. | Add route permission matrix link or OpenAPI extensions. |
| AsyncAPI event accuracy | partially resolved | AsyncAPI uses beta-candidate/not-production-ready wording. | Emitted-vs-documented events are not mechanically verified. | Add event contract tests once event bus stabilizes. |

## 7. Production Readiness Score Delta

| Dimension | Previous Score | Current Score | Change | Evidence |
|---|---:|---:|---:|---|
| runtime durability | 3 | 3 | 0 | Deployment/Release persistence improved, but restart recovery tests remain limited. |
| persistence | 2 | 3 | +1 | Postgres deployment/release stores and migration 000007 added. |
| security | 3 | 3 | 0 | Better docs/tests/config validation; still no full human/security review closure. |
| auth/RBAC | 2 | 3 | +1 | Route matrix and critical RBAC tests added. |
| secrets | 3 | 3 | 0 | Token/credential response tests improved; external provider lifecycle remains partial. |
| audit/compliance | 2 | 2 | 0 | Deployment/release audit persistence improved; tamper-evident audit still missing. |
| runner security | 2 | 3 | +1 | Runner token scope/admin denial docs/tests added; no sandbox. |
| Kubernetes CD safety | 3 | 3 | 0 | Config flags improved; production cluster behavior unchanged. |
| GitOps safety | 2 | 2 | 0 | Argo sync remains guarded foundation. |
| artifact immutability | 3 | 3 | 0 | No major change in this delta. |
| policy gates | 2 | 2 | 0 | No major change in this delta. |
| observability | 3 | 3 | 0 | No major change in this delta. |
| HA/DR | 2 | 2 | 0 | No restore drill added. |
| operations docs | 4 | 4 | 0 | Runner security docs improved, but procedures still not fully validated. |
| packaging/install | 3 | 3 | 0 | Helm template/lint passes; production values still need review. |
| testing | 4 | 4 | 0 | New focused tests added; external DB/runner e2e still missing. |
| performance | 2 | 2 | 0 | No major change in this delta. |
| multi-tenancy | 1 | 1 | 0 | No major change in this delta. |
| upgrade/migrations | 2 | 3 | +1 | New reversible migration and index checks for deployment/release runtime. |
| supportability | 3 | 3 | 0 | Runner docs improved; production drills still pending. |

## 8. Remaining Top Risks

1. Risk: Runner workload isolation is not a sandbox. Why it matters: shell jobs can execute local commands with runner privileges. Evidence: `docs/security/runner-trust-boundary.md`. Mitigation: define enforced runner isolation profiles. Recommended next goal: runner isolation and execution policy hardening.
2. Risk: DB-backed restart recovery is not proven end-to-end. Why it matters: durable stores are useful only if server/worker/runner can resume safely. Evidence: Postgres stores exist, but tests are mostly unit/migration-level. Mitigation: add disposable Postgres and multi-process recovery tests. Recommended next goal: durable runtime recovery test harness.
3. Risk: RBAC remains non-exhaustive. Why it matters: broad API surface can hide underprotected endpoints. Evidence: selected route tests exist, matrix is manual. Mitigation: generate table-driven route permission tests from the matrix. Recommended next goal: exhaustive RBAC/tenant enforcement.
4. Risk: Audit/evidence is not tamper-evident or uniformly durable. Why it matters: compliance claims depend on reliable audit. Evidence: audit stores are mixed and compliance service aggregates foundations. Mitigation: durable append-only audit/evidence store. Recommended next goal: compliance persistence hardening.
5. Risk: Production configuration may still be confusing through packaging defaults. Why it matters: Helm renders memory runtime store by default. Evidence: `make verify` Helm output. Mitigation: add production values profile and validation. Recommended next goal: production install safety hardening.
6. Risk: API contract validation is path-level only. Why it matters: schemas/security/maturity can still drift. Evidence: `openapi_contract_test.go` checks route/method and placeholder text only. Mitigation: add schema and security contract tests. Recommended next goal: API contract hardening.
7. Risk: Governance/security stores remain memory-backed. Why it matters: approvals, scans, credentials metadata, and audit can be lost in production-like restarts. Evidence: runtime wiring still constructs memory stores for security, credential, approval, cloud, tenancy, compliance inputs. Mitigation: prioritize critical durable stores. Recommended next goal: governance persistence hardening.
8. Risk: External integrations are still foundation/noop/skeleton. Why it matters: users may overestimate cloud, Argo CD, registry, scanner, host support. Evidence: capability status labels many integrations experimental/foundation/placeholder. Mitigation: keep docs honest and avoid feature expansion claims. Recommended next goal: integration readiness gates.
9. Risk: Migration tests do not execute against a real database in baseline verification. Why it matters: reversible SQL syntax/behavior can break outside static checks. Evidence: migration validation checks files/strings. Mitigation: add optional or CI disposable Postgres migration test. Recommended next goal: database migration integration tests.
10. Risk: Token/secret leakage tests are not universal. Why it matters: future endpoints can leak sensitive values. Evidence: selected route tests cover token/hash/credential responses. Mitigation: add generic response scan tests for sensitive keys. Recommended next goal: secret response safety expansion.

## 9. Recommended Next Three Goals

### Goal 1: Durable Runtime Recovery Integration Tests

- Why now: Postgres stores now exist for deployment/release runtime, but restart recovery is not proven.
- Scope: disposable Postgres migration up/down, DB-backed server/worker/runner recovery tests, stale DeploymentRun/ReleaseExecution reconciliation, idempotent create replay tests.
- Non-goals: no new deployment engines, no cloud integrations, no production Argo automation.
- Acceptance criteria: process restart does not lose PipelineRun, DeploymentRun, ReleasePlan, ReleaseExecution, logs, events, audit, or runner claim state in configured Postgres mode.

### Goal 2: Exhaustive RBAC, Tenant, and Runner Boundary Tests

- Why now: route matrix exists, but it is not yet mechanically complete.
- Scope: table-driven permission tests for critical routes, service account scope tests, runner token negative tests for unrelated jobs/admin routes, tenant isolation tests for project/environment resources.
- Non-goals: no full SSO UI, no new identity provider implementation.
- Acceptance criteria: every critical route has allow/deny tests for anonymous, viewer, developer, maintainer/admin, auditor, service account, and runner token where relevant.

### Goal 3: Production Install and Audit Durability Hardening

- Why now: config validation improved, but Helm/package defaults and audit durability are still beta blockers.
- Scope: production Helm values, config validation docs/tests, durable audit/evidence repository plan or implementation for critical runtime events, restore/migration drills.
- Non-goals: no operator, no new cloud providers, no external notification/scanner integrations.
- Acceptance criteria: production-like config rejects unsafe defaults, Helm production profile avoids memory runtime store, and critical audit/evidence survives restart in Postgres mode.

## 10. Go / No-Go Decision

- Is Nivora ready for broader public beta? **Conditional yes.** It is credible as a beta-candidate if public materials clearly state foundation/experimental limits and not production-ready status.
- Is Nivora ready for production-candidate labeling? **No.** Runner sandboxing, full recovery, durable audit/evidence, exhaustive RBAC/tenant enforcement, and production install validation remain blockers.
- Is Nivora ready for new feature expansion? **Mostly no.** Small feature work may be acceptable only if it supports hardening, tests, or docs. Broad new integrations should wait.
- Should development continue with hardening, or move to integration work? **Continue hardening.** The three completed goals improved the baseline, but the next highest-value work is recovery, RBAC/tenant enforcement, and production install/audit durability.

## 11. Machine-Readable Summary

```json
{
  "overall_maturity": "beta-candidate",
  "production_readiness_score": 2.8,
  "resolved_blockers": [
    "maturity language mostly aligned to beta-candidate foundation",
    "OpenAPI and AsyncAPI status drift resolved",
    "OpenAPI route coverage check added",
    "placeholder route clarity tested",
    "DeploymentRun and release runtime PostgreSQL persistence foundation added",
    "route permission matrix added",
    "critical RBAC tests added",
    "runner token scope and admin denial tests added",
    "secret/token hash response safety tests added",
    "production unsafe config validation added",
    "runner trust boundary docs added"
  ],
  "remaining_blockers": [
    "restart recovery not proven end-to-end for deployment and release runtime",
    "RBAC and tenant isolation are not exhaustive",
    "runner execution is not sandboxed",
    "governance/security/compliance stores remain partly memory-backed",
    "audit/evidence is not tamper-evident",
    "production Helm/install profile needs hardening",
    "schema/security OpenAPI semantics are not fully contract-tested"
  ],
  "regressions": [],
  "next_recommended_goals": [
    "Durable Runtime Recovery Integration Tests",
    "Exhaustive RBAC, Tenant, and Runner Boundary Tests",
    "Production Install and Audit Durability Hardening"
  ],
  "go_no_go": {
    "public_beta": "conditional yes",
    "production_candidate": "no",
    "feature_expansion": "mostly no"
  },
  "verification": {
    "passed": [
      "git status --short",
      "git log --oneline -n 20",
      "go mod tidy",
      "make fmt-check",
      "go test ./...",
      "go vet ./...",
      "go build ./cmd/nivora-server",
      "go build ./cmd/nivora-worker",
      "go build ./cmd/nivora-runner",
      "go build ./cmd/nivora",
      "./scripts/verify-architecture.sh",
      "./scripts/verify-no-secrets.sh",
      "make verify",
      "git diff --check"
    ],
    "failed": [],
    "skipped": []
  }
}
```
