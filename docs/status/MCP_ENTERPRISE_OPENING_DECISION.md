# MCP Enterprise Opening Decision

Current decision: **local stdio MCP is allowed for maintainer-local use; remote read-only MCP has an experimental opt-in foundation; broad remote exposure and action MCP remain no-go**.

This document separates Nivora's current MCP value from the controls required before broad remote exposure. It does not authorize production remote MCP deployment.

## Decision Table

| Surface | Decision | Required Controls | Current Evidence | Missing Controls | Abuse Cases | Operational Risk | Recommendation |
|---|---|---|---|---|---|---|---|
| Local stdio MCP | go | local process trust, RBAC subject, redaction, audit, blocked actions, request timeout, response cap | `cmd/nivora-mcp`, `make verify-mcp`, `internal/api/mcp`, 29 scenarios, local timeout/cap tests | OS process trust remains local only | local operator asks AI for unsafe apply or secret | AI answer may be misread as execution authority | Keep enabled for maintainer workflows and demos with scripted data |
| Remote read-only MCP | experimental opt-in foundation | OAuth/OIDC or service-account bearer, tenant filters, resource ownership checks, rate limit, timeout, request/response cap, pagination, remote audit metadata | `POST /api/v1/mcp/rpc`, bearer/static-token route tests, runner-token denial, shared timeout/cap helpers, in-process per-subject rate-limit tests, and Postgres MCP hash-chain proof exist | broader OIDC/service-account contract tests, tenant-filtered fixtures for every resource, distributed rate limits, pagination, remote audit attribution tests | IDOR against run IDs, broad audit search, log scraping | Could expose cross-tenant delivery metadata if enabled too broadly | Keep disabled by default; use only controlled experiments until remaining tests pass |
| Remote plan-only MCP | experimental opt-in foundation | all read-only controls plus stronger plan-output labeling and abuse throttles | local and remote use the same plan-only tools that return `mutated=false` | distributed rate limits, plan artifact scope filters, remote denial audit | repeated large plan requests, social engineering apply/sync claims | Plans can be treated as authorization decisions | Keep disabled by default until read-only remote is stronger |
| Remote action MCP | no-go | separate governance design, approvals, confirmations, scoped break-glass controls | blocked action tools are denied | intentionally not designed | apply, sync, rollback, approve, reject, rotate token, get secret, host deploy | AI-controlled mutation would cross the delivery trust boundary | Do not expose in the current model |

## Minimum Remote Read-Only Gate

| Control | Status | Evidence | Gap |
|---|---|---|---|
| Authentication | partial | auth/RBAC foundation and remote bearer/static-token tests exist | broader OIDC/service-account scoped contract incomplete |
| Service-account scope | partial | service account model exists | per-resource MCP scope filters missing |
| Runner token rejection | implemented for local MCP | MCP tests reject runner token subjects | must remain true for remote transport |
| Tenant filters | partial | tenant scope review exists | resource ID ownership checks incomplete |
| Rate limit | foundation | in-process per-subject MCP JSON-RPC rate limit exists | distributed rate limit across replicas missing |
| Request timeout | foundation | shared MCP request timeout config exists | remote-specific timeout test depth limited |
| Response size cap | foundation | shared MCP response cap config and tests exist | pagination missing |
| Pagination | partial | some API pagination exists | MCP audit/log pagination missing |
| Audit chain | partial | compliance recorder, audit hash tests, and `TestPostgresIntegrationMCPAuditHashChain` exist | remote client attribution and tenant-safe audit search still missing |
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
- rate limit is absent or timeout and response cap are not enforced by the remote transport;
- documentation implies remote MCP can mutate delivery state.

## Current Recommendation

Use MCP as a **local read-only and plan-only operator assistant** and treat remote read-only MCP as an explicitly enabled experiment only. Do not broadly expose it until tenant-scoped read-only resources, distributed rate limits, pagination, and remote audit attribution are implemented and tested. Action MCP remains blocked.
