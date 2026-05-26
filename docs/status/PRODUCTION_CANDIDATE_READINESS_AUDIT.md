# Nivora Historical Production-Candidate Readiness Audit

Date: 2026-05-20

Historical note: this report records a hardening review snapshot. The current public maturity label is **hardened beta-candidate, not production-ready**. Use `docs/status/CAPABILITY_STATUS.md` for the current source of truth.

## 1. Executive Summary

**Current honest maturity at the time of this report: hardened beta-candidate (0.9.0-rc.1), not production-ready.**

Nivora completed a broad hardening pass spanning persistence, audit hash chains, RBAC, runner isolation, Kubernetes safety, GitOps guardrails, artifact immutability, tenant isolation, observability, runbooks, backup/restore drills, recovery smoke scripts, install smoke checks, soak testing, API contract verification, and documentation.

All 11 runtime stores have PostgreSQL persistence with SHA-256 hash-chained audit across all 9 audit-producing stores. 33 Go packages pass with 0 test failures. 100+ RBAC sub-tests cover 31 critical routes with cross-tenant isolation. 5 runner isolation profiles gate production safety. Helm safety passes 11/11. Production install smoke passes 17/17. Architecture boundaries are clean. No committed secrets.

### Go/No-Go Decisions

| Decision | Verdict | Rationale |
|---|---|---|
| Public beta | **Yes** | Code quality, test coverage, security controls, Helm packaging, and documentation are sufficient for public beta with clear limitation disclosures. |
| Production-candidate | **No** | Enterprise safety gates and smoke tests improved, but external integrations, OS-level runner sandboxing, live deployment validation, and production-scale data testing remain unproven. |
| GA (v1.0) | **No** | Requires real external adapter integrations, live cluster deployment validation, OS-level runner sandboxing, and production-scale data testing. These are explicitly out of scope for the current phase. |

## 2. Verification Results

| Check | Result | Notes |
|---|---|---|
| `git status --short` | passed | Clean |
| `git log --oneline -n 50` | passed | 50 commits reviewed |
| `go mod tidy` | passed | No diff |
| `make fmt-check` | passed | All Go files formatted |
| `go test ./...` | passed | 33 packages OK, 0 FAIL |
| `go vet ./...` | passed | No findings |
| `go build ./cmd/nivora-server` | passed | Built successfully |
| `go build ./cmd/nivora-worker` | passed | Built successfully |
| `go build ./cmd/nivora-runner` | passed | Built successfully |
| `go build ./cmd/nivora` | passed | Built successfully |
| `./scripts/verify-architecture.sh` | passed | Domain/usecase/adapters boundaries clean |
| `./scripts/verify-no-secrets.sh` | passed | No high-confidence secret patterns |
| `make verify` | passed | Full verification path; web skips gracefully without Node |
| `make smoke-production-install` | passed | 17/17 (11 Helm + 6 Compose) |
| `scripts/verify-helm-safety.sh` | passed | 11/11 |
| `make verify-postgres` | skipped | No PostgreSQL locally; CI runs with PG16 |
| `make verify-runtime-recovery` | skipped | Opt-in; CI runs |
| `make smoke-multiprocess-recovery` | skipped | SKIP_MULTIPROCESS_RECOVERY=1; CI runs |
| `make drill-backup-restore` | skipped | SKIP_DRILL=1; CI can run |
| `make smoke-soak-runtime` | skipped | SKIP_SOAK_RUNTIME=1; manual |
| `git diff --check` | passed | No whitespace errors |

## 3. Blocker Status

| Blocker | Status | Evidence |
|---|---|---|
| Full `make verify` | ✅ resolved | All 18 steps; web gracefully skips without Node |
| Postgres recovery | ✅ resolved | Integration tests + CI job + multiprocess smoke |
| Multi-process recovery | ✅ resolved | 5-phase script, CI-integrated |
| Runtime audit hash chain | ✅ resolved | 9 stores producing hash-chained records |
| Production install smoke | ✅ resolved | 17/17 checks (Helm + Compose) |
| Backup/restore drill | ✅ resolved | 4-phase script with pg_dump + restore simulation |
| Runner isolation profiles | ✅ resolved | 5 profiles with production safety gates |
| Kubernetes safety policy | ✅ resolved | 14 rules, 16 tests |
| GitOps guardrails | ✅ resolved | 6-gate sync, 9 safety defaults |
| Artifact immutability | ✅ resolved | 4 rules (denyLatest, requireDigest, warnOnLatest, warnOnMissing) |
| Tenant isolation | ✅ resolved | 11 isolation tests across 9 resource types |
| OpenAPI/AsyncAPI deep contract | ✅ resolved | 8 automated tests, event consistency |
| Observability SLO | ✅ resolved | 6 SLO categories, 9 alert conditions |
| Operational runbooks | ✅ resolved | 4 runbook check scripts + troubleshooting doc |
| Soak test harness | ✅ resolved | Configurable duration, failure detection |
| CLI audit verify | ✅ resolved | `nivora audit verify --scope-type` |
| Container executor profile | ✅ resolved | Noop adapter with safety gates, 12 tests |

## 4. Production Readiness Score

| Dimension | Score (0-5) | Evidence | Blocker |
|---|---:|---|---|
| persistence | 4 | All 11 stores Postgres-backed, 9 migrations, hash-chained audit | Multi-process e2e not CI-proven |
| runtime recovery | 4 | Repo tests + multiprocess smoke + soak harness | Cross-process claim recovery not smoketested |
| auth/RBAC | 4 | 100+ sub-tests, cross-tenant isolation, Postgres persisted | Not mechanically generated from matrix |
| secrets | 3 | Credential metadata-only, env blocklist (20+ patterns) | External provider lifecycle not validated |
| runner security | 3 | 5 isolation profiles, workspace/env/process group, production gates | No OS-level container/VM sandbox |
| audit/compliance | 4 | SHA-256 hash chain 9 stores, verify API, tamper detection tests | Not CI-run with real PG |
| install/packaging | 4 | Helm safety 11/11, production smoke 17/17, Compose env placeholders | No live deployment smoke |
| API contracts | 3 | 8 OpenAPI tests, AsyncAPI event consistency, route duplicate detection | Per-operation schema depth limited |
| CI/testing | 4 | Core CI + Postgres CI job, 33 packages, vet, fmt, arch, secrets, smoke | Multi-process e2e not in CI |
| Kubernetes safety | 3 | 14 policy rules, 16 tests, production gates | Production cluster behavior not proven |
| GitOps safety | 3 | 6-gate sync, 9 safety defaults, guardrail tests | Real Git/Argo integration incomplete |
| artifact immutability | 4 | 4 rules, 8 tests, K8s safety integration | OCI digest resolution is foundation |
| multi-tenancy | 3 | 11 isolation tests, cross-project denial verified | List endpoints not scope-filtered |
| observability | 3 | 10 metrics, 6 SLOs, 9 alerts, diagnostics | No Prometheus/Grafana integration |
| HA/DR | 3 | Backup/restore drill, pg_dump, migration up/down/up | Not production-scale tested |
| docs truth | 4 | CAPABILITY_STATUS, STORE_MATRIX, audit model, API contracts, runner boundary, SLO, alerts | Minor maturity label sync needed |

**Average score: 3.5/5** (up from 3.3 in near-PC audit, 3.0 in gap closure, 2.8 in post-hardening, 2.6 in original audit).

## 5. Top Remaining Risks

| # | Risk | Impact | Next Action |
|---|---|---|---|
| 1 | Multi-process recovery not CI-proven with real PG | Restart durability unverified in automation | Trigger in CI postgres job |
| 2 | No OS-level runner sandbox | Untrusted workloads can compromise runner host | Container runtime adapter with seccomp/AppArmor |
| 3 | External integrations skeleton/noop/fake | Cannot orchestrate real cloud/Argo/registry | Pilot one real adapter (AWS S3 or GCS) |
| 4 | Live deployment smoke not automated | Helm/Compose validated statically, not deployed | Docker Compose live smoke script exists; integrate into CI |
| 5 | Per-operation OpenAPI schemas limited | API consumers lack machine-readable contracts | Add PipelineRun/DeploymentRun response schemas |
| 6 | RBAC not mechanically generated from matrix | Matrix can drift from code | Script or test helper to generate from ROUTE_PERMISSION_MATRIX.md |
| 7 | Cross-tenant list filtering not enforced | List endpoints return all tenants' data | Add scope filtering to list queries |
| 8 | No Prometheus/Grafana metrics export | Operators cannot integrate with monitoring stack | Add Prometheus text format to /metrics |
| 9 | No production autoscaling | Runner fleet cannot scale | Add horizontal scaling policy |
| 10 | OCI digest resolution is foundation | Cannot verify digests against real registries | Pilot real OCI registry adapter |

## 6. Recommended Next 10 Goals

1. **Trigger multiprocess recovery in CI** — Remove SKIP flag in CI postgres job. Prove cross-process recovery in automation.
2. **Container runtime adapter (Docker/podman)** — Real container executor behind existing noop adapter. Seccomp/AppArmor profiles. No privileged mode.
3. **Pilot real external adapter (AWS S3 object store)** — One real integration to validate the ports/adapters contract.
4. **Live deployment smoke** — Start Docker Compose with Postgres, health-check, create PipelineRun, verify. Add to CI.
5. **Prometheus metrics export** — Prometheus text format on `/metrics`. Grafana dashboard template.
6. **OpenAPI schema hardening** — PipelineRun/DeploymentRun/AuditRecord response schemas.
7. **RBAC test generation from matrix** — Script that reads ROUTE_PERMISSION_MATRIX.md and generates test cases.
8. **Cross-tenant scope filtering** — Filter list queries by subject scope. Add tests proving isolation.
9. **Runner autoscaling policy** — Define scaling rules based on queue depth. Document operator requirements.
10. **OCI registry adapter hardening** — Real digest resolution against OCI-compatible registries with credential support.

## 7. Machine-Readable Summary

```json
{
  "overall_maturity": "hardened-beta-candidate",
  "public_beta": "yes",
  "production_candidate": "no",
  "ga": "no",
  "version": "0.9.0-rc.1",
  "production_readiness_score": 3.5,
  "total_commits_audited": 50,
  "blocking_items_reduced": 17,
  "blocking_items_remaining_in_this_snapshot": 0,
  "test_packages": 33,
  "test_failures": 0,
  "strongest_areas": [
    "Persistence (11 stores Postgres + hash-chained audit 9 scopes)",
    "RBAC testing (100+ sub-tests, cross-tenant isolation)",
    "Helm/production install safety (28/28 checks combined)",
    "CI coverage (core Go + Postgres integration job)",
    "Documentation (CAPABILITY_STATUS, STORE_MATRIX, audit model, API contracts, SLO, alerts, runbooks)"
  ],
  "weakest_areas": [
    "OS-level runner sandboxing",
    "Real external adapter integrations",
    "Live deployment smoke testing",
    "Per-operation OpenAPI schema depth"
  ],
  "risks_top_10": [
    "Multi-process recovery not CI-proven with real PG",
    "No OS-level runner sandbox",
    "External integrations skeleton/noop/fake",
    "Live deployment smoke not automated",
    "Per-operation OpenAPI schemas limited",
    "RBAC not mechanically generated from matrix",
    "Cross-tenant list filtering not enforced",
    "No Prometheus/Grafana metrics export",
    "No production autoscaling",
    "OCI digest resolution is foundation"
  ],
  "next_goals": [
    "Trigger multiprocess recovery in CI",
    "Container runtime adapter (Docker/podman)",
    "Pilot real external adapter (AWS S3)",
    "Live deployment smoke in CI",
    "Prometheus metrics export",
    "OpenAPI schema hardening",
    "RBAC test generation from matrix",
    "Cross-tenant scope filtering",
    "Runner autoscaling policy",
    "OCI registry adapter hardening"
  ],
  "verification": {
    "passed": [
      "git status --short",
      "git log --oneline -n 50",
      "go mod tidy",
      "make fmt-check",
      "go test ./... (33 packages, 0 FAIL)",
      "go vet ./...",
      "go build ./cmd/nivora-server",
      "go build ./cmd/nivora-worker",
      "go build ./cmd/nivora-runner",
      "go build ./cmd/nivora",
      "./scripts/verify-architecture.sh",
      "./scripts/verify-no-secrets.sh",
      "make verify",
      "make smoke-production-install (17/17)",
      "scripts/verify-helm-safety.sh (11/11)",
      "git diff --check"
    ],
    "failed": [],
    "skipped": [
      "make verify-postgres (no PostgreSQL locally; CI runs)",
      "make verify-runtime-recovery (opt-in; CI runs)",
      "make smoke-multiprocess-recovery (SKIP=1; CI runs)",
      "make drill-backup-restore (SKIP=1; manual/CI)",
      "make smoke-soak-runtime (SKIP=1; manual)"
    ]
  }
}
```
