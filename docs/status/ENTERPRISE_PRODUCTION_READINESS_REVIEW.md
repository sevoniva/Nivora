# Enterprise Production Readiness Review

Current maturity label: **hardened beta-candidate foundation, not production-ready**.

## Executive Summary

Nivora has moved beyond a simple MVP foundation. It has a modular backend, multiple binaries, PostgreSQL-backed runtime stores, migration tests, CI checks, guarded deployment surfaces, RBAC route tests, runner token boundaries, production-profile smoke checks, and a local stdio MCP control plane with validated scenarios.

It is still not production-ready. The largest remaining gaps are runner sandboxing, complete tenant isolation proof, remote MCP controls, full install/backup/restore drills, external adapter hardening, and production-scale operational validation.

## What Is Production-Candidate

| Area | Evidence | Caveat |
|---|---|---|
| Architecture boundaries | `./scripts/verify-architecture.sh`, AGENTS rules | Must stay enforced for new adapters |
| PostgreSQL store foundation | 11 Postgres stores, migrations, integration tests | Full live service recovery still expanding |
| Route/OpenAPI coverage | `openapi_contract_test.go` | Semantic API stability still needs release freeze |
| Production config safety | config tests and Helm/Compose smoke | Operators can still override unsafe flags |
| Local MCP denial model | blocked action tools and tests | Remote MCP is not approved |

## What Is Beta Only

| Area | Reason |
|---|---|
| Pipeline runtime | Useful shell runtime, but not a heavy workflow engine |
| Deployment runtime | Guarded dry-run/apply foundations, not full production CD |
| Release orchestration | Sequential and persisted, but governance/recovery depth remains limited |
| Runner protocol | Token/claim/log/status foundations, but sandboxing and scale need more proof |
| Auth/RBAC | Route coverage exists, but enterprise SSO lifecycle and full tenant isolation are incomplete |
| Packaging | Safe profiles exist, but live install/restore at scale is not proven |
| Observability | Metrics/diagnostics/runbooks exist, but SLO enforcement is incomplete |

## What Is Foundation Only

| Area | Reason |
|---|---|
| DevSecOps scanners | Noop/fake scanner and simple rules; no production scanner integration |
| Secret providers | Builtin/skeleton providers; full lifecycle not validated |
| Cloud inventory | Fake/skeleton inventory only |
| GitOps/Argo CD | Plan/status/guarded sync foundation only |
| Host deployment | Plan/dry-run/noop plus guarded SSH skeleton |
| Plugin system | Capability registry and protocol skeleton only |
| Web console | Minimal experimental UI |

## What Is Unsafe For Production Without Extra Controls

| Area | Risk |
|---|---|
| Shell executor on shared hosts | Not an OS-level sandbox |
| Remote MCP | Missing auth, tenant filters, rate limits, response caps, pagination, and remote audit proof |
| Kubernetes apply/sync/rollback execution | Guarded but not GA production CD |
| Remote host deploy | SSH risk and no default remote execution |
| Insecure registries | Must be explicit and environment-scoped |

## Top P0 Blockers

1. Complete remote MCP controls are absent.
2. Runner sandboxing remains operator-dependent.
3. Full tenant isolation proof is incomplete for every read model and MCP resource.
4. Live backup/restore and install drills are not a mandatory release gate.
5. External integration adapters remain foundation/experimental.

## Top P1 Blockers

1. Audit hash-chain proof is not uniform across every runtime audit path.
2. Performance and load limits are not fully characterized.
3. SLO metrics and alerts are not complete.
4. Upgrade/migration compatibility is not a full release gate.
5. Golden MCP answers need continued drift checks as tools grow.

## Platform Readiness

The platform core is beta-candidate for backend development and controlled demos. It has real binaries, tests, packaging assets, config validation, and API specs. It is not ready for unsupervised production adoption.

## Security Readiness

Security is stronger than an MVP: RBAC, route matrix tests, runner token boundaries, redaction tests, production unsafe config checks, and threat models exist. Enterprise readiness still requires full tenant proof, SSO lifecycle hardening, provider secret lifecycle tests, and stronger audit evidence.

## Runtime Readiness

Runtime persistence is materially improved. Pipeline, deployment, release, runner claim, and outbox recovery have optional PostgreSQL integration tests. Full production-candidate runtime still needs multiprocess worker/runner recovery drills as a standard release gate.

## Persistence Readiness

PostgreSQL persistence exists for the critical runtime stores and governance stores. Migration up/down tests exist. Production-candidate status requires upgrade compatibility checks, live restore drills, and tighter evidence around audit-chain coverage.

## Runner Readiness

Runner token and claim foundations are useful for beta. Shell execution is not a sandbox. Production use requires isolated runners, least-privilege host configuration, secret minimization, and stronger fleet-scale tests.

## MCP Readiness

Local stdio MCP is a useful maintainer assistant for read-only and plan-only workflows. Remote MCP is no-go. Action MCP is no-go.

## Install Readiness

Helm and Docker Compose production-like profiles exist and are statically checked. They should be treated as beta install assets until live install, migration, backup, and restore drills are repeatable.

## Observability Readiness

Request/correlation IDs, metrics, diagnostics, and runbooks exist. Production-candidate operation needs alert examples, dashboard guidance, and failure-mode tests.

## Backup / Restore Readiness

Backup and restore procedures are documented with optional scripts. This is not enough for production-candidate until the restore drill is automated and run against the same persistence profile used for runtime recovery.

## API Contract Readiness

Route/OpenAPI coverage tests exist and placeholder routes are checked. Semantic compatibility and versioning need a release freeze process before GA.

## Test Readiness

The test suite is broad for a beta-candidate foundation. The biggest missing class is long-running, production-like, multi-process, database-backed validation under load.

## Go / No-Go Decision

| Decision Area | Result |
|---|---|
| Public beta with honest limitations | conditional go |
| Production-candidate label | no-go |
| GA production-ready label | no-go |
| Local stdio MCP for maintainers | go |
| Remote read-only MCP | no-go |
| Remote action MCP | no-go |

## Required Work Before Public Beta

- Keep maturity language accurate.
- Keep `make verify`, `make verify-mcp`, and enterprise readiness checks green.
- Document demo paths that do not require external systems.
- Publish known limitations clearly.

## Required Work Before Production-Candidate

- Prove runtime recovery through multiprocess worker/runner scenarios.
- Close tenant isolation gaps.
- Add remote MCP controls or keep remote MCP disabled.
- Add live install, backup, restore, and migration drills.
- Harden runner isolation defaults.
- Add SLO/alert/runbook validation.

## Required Work Before GA

- Freeze API compatibility.
- Prove upgrade and rollback behavior.
- Complete enterprise auth/secret lifecycle validation.
- Prove HA/DR procedures.
- Characterize performance and capacity limits.
- Close or explicitly scope every P0/P1 enterprise risk.
