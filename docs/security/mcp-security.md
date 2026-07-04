# MCP Security

MCP makes Nivora easier for AI tools to inspect. That also creates a new trust boundary: an MCP client can ask for broad control-plane context. The foundation therefore starts with read-only and plan-only behavior.

## Security Rules

- MCP does not bypass Nivora RBAC.
- MCP is disabled by default.
- Production MCP requires token-backed identity.
- Runner tokens cannot use MCP.
- Action tools are not exposed.
- Secret values are never returned.
- Token hashes are never returned.
- Kubeconfigs, private keys, Authorization headers, access keys, and bearer tokens are redacted.
- Logs are truncated before output.
- MCP JSON-RPC request bodies are capped by `mcp.max_request_bytes`.
- MCP responses are capped by `mcp.max_response_bytes`; resource/tool text returns a structured truncation object and over-limit JSON-RPC transport responses return a structured `mcp_response_too_large` error.
- MCP requests use `mcp.request_timeout` when configured.
- Local stdio JSON-RPC requests are limited by `mcp.max_requests_per_minute` when configured.

## Blocked Tool Classes

MCP must not execute:

- Kubernetes apply, prune, or delete
- Argo CD sync
- rollback execution
- approval approve/reject
- token create/rotate/revoke
- secret retrieval
- runner registration
- remote host deployment
- Git push

Blocked action-shaped calls return `mcp_action_not_allowed`.

## RBAC Expectations

| Subject | Expected MCP Behavior |
|---|---|
| anonymous production subject | denied |
| viewer | read allowed resources; plan tools denied |
| developer/maintainer/admin | read and allowed plan tools according to permissions |
| auditor | audit reads allowed; plan tools denied unless separately granted |
| service account | explicit token permissions only |
| runner token | denied |

## Audit Events

MCP emits:

- `mcp.resource.read`
- `mcp.tool.called`
- `mcp.tool.denied`
- `mcp.prompt.rendered`

Audit/log records include actor, auth mode, operation name, decision, reason, and time. They must not include secrets or token material.

Runtime wiring records MCP audit through the compliance service. In PostgreSQL runtime mode this enters the existing compliance audit path and hash-chain tables. Local tests can still use the in-memory recorder.

## Response and Timeout Controls

The local stdio MCP foundation enforces a configured request body cap, a response cap at both resource/tool text boundaries and the JSON-RPC response boundary, a request timeout, and a simple request rate limit. The default examples use `mcp.max_request_bytes: 1048576`, `mcp.max_response_bytes: 262144`, `mcp.request_timeout: 15s`, and `mcp.max_requests_per_minute: 120`. These controls reduce accidental abuse in local AI workflows, but they are not proof that a future remote MCP transport is safe. Remote MCP still needs authentication, tenant filters, per-client rate limits, pagination, remote transport tests, and remote audit tests before exposure.

See [MCP Permission Matrix](MCP_PERMISSION_MATRIX.md) for the resource, tool, prompt, permission, and audit-event mapping.

See [MCP Threat Model](mcp-threat-model.md) for the current trust-boundary review. Golden scenarios in `examples/mcp/scenarios/` and golden answers in `examples/mcp/golden-answers/` are validated by `internal/api/mcp/scenario_test.go` and `scripts/validate-mcp-scenarios.sh`. They keep AI answers grounded in facts, inference, unknowns, blocked actions, required permissions, and safety notes.

See [MCP Tenant Scope Review](MCP_TENANT_SCOPE_REVIEW.md) for the current cross-tenant exposure review. Local RBAC and runner-token denial are tested. Complete tenant-filtered remote MCP is not proven yet.

## Future Work

- Remote read-only MCP with OAuth/OIDC and tenant scoping.
- Tenant-aware scope filters for every resource URI.
- Guarded action tier with explicit confirmation, policy gates, and independent audit evidence.
