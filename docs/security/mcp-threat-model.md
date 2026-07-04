# MCP Threat Model

Current maturity: local stdio MCP foundation. Remote MCP is not implemented. Nivora is not production-ready.

| Threat | Asset At Risk | Attacker Capability | Current Mitigation | Missing Mitigation | Required Test | Decision |
|---|---|---|---|---|---|---|
| Local stdio client reads broad state | delivery metadata, logs, audit summaries | local process can start MCP with configured subject | MCP disabled by default; RBAC checks; runner tokens rejected | stronger operator guidance for local token handling | local subject and runner-token tests | acceptable for local maintainer use |
| Future remote MCP client overreach | tenant data and runtime posture | remote client obtains or reuses token | remote MCP not implemented | OAuth/OIDC, service-account scopes, rate limits, tenant filters | remote auth contract tests | block remote MCP until hardened |
| AI client compromise | control-plane read access | compromised AI client calls read resources repeatedly | read-only/plan-only tools; denied actions | token rotation process and remote client audit policy | rate-limit and token-scope tests for remote phase | needs hardening before remote |
| Prompt injection through logs/events | operator judgment and unsafe recommendations | attacker places instructions in logs or event messages | prompts require facts vs inference and safe checks; output redaction | stronger prompt-injection tests over log fixtures | golden scenario with hostile log text | needs hardening |
| Secret leakage through logs/audit | tokens, keys, kubeconfigs, passwords | secret-like content appears in log/audit fields | redaction helper; MCP output sanitization; log truncation | broader fixture corpus for secret-like strings | redaction regression tests | acceptable locally; keep testing |
| Runner token misuse | runner identity and job protocol | runner token used against MCP | resolver and server reject runner tokens | remote transport must preserve token type distinction | runner-token denial tests | acceptable locally |
| Confused deputy via AI tools | deployment authority | user asks AI to apply/sync/approve through MCP | blocked action tool names return `mcp_action_not_allowed` | future action-tier policy design | denied action matrix tests | action MCP blocked |
| Action escalation by alias | guarded operations | caller tries alternate tool names | denied aliases for apply, sync, rollback, approvals, tokens, runner, host, git, prune/delete | maintain denylist as new names appear | matrix coverage for blocked actions | acceptable for current catalog |
| Tenant data leakage | project/environment data | subject asks for resources outside scope | core RBAC is used; matrix documents tenant scope | per-resource tenant filters need deeper remote proof | tenant-scoped MCP resource tests in remote phase | needs hardening before remote |
| Audit tampering | evidence chain | attacker mutates audit records or bypasses recorder | runtime MCP recorder uses compliance audit path; Postgres path hash-chains governance audit | CI hash-chain verification for MCP with real Postgres is limited | Postgres MCP audit chain integration test | needs hardening |
| Replay/rate abuse | availability and audit volume | client repeats expensive reads | local stdio only; local request timeout and response cap exist | remote rate limits, request IDs, idempotency, transport-level body limits | remote DoS/rate-limit tests | block remote MCP until designed |
| Resource enumeration | operational topology | client lists resources and IDs | catalog requires `project.read`; audit requires `audit.read` | per-tenant resource filters and pagination for remote | remote list filtering tests | needs hardening |
| Denial of service by large outputs | server and AI client memory | large logs or docs requested | log truncation; JSON output sanitization; local MCP response cap | remote pagination and transport-level cap proof | truncation and max response tests | acceptable locally; remote needs limits |
| Unsafe recommendations | operator safety | AI infers readiness or asks for action | prompts require unknowns, safe checks, no production-ready claims | scenario review corpus and golden expected answers | golden scenario tests | acceptable if prompts are used |

## Invariants

- MCP must not expose secret values, token hashes, private keys, kubeconfigs, cloud keys, or Authorization headers.
- Runner tokens must not authenticate to MCP.
- Plan-only tools must not mutate Nivora state and must return `mutated=false`.
- Action-shaped tools must remain denied in this phase.
- Remote MCP must not ship before scoped identity, tenant filters, rate limits, and remote audit tests exist.
