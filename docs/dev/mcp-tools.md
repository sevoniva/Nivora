# MCP Tools

Nivora's MCP server is a local stdio foundation for AI-assisted inspection and planning. It is disabled by default and does not expose action tools.

## Safe Tool Classes

Read-only tools and resources expose control-plane state such as:

- system status and runtime recovery status
- runner summaries
- DeploymentRun and ReleaseExecution inspection
- release-bound artifact inventory and artifact-to-release bindings
- security and audit summaries
- persisted evidence bundles
- capability status

Plan-only tools may explain failures or produce local plans. They return `mutated=false` and must not perform apply, sync, rollback, approval, token, secret, runner, host, Git, prune, or delete operations.

## Permission Rules

MCP calls use the same role and permission model as the rest of Nivora's control plane:

- normal read tools require project read permissions
- audit and evidence tools require `audit.read`
- runner tokens are rejected for MCP administrative reads
- action-shaped tools are denied in this foundation phase

The permission matrix lives in `docs/security/MCP_PERMISSION_MATRIX.md`.

## Audit And Redaction

MCP records audit events for:

- `mcp.resource.read`
- `mcp.tool.called`
- `mcp.tool.denied`
- `mcp.prompt.rendered`

Responses are redacted before they are returned. MCP must not return secret values, token hashes, kubeconfigs, authorization headers, private keys, or raw credential payloads.

## Verification

Run:

```bash
make verify-mcp
make verify-ai-control-plane
```

These targets check local MCP tests, tool/resource catalogs, golden operator scenarios, denied action tools, and local stdio smoke behavior. They do not require external systems.

## Current Limits

Remote MCP is still a no-go. It needs OAuth or service-account auth, tenant scope enforcement, rate limits, response size caps, pagination, and remote audit tests before it can be considered for opening. Action MCP remains blocked.

See also: `docs/dev/mcp-server.md`.
