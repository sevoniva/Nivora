# MCP Control Plane Architecture

MCP is an API adapter over existing Nivora use cases. It is not a domain concept and it does not own delivery state.

```text
MCP client
  -> cmd/nivora-mcp or nivora mcp serve
  -> internal/app/mcp
  -> internal/api/mcp
  -> existing usecase services
  -> stores/adapters through existing ports
```

The domain layer remains pure. Domain packages do not import MCP, JSON-RPC, HTTP, SDK, database, Kubernetes, Argo CD, cloud, or logging packages.

## Transport

The foundation transport is stdio with line-delimited JSON-RPC messages. Supported methods:

- `initialize`
- `resources/list`
- `resources/read`
- `tools/list`
- `tools/call`
- `prompts/list`
- `prompts/get`

Remote HTTP/SSE MCP and OAuth are future work.

## Service Wiring

`internal/app/mcp` builds the MCP server from the same runtime service constructors used elsewhere:

- Pipeline service
- Deployment service
- Artifact/release service
- Release orchestration service
- Security service
- Compliance service
- Auth service
- Plugin registry

MCP tools must call usecase methods. They must not query database tables directly.

## Permission Tiers

| Tier | Purpose | Examples | Current Status |
|---|---|---|---|
| Read-only | Inspect control-plane state | runtime, runs, timelines, health, diff, audit search | implemented |
| Plan-only | Explain or produce non-mutating plans | failure diagnosis, readiness review, policy local evaluation | implemented |
| Guarded action | Mutate state or external systems | apply, sync, rollback, approve/reject | not implemented |

Guarded action tools require a future design with confirmation, policy gates, scope checks, audit, and rollback evidence.

## Safety Invariants

- MCP is disabled by default.
- Production MCP requires token-backed identity.
- Runner tokens cannot use MCP.
- Action tools are not exposed.
- Secret-like values are redacted before JSON output.
- Logs are truncated.
- Missing records return structured errors instead of fake data.

## Audit

MCP emits operation-level audit/log decisions:

- `mcp.resource.read`
- `mcp.tool.called`
- `mcp.tool.denied`
- `mcp.prompt.rendered`

Local tests can use the in-memory MCP audit recorder. Runtime wiring uses a compliance-backed recorder, so PostgreSQL runtime mode persists MCP audit through the existing compliance audit path and hash-chain tables. This keeps MCP audit outside the domain layer while preserving durable evidence for production-like runtime configurations.
