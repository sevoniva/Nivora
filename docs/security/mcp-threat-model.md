# MCP Threat Model

Current maturity: local stdio MCP foundation plus experimental opt-in remote read-only JSON-RPC. Remote MCP is disabled by default and is not ready for broad exposure. Nivora is not production-ready.

| Threat | Asset At Risk | Attacker Capability | Current Mitigation | Missing Mitigation | Required Test | Decision |
|---|---|---|---|---|---|---|
| Local stdio client reads broad state | delivery metadata, logs, audit summaries | local process can start MCP with configured subject | MCP disabled by default; RBAC checks; runner tokens rejected | stronger operator guidance for local token handling | local subject and runner-token tests | acceptable for local maintainer use |
| Remote MCP client overreach | tenant data and runtime posture | remote client obtains or reuses token | remote MCP requires bearer/OIDC/service-account auth, rejects runner tokens, and applies request/response caps | broader OIDC/service-account scoped tests, pagination, per-client rate limits, tenant filters for future resources | remote auth and tenant contract tests | allow controlled experiments only; block broad exposure |
| AI client compromise | control-plane read access | compromised AI client calls read resources repeatedly | read-only/plan-only tools; denied actions | token rotation process and remote client audit policy | rate-limit and token-scope tests for remote phase | needs hardening before remote |
| Prompt injection through logs/events | operator judgment and unsafe recommendations | attacker places instructions in logs or event messages | prompts require facts vs inference and safe checks; output redaction | stronger prompt-injection tests over log fixtures | golden scenario with hostile log text | needs hardening |
| Secret leakage through logs/audit | tokens, keys, kubeconfigs, passwords | secret-like content appears in log/audit fields | redaction helper; MCP output sanitization; log truncation | broader fixture corpus for secret-like strings | redaction regression tests | acceptable locally; keep testing |
| Runner token misuse | runner identity and job protocol | runner token used against MCP | resolver and server reject runner tokens | remote transport must preserve token type distinction | runner-token denial tests | acceptable locally |
| Confused deputy via AI tools | deployment authority | user asks AI to apply/sync/approve through MCP | blocked action tool names return `mcp_action_not_allowed` | future action-tier policy design | denied action matrix tests | action MCP blocked |
| Action escalation by alias | guarded operations | caller tries alternate tool names | denied aliases for apply, sync, rollback, approvals, tokens, runner, host, git, prune/delete | maintain denylist as new names appear | matrix coverage for blocked actions | acceptable for current catalog |
| Tenant data leakage | project/environment data | subject asks for resources outside scope | core RBAC is used; matrix documents tenant scope | per-resource tenant filters need deeper remote proof | tenant-scoped MCP resource tests in remote phase | needs hardening before remote |
| Audit tampering | evidence chain | attacker mutates audit records or bypasses recorder | runtime MCP recorder uses compliance audit path; `TestPostgresIntegrationMCPAuditHashChain` proves Postgres hash-chain persistence for MCP audit | remote client identity and per-client audit attribution are not implemented | remote MCP audit attribution and scope tests | needs hardening before remote |
| Replay/rate abuse | availability and audit volume | client repeats expensive reads | shared request timeout, request body cap, response cap, and in-process request rate limit exist | remote per-client distributed rate limits, request IDs, idempotency, deeper transport tests | remote DoS/rate-limit tests | harden before broad exposure |
| Resource enumeration | operational topology | client lists resources and IDs | catalog requires `project.read`; audit requires `audit.read` | per-tenant resource filters and pagination for remote | remote list filtering tests | needs hardening |
| Denial of service by large inputs/outputs | server and AI client memory | large requests, logs, or docs requested | request body cap; log truncation; JSON output sanitization; local MCP resource/tool and JSON-RPC response caps | remote pagination and transport proof | request/body truncation and transport max-response tests | acceptable locally; remote needs limits |
| Unsafe recommendations | operator safety | AI infers readiness or asks for action | prompts require unknowns, safe checks, no production-ready claims | scenario review corpus and golden expected answers | golden scenario tests | acceptable if prompts are used |

## Invariants

- MCP must not expose secret values, token hashes, private keys, kubeconfigs, cloud keys, or Authorization headers.
- Runner tokens must not authenticate to MCP.
- Plan-only tools must not mutate Nivora state and must return `mutated=false`.
- Action-shaped tools must remain denied in this phase.
- Remote MCP must remain disabled by default and must not be broadly exposed before scoped identity, tenant filters, stronger per-client rate limits, pagination, and remote audit tests exist.
