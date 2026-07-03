# MCP Server

Nivora includes a foundation-level local MCP server for AI-assisted inspection and planning.

It is read-only and plan-only. It does not execute deployments, sync Argo CD, retrieve secrets, rotate tokens, register runners, approve changes, reject changes, or run rollback.

## Run Locally

List tools:

```bash
go run ./cmd/nivora mcp list-tools --local
```

List resources:

```bash
go run ./cmd/nivora mcp list-resources --local
```

Serve stdio:

```bash
go run ./cmd/nivora-mcp --config configs/server.yaml --stdio
```

Or:

```bash
go run ./cmd/nivora mcp serve --config configs/server.yaml --stdio
```

`make mcp-serve-local` runs the same stdio server.

## JSON-RPC Example

Send one JSON-RPC object per line:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"nivora://capabilities/current"}}
```

## Verification

```bash
make verify-mcp
```

This target runs deterministic local MCP tests, builds `cmd/nivora-mcp`, checks local tool/resource listing, validates golden operator scenarios, and runs `scripts/smoke-mcp-local.sh`. It does not require Kubernetes, Argo CD, Harbor, cloud credentials, external registries, or external scanners.

Golden scenarios live in `examples/mcp/scenarios/`. Golden answers live in `examples/mcp/golden-answers/`. They describe what AI can safely answer, which MCP resources/tools/prompts provide evidence, what must be treated as unknown, and which action-shaped requests stay denied.

```bash
make validate-mcp-scenarios
make verify-ai-control-plane
```

The current corpus has 21 scenarios and 21 golden answers. Scenario tests create deterministic PipelineRun, DeploymentRun, and ReleaseExecution fixtures; they do not skip fixture-backed tools just because live external systems are absent.

## Configuration

Default:

```yaml
mcp:
  enabled: false
  mode: stdio
  readonly: true
  allow_plan_tools: true
  allow_action_tools: false
  subject_role: viewer
  token_env: NIVORA_MCP_TOKEN
```

Production mode requires explicit token-backed identity. Runner tokens are rejected.

## Audit

MCP records:

- `mcp.resource.read`
- `mcp.tool.called`
- `mcp.tool.denied`
- `mcp.prompt.rendered`

Local tests can use the in-memory recorder. Runtime wiring uses the compliance service recorder, so PostgreSQL runtime mode persists MCP audit through the existing compliance audit path and hash-chain tables.

## Limitations

- The transport is a minimal stdio JSON-RPC foundation.
- Remote MCP/OAuth is not implemented.
- Remote MCP-specific OAuth, rate limiting, and per-client scoping are future hardening.
- Full tenant filtering for every future remote MCP resource is not proven yet.
- Action MCP is not implemented and remains blocked for apply, sync, rollback, approval, token, secret, runner, host, Git, prune, and delete operations.
- MCP does not make Nivora production-ready.
