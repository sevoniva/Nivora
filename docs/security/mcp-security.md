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

See [MCP Permission Matrix](MCP_PERMISSION_MATRIX.md) for the resource, tool, prompt, permission, and audit-event mapping.

## Future Work

- Remote MCP with OAuth/OIDC.
- Tenant-aware scope filters for every resource URI.
- Guarded action tier with explicit confirmation, policy gates, and independent audit evidence.
