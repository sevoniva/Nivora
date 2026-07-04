# RFC: Remote Read-Only MCP

Status: experimental foundation implemented; not broadly opened.

## Recommendation

Local stdio MCP can be used by maintainers today for read-only and plan-only workflows. A first remote read-only JSON-RPC foundation is available at `POST /api/v1/mcp/rpc` only when explicitly enabled with `mcp.enabled=true` and `mcp.mode=http`. Remote action MCP remains blocked.

## Transport Options

| Option | Notes | Decision |
|---|---|---|
| HTTP streaming/SSE MCP endpoint in `nivora-server` | Fits existing server ownership and RBAC middleware | preferred future path |
| Separate `nivora-mcp` sidecar | Isolates MCP runtime but adds deployment wiring | acceptable later |
| Cloud-hosted MCP proxy | Could be useful for managed deployments | not in current scope |

## Auth Model

- Require bearer-token identity backed by existing auth/RBAC.
- Support service accounts with explicit permissions.
- Reject runner tokens.
- Do not support anonymous remote MCP.
- OIDC/OAuth should be provider-neutral and configured through existing auth config.

Suggested flow:

1. Client authenticates with an existing bearer, service-account, or OIDC token.
2. Server resolves the subject through the normal auth service.
3. MCP request handling evaluates the same permissions used by local MCP.
4. Runner-token subjects are rejected before resource/tool dispatch.
5. Every response records an audit event with subject, auth mode, resource/tool/prompt name, decision, request ID, and correlation ID when available.

## Scope And Permission Mapping

Remote MCP resources must map to existing Nivora permissions:

- ordinary read resources: `project.read`
- audit search: `audit.read`
- plan-only tools: `deployment.create` or a narrower future `mcp.plan`
- runner summary: project or environment-scoped read

Every resource must filter by org/project/environment where the underlying store supports it. If a resource cannot be scoped, it should not be remote-exposed.

## Limits

Required before broad exposure:

- request body size limit
- response size limit
- log preview truncation
- per-client rate limit
- per-subject audit correlation
- timeout per MCP call
- pagination or capped result sets for audit/log/event resources

Initial limits should be conservative. Logs and audit search should require pagination from the first remote release; unbounded list responses should not be remote-exposed.

## Redaction And Audit

Remote MCP responses must use the same redaction path as local MCP. Audit records must include:

- actor
- auth mode
- MCP client id if available
- resource/tool/prompt name
- decision
- scope
- request id
- correlation id
- timestamp

Audit must not include raw tokens, token hashes, private keys, kubeconfigs, cloud keys, passwords, or Authorization headers.

## Operator Deployment Model

Remote MCP should be disabled by default. Operators should enable it only with:

- production-safe auth
- Postgres-backed audit
- TLS-terminating ingress or equivalent transport security
- explicit allowed client configuration
- rate limits
- monitoring and runbook entries

## Non-Goals

- No apply/sync/rollback/approve/reject tools.
- No secret retrieval.
- No runner registration or token rotation.
- No Git push.
- No Kubernetes prune/delete.
- No production-ready claim.

## Blockers Before Broad Exposure

1. Broader OAuth/OIDC contract tests beyond the current bearer/static-token route coverage.
2. More tenant-scoped resource filtering tests for every future MCP resource family.
3. Stronger per-client remote rate limits and pagination for large event/log/audit result sets.
4. Remote MCP audit attribution tests for client identity, subject scope, request IDs, and tenant-safe audit search. The local/Postgres hash-chain path is already covered by `TestPostgresIntegrationMCPAuditHashChain`.
5. Golden scenario prompt-injection tests for remote-specific usage.
6. Operator docs, ingress/TLS guidance, and threat-model update.

Current audit status is tracked in `docs/status/REMOTE_MCP_READINESS_AUDIT.md`. The current decision is that remote read-only MCP can be tested as an explicitly enabled foundation, but it is not ready for broad production exposure. Remote action MCP remains no-go.
