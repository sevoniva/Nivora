# Remote MCP Readiness Audit

Current decision: **experimental go for explicitly enabled remote read-only MCP foundation; no-go for broad production exposure or action MCP**. Local stdio MCP is usable for maintainer workflows. Remote read-only MCP has a minimal HTTP JSON-RPC endpoint, but it remains disabled by default and needs more tenant, pagination, and operational proof before broad exposure.

## Decision Matrix

| Area | Decision | Evidence | Blocker |
|---|---|---|---|
| Local stdio read-only MCP | go for local maintainer use | `cmd/nivora-mcp`, `make verify-mcp` | Local trust boundary only. |
| Local stdio plan-only MCP | go for local maintainer use | plan tools return `mutated=false` | Summaries are not execution authority. |
| Remote read-only MCP | experimental foundation | `POST /api/v1/mcp/rpc`; bearer/static-token route tests; runner-token rejection; request body cap; JSON-RPC response cap; in-process per-subject rate limit; event/log/audit/artifact/security/evidence list pagination; remote MCP audit attribution for allowed calls and auth-boundary denials; blocked action denial; OpenAPI contract | Broader OIDC coverage, distributed rate limits, remaining list-like resource pagination, and operator deployment docs remain incomplete. |
| Remote plan-only MCP | experimental foundation | plan-only local tests exist and remote JSON-RPC uses the same server dispatch | Remote abuse controls and result-size pagination need more proof. |
| Remote action MCP | no-go | blocked action tools | Destructive actions are intentionally excluded. |

## Required Remote Design Controls

| Control | Required State | Current State |
|---|---|---|
| Transport | HTTP/SSE or equivalent with TLS termination | minimal HTTP JSON-RPC endpoint; TLS/ingress guidance still future |
| Auth | bearer identity backed by existing Auth/RBAC | bearer/static-token tests exist; OIDC-specific remote contract still future |
| Service account model | explicit permissions and scope | foundation exists; remote MCP uses existing bearer subject |
| Runner token handling | reject all runner-token MCP calls | implemented and tested |
| Tenant scope | org/project/environment filters per resource/tool | partially modeled, not complete |
| Response limits | body size, log truncation, capped lists | local and remote JSON-RPC response cap exists; event/log/audit/artifact/security/evidence list tools support limit/offset pagination |
| Request timeout | per request timeout | shared JSON-RPC timeout path exists |
| Rate limits | per subject/client | in-process per-subject JSON-RPC rate limit exists; distributed limit missing |
| Audit | actor, auth mode, client, resource/tool, decision, scope, request/correlation IDs | compliance recorder exists; remote bearer calls and auth-boundary denials record actor/operation/decision plus request ID, correlation ID, MCP client ID, transport, and remote address; `TestPostgresIntegrationMCPAuditHashChain` proves Postgres hash-chain persistence | auth mode may be redacted by the current sanitizer; distributed audit correlation across replicas remains future work |
| Secrets | never return values or token hashes | implemented in redaction tests |

## Remote Resource Readiness

| Resource Class | Remote Candidate | Required Before Exposure |
|---|---|---|
| capability/runtime/API inventory | yes | auth, audit, remote response cap and timeout proof |
| pipeline/deployment/release records | yes | tenant filters and pagination |
| logs/events/audit | limited | strict truncation, pagination, audit scope filters |
| runner summary | limited | environment/project scope filters |
| security summary | yes | tenant filters |
| plugin capabilities | yes | no dynamic execution |

## Non-Goals

- No apply/sync/rollback/approval/token/secret/runner/host/Git/Kubernetes delete actions.
- No anonymous remote MCP.
- No production exposure claim.
- No broad remote MCP exposure until operational docs and remaining tests are in place.

## Next Steps

1. Expand remote MCP auth contract tests for OIDC and service-account scoped tokens.
2. Add tenant-scoped fixture tests for every resource/tool.
3. Add remote response-size, request-timeout, pagination, and rate-limit checks.
4. Add remaining tenant-scope contract tests for every remote resource/tool.
5. Update deployment docs only after the above are green.
