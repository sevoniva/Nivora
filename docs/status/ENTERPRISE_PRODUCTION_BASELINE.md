# Nivora Enterprise Production Baseline

Current maturity: **hardened beta-candidate foundation, not production-ready**.

This baseline records the current enterprise-readiness evidence after the recent runtime, security, production-profile, and MCP hardening work. It is not a GA declaration. It is a working baseline for closing production-candidate gaps.

## Evidence Reviewed

| Area | Evidence |
|---|---|
| Public status | `README.md`, `docs/status/CAPABILITY_STATUS.md`, `docs/status/IMPLEMENTATION_AUDIT.md`, `docs/status/POST_HARDENING_DELTA_AUDIT.md` |
| Runtime persistence | `internal/adapters/repository/postgres/*`, `internal/adapters/repository/postgres/runtime_recovery_integration_test.go`, `docs/status/STORE_PERSISTENCE_MATRIX.md` |
| Security and RBAC | `docs/security/ROUTE_PERMISSION_MATRIX.md`, `internal/api/http/routes/rbac_matrix_test.go`, `internal/api/http/routes/tenant_isolation_test.go`, `docs/security/runner-trust-boundary.md` |
| MCP control plane | `cmd/nivora-mcp`, `internal/api/mcp`, `docs/security/MCP_PERMISSION_MATRIX.md`, `examples/mcp/scenarios`, `examples/mcp/golden-answers` |
| Packaging | `deployments/helm/values-production.yaml`, `deployments/docker-compose/docker-compose.production.example.yaml`, `scripts/smoke-production-install.sh`, `scripts/verify-helm-safety.sh` |
| API contracts | `api/openapi/openapi.yaml`, `api/asyncapi/asyncapi.yaml`, `internal/api/http/routes/openapi_contract_test.go`, `scripts/verify-api-specs.sh` |
| CI | `.github/workflows/ci.yaml`, `Makefile` |

## Current Maturity

Nivora has enough backend structure, runtime persistence, guardrails, and verification to be treated as a **hardened beta-candidate foundation**. It should not be marketed or installed as production-ready. The current strongest areas are backend modularity, PostgreSQL store coverage, local runtime smoke tests, RBAC route testing, safe production profile checks, and local stdio MCP evidence.

## Production Blockers

| Blocker | Current Evidence | Required Before Production-Candidate |
|---|---|---|
| Runner sandboxing is operator-dependent | Shell executor safety docs and tests exist, but shell execution is not an OS-level sandbox | Container/VM isolation story, default production runner profile, and operational proof |
| Remote MCP is experimental and not broadly safe to expose | Local stdio MCP is tested; an opt-in read-only JSON-RPC foundation exists; distributed rate limits, pagination, and tenant-filter proof are still missing | Harden remote read-only auth contract, scope filters, response limits, audit proof |
| Live install and restore drills are incomplete | Helm/Compose profiles and smoke scripts exist; full live restore is not automated at production scale | Repeatable install, backup, restore, and migration drill in CI or documented release gate |
| External integrations remain foundation-level | Kubernetes, Argo CD, OCI, cloud, scanner, notification, and secret providers are guarded or skeleton/fake | Explicit beta/production integration hardening per adapter |
| Complete multi-tenant enforcement is not proven everywhere | Route tests and tenant isolation tests exist; MCP and some read models still need deeper scope proof | Tenant fixture coverage for all critical API and MCP resource paths |

## Enterprise Delivery Blockers

| Blocker | Impact | Next Evidence Needed |
|---|---|---|
| Operational SLOs are documented but not enforced | Operators cannot yet rely on measured service objectives | Metrics coverage tests, alert examples, runbook smoke checks |
| Audit/evidence is partial | Compliance workflows need durable, scoped, searchable evidence | Hash-chain verification in real Postgres CI for all critical audit-producing paths |
| Upgrade/migration safety is not a release gate | Enterprise upgrades need repeatable forward/backward confidence | Migration compatibility tests and release checklist evidence |
| Performance limits are not characterized | Capacity planning is weak | Benchmarks, log ingestion load test, runner heartbeat load test |
| Supportability still depends on maintainers reading internals | Enterprise support needs predictable diagnostics | Diagnostics endpoint coverage and troubleshooting decision trees |

## MCP Opening Blockers

| Surface | Current Decision | Blockers |
|---|---|---|
| Local stdio MCP | Go for maintainer-local read-only and plan-only use | Keep scenario/golden-answer validation in CI |
| Remote read-only MCP | Experimental opt-in foundation only | Bearer/service-account identity, response caps, and in-process per-subject rate limits exist; deeper OAuth/OIDC lifecycle, tenant filters, distributed rate limits, pagination, and remote audit metadata remain blockers |
| Remote plan-only MCP | Experimental opt-in foundation only | Same as read-only plus stronger abuse controls for plan summaries |
| Action MCP | No-go | Destructive delivery, approval, token, secret, runner, Git, host, and Kubernetes actions remain intentionally denied |

## Local Verification Baseline

Required local checks for this baseline:

| Check | Required Result |
|---|---|
| `make verify` | pass |
| `make verify-mcp` | pass |
| `make verify-ai-control-plane` | pass |
| `make verify-enterprise-readiness` | pass |
| `./scripts/verify-no-secrets.sh` | pass |
| `git diff --check` | pass |

## Current Non-Commitments

Nivora does not currently promise:

- production-safe untrusted shell execution;
- broad remote MCP exposure;
- remote MCP actions;
- full enterprise SSO lifecycle;
- full cloud provider inventory parity;
- full Kubernetes production CD;
- full Argo CD automation;
- full Harbor/Nexus/JFrog product integration;
- full backup/restore automation;
- production-scale HA validation.
