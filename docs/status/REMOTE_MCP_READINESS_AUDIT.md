# Remote MCP Readiness Audit

Current decision: **no-go for remote MCP implementation today**. Local stdio MCP is usable for maintainer workflows. Remote read-only MCP is a future candidate after auth, tenant filtering, limits, and audit proof are finished.

## Decision Matrix

| Area | Decision | Evidence | Blocker |
|---|---|---|---|
| Local stdio read-only MCP | go for local maintainer use | `cmd/nivora-mcp`, `make verify-mcp` | Local trust boundary only. |
| Local stdio plan-only MCP | go for local maintainer use | plan tools return `mutated=false` | Summaries are not execution authority. |
| Remote read-only MCP | conditional no-go | RFC exists | OAuth/OIDC contract, tenant filters, rate limits, pagination, and remote audit tests missing. |
| Remote plan-only MCP | no-go for now | plan-only local tests exist | Remote abuse controls are not implemented. |
| Remote action MCP | no-go | blocked action tools | Destructive actions are intentionally excluded. |

## Required Remote Design Controls

| Control | Required State | Current State |
|---|---|---|
| Transport | HTTP/SSE or equivalent with TLS termination | proposed only |
| Auth | bearer identity backed by existing Auth/RBAC | local token/dev foundation |
| Service account model | explicit permissions and scope | foundation exists, remote MCP tests missing |
| Runner token handling | reject all runner-token MCP calls | implemented and tested |
| Tenant scope | org/project/environment filters per resource/tool | partially modeled, not complete |
| Response limits | body size, log truncation, capped lists | local log truncation exists; remote caps missing |
| Rate limits | per subject/client | missing |
| Audit | actor, auth mode, client, resource/tool, decision, scope, request/correlation IDs | local compliance recorder exists; remote-style Postgres test missing |
| Secrets | never return values or token hashes | implemented in redaction tests |

## Remote Resource Readiness

| Resource Class | Remote Candidate | Required Before Exposure |
|---|---|---|
| capability/runtime/API inventory | yes | auth, audit, response caps |
| pipeline/deployment/release records | yes | tenant filters and pagination |
| logs/events/audit | limited | strict truncation, pagination, audit scope filters |
| runner summary | limited | environment/project scope filters |
| security summary | yes | tenant filters |
| plugin capabilities | yes | no dynamic execution |

## Non-Goals

- No apply/sync/rollback/approval/token/secret/runner/host/Git/Kubernetes delete actions.
- No anonymous remote MCP.
- No production exposure claim.
- No remote MCP until operational docs and tests are in place.

## Next Steps

1. Add remote MCP auth contract tests.
2. Add tenant-scoped fixture tests for every resource/tool.
3. Add response-size, pagination, and rate-limit checks.
4. Add dedicated Postgres MCP audit-chain integration proof.
5. Update deployment docs only after the above are green.
