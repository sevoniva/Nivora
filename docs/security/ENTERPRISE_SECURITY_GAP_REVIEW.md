# Enterprise Security Gap Review

Current maturity: **security beta foundation, not production-ready**.

This review tracks enterprise security gaps that remain after route/RBAC, runner-token, production-config, and MCP hardening.

## Current Strengths

| Area | Evidence |
|---|---|
| Route permissions | `docs/security/ROUTE_PERMISSION_MATRIX.md`, `internal/api/http/routes/rbac_matrix_test.go` |
| Runner token boundary | runner protocol tests and negative route tests |
| Secret response safety | credential/secret route tests, redaction helpers, no-secrets script |
| Production unsafe defaults | config validation tests, production Helm/Compose smoke scripts |
| MCP safety | MCP permission matrix, scenario tests, denied action tools |
| Threat models | `docs/security/threat-model.md`, `docs/security/mcp-threat-model.md` |

## Remaining Enterprise Gaps

| Gap | Risk | Evidence | Required Closure |
|---|---|---|---|
| Complete tenant isolation across every read model is not fully proven | Cross-project metadata exposure | route tenant tests exist, MCP tenant scope review says remote is incomplete | Add ownership tests for all MCP resources and sensitive visualization/read models |
| Runner shell execution is not an OS sandbox | Malicious job can affect host if runner is poorly isolated | runner trust-boundary docs | Provide production runner isolation profile and operator checklist |
| Remote MCP controls are incomplete | Remote AI clients could enumerate or over-read state if broadly exposed | opt-in remote read-only JSON-RPC exists; local stdio cap/timeout tests exist; broad remote exposure remains blocked | Add tenant scope, per-client rate limit, pagination, remote timeout/cap proof, and stronger audit proof before broad remote exposure |
| Audit evidence is not uniformly proven through a live production profile | Weak compliance evidence | hash-chain tests exist, full live audit chain still limited | Add Postgres audit-chain integration coverage for critical flows |
| External secret providers are foundation-level | Secret lifecycle varies by operator | provider skeletons/docs | Add provider validation contract tests before production claims |
| OIDC/SSO lifecycle is incomplete | Enterprise identity lifecycle gaps | auth foundation docs | Add provider-specific integration tests and token lifecycle docs |
| Destructive delivery actions remain dangerous if config is misused | Unsafe apply/sync/rollback/host deploy | production unsafe config tests | Keep disabled by default and audit explicit overrides |

## Required Security Gates Before Production-Candidate

1. Route/RBAC matrix must pass as a hard CI gate.
2. Runner token must remain denied outside runner protocol endpoints.
3. Secret/token response scanner must cover API, audit, events, logs, and MCP.
4. Production config must reject unsafe defaults.
5. Tenant isolation must be proven for critical read and mutation paths.
6. MCP remote exposure must remain opt-in and read-only until deeper remote controls are implemented.
7. Runner isolation must be documented as an operator requirement and backed by safe defaults.
8. Audit/evidence persistence must be verified against PostgreSQL in integration tests.

## Current Go / No-Go

| Area | Decision | Reason |
|---|---|---|
| Local development auth | go | useful for local development only |
| Token auth foundation | go for beta | tests and docs exist |
| Enterprise SSO | no-go for production | OIDC lifecycle not complete |
| Runner protocol | conditional go | token boundary tested; sandboxing remains operator responsibility |
| Local stdio MCP | go for maintainer-local use | read-only/plan-only and denial tests exist |
| Remote MCP | experimental foundation only | opt-in read-only JSON-RPC exists, but broad exposure still lacks deeper auth/scope/pagination/per-client-limit/audit controls |
| Production install | conditional beta | safe profiles exist; live restore and HA drills incomplete |
