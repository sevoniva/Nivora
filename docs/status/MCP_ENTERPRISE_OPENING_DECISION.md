# MCP Enterprise Opening Decision

Current decision: **local stdio MCP is allowed for maintainer-local use; remote MCP remains no-go; action MCP remains no-go**.

This document separates Nivora's current local MCP value from the controls required before any remote exposure. It does not authorize remote MCP deployment.

## Decision Table

| Surface | Decision | Required Controls | Current Evidence | Missing Controls | Abuse Cases | Operational Risk | Recommendation |
|---|---|---|---|---|---|---|---|
| Local stdio MCP | go | local process trust, RBAC subject, redaction, audit, blocked actions | `cmd/nivora-mcp`, `make verify-mcp`, `internal/api/mcp`, 25+ scenarios after this gate | OS process trust remains local only | local operator asks AI for unsafe apply or secret | AI answer may be misread as execution authority | Keep enabled for maintainer workflows and demos with scripted data |
| Remote read-only MCP | no-go | OAuth/OIDC or service-account bearer, tenant filters, resource ownership checks, rate limit, timeout, response cap, pagination, remote audit metadata | RFC and local read-only resources exist | remote auth contract, tenant-filtered fixtures, rate limits, pagination, remote audit-chain test | IDOR against run IDs, broad audit search, log scraping | Could expose cross-tenant delivery metadata | Build remote-readiness tests before transport |
| Remote plan-only MCP | no-go | all read-only controls plus stronger plan-output labeling and abuse throttles | local plan-only tools return `mutated=false` | remote rate limits, plan artifact scope filters, remote denial audit | repeated large plan requests, social engineering apply/sync claims | Plans can be treated as authorization decisions | Keep disabled until read-only remote is proven |
| Remote action MCP | no-go | separate governance design, approvals, confirmations, scoped break-glass controls | blocked action tools are denied | intentionally not designed | apply, sync, rollback, approve, reject, rotate token, get secret, host deploy | AI-controlled mutation would cross the delivery trust boundary | Do not expose in the current model |

## Minimum Remote Read-Only Gate

| Control | Status | Evidence | Gap |
|---|---|---|---|
| Authentication | partial | auth/RBAC foundation exists | remote MCP bearer/OIDC contract missing |
| Service-account scope | partial | service account model exists | per-resource MCP scope filters missing |
| Runner token rejection | implemented for local MCP | MCP tests reject runner token subjects | must remain true for remote transport |
| Tenant filters | partial | tenant scope review exists | resource ID ownership checks incomplete |
| Rate limit | missing | no remote MCP transport | required before remote exposure |
| Request timeout | missing | no remote MCP transport | required before remote exposure |
| Response size cap | partial | log preview truncation exists | transport-wide cap missing |
| Pagination | partial | some API pagination exists | MCP audit/log pagination missing |
| Audit chain | partial | compliance recorder and audit hash tests exist | remote MCP Postgres audit-chain test missing |
| Prompt injection defense | beta | prompt templates and scenarios | expand adversarial corpus |
| Secret redaction | beta | sanitizer and scenario tests | keep central corpus current |
| Scenario coverage | beta | scenario/golden-answer validator | add enterprise abuse scenarios as tools grow |
| Incident response | foundation | security/runbook docs | remote MCP incident runbook missing |

## Non-Negotiable Remote No-Go Conditions

Remote MCP must stay disabled if any of these are true:

- anonymous clients can connect;
- runner tokens can call MCP;
- a resource can be read by ID without scope ownership checks;
- audit search can return cross-tenant data;
- logs are returned without pagination, truncation, and redaction;
- plan-only output can be confused with applied state;
- denied action attempts are not audited;
- rate limit, timeout, and response cap are absent;
- documentation implies remote MCP can mutate delivery state.

## Current Recommendation

Use MCP as a **local read-only and plan-only operator assistant**. Do not expose it remotely until tenant-scoped read-only resources, rate limits, response caps, pagination, and remote audit-chain proof are implemented and tested.
