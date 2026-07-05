# MCP Tools

Nivora's MCP server is a local stdio foundation for AI-assisted inspection and planning. An experimental remote read-only JSON-RPC endpoint can be enabled explicitly with `mcp.enabled=true` and `mcp.mode=http`. MCP is disabled by default and does not expose action tools.

In production mode, MCP must remain read-only in this foundation phase. Configuration validation rejects enabled MCP with `mcp.readonly=false`.

## Safe Tool Classes

Read-only tools and resources expose control-plane state such as:

- system status and runtime recovery status
- explicit runtime recovery summaries across pipeline, deployment, release, and outbox state
- organization, project, application, environment, repository, and release-target catalog summaries
- repository snapshots, static repository intelligence, DevOps plans, and readiness reviews
- stored Nivora Workflow summaries, plans, and guarded WorkflowRun metadata
- pipeline definition catalog reads
- runner summaries
- DeploymentRun and ReleaseExecution inspection
- filtered Release inventory
- filtered release-bound artifact inventory and artifact-to-release bindings
- security finding summaries, policy gate decision summaries, and audit summaries
- persisted evidence bundles
- capability status

Plan-only tools may explain failures or produce local plans. They return `mutated=false` and must not perform apply, sync, rollback, approval, token, secret, runner, host, Git, prune, or delete operations.

Repository and workflow prompts are also plan-only guidance. They must treat detected build/test/package commands as suggestions until a guarded workflow run is requested through the normal control-plane path.

## Permission Rules

MCP calls use the same role and permission model as the rest of Nivora's control plane:

- normal read tools require project read permissions
- audit and evidence tools require `audit.read`
- runner tokens are rejected for MCP administrative reads
- action-shaped tools are denied in this foundation phase

Remote MCP requires bearer, service-account, or OIDC authentication through the normal HTTP auth middleware. It does not accept anonymous/dev auth and rejects runner tokens.

The permission matrix lives in `docs/security/MCP_PERMISSION_MATRIX.md`.

## Audit And Redaction

MCP records audit events for:

- `mcp.resource.read`
- `mcp.tool.called`
- `mcp.tool.denied`
- `mcp.prompt.rendered`

Responses are redacted before they are returned and capped by `mcp.max_response_bytes`. Local stdio and remote HTTP JSON-RPC request bodies are capped by `mcp.max_request_bytes`, and request rate is limited by `mcp.max_requests_per_minute`. MCP must not return secret values, token hashes, kubeconfigs, authorization headers, private keys, or raw credential payloads.

## Verification

Run:

```bash
make verify-mcp
make verify-ai-control-plane
```

These targets check MCP tests, tool/resource catalogs, golden operator scenarios, denied action tools, and local stdio smoke behavior. They do not require external systems.

## Current Limits

Remote MCP is still experimental and off by default. The first HTTP JSON-RPC foundation is read-only/plan-only, requires bearer/service-account/OIDC auth, applies request and response caps, uses an in-process per-subject request limit, supports limit/offset pagination for event, log, audit, release, artifact, security finding, stored policy result, and evidence bundle list tools, rejects runner tokens, and records MCP audit events through the existing compliance recorder. It still needs remote deployment guidance, distributed rate limits for multi-replica deployments, broader pagination across every list-like resource, and more tenant-scope proof before it should be broadly exposed. Action MCP remains blocked.

See also: `docs/dev/mcp-server.md`.
