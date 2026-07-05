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

Read a resource:

```bash
go run ./cmd/nivora mcp read-resource nivora://runtime/recovery --local
```

Call a read-only tool:

```bash
go run ./cmd/nivora mcp call-tool nivora_get_runtime_recovery_status --local
go run ./cmd/nivora mcp call-tool nivora_list_security_findings --local --arg severity=High
```

Action-shaped tools stay blocked:

```bash
go run ./cmd/nivora mcp call-tool nivora_apply_deployment --local
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

The current corpus has 31 scenarios and 31 golden answers. Scenario tests create deterministic PipelineRun, DeploymentRun, ReleaseExecution, and security scan fixtures; they do not skip fixture-backed tools just because live external systems are absent.

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
  request_timeout: 15s
  max_request_bytes: 1048576
  max_response_bytes: 262144
  max_requests_per_minute: 120
```

Production mode requires explicit token-backed identity, request timeout, response cap, and a positive local stdio request rate limit. Runner tokens are rejected.

## Audit

MCP records:

- `mcp.resource.read`
- `mcp.tool.called`
- `mcp.tool.denied`
- `mcp.prompt.rendered`

Local tests can use the in-memory recorder. Runtime wiring uses the compliance service recorder, so PostgreSQL runtime mode persists MCP audit through the existing compliance audit path and hash-chain tables. `TestPostgresIntegrationMCPAuditHashChain` proves MCP resource and denied-tool audit entries are persisted with a hash chain when PostgreSQL integration tests are enabled.

## Limitations

- The primary transport is a minimal stdio JSON-RPC foundation.
- A minimal remote HTTP JSON-RPC foundation exists behind `mcp.enabled=true` and `mcp.mode=http`; it is experimental and disabled by default.
- Remote MCP requires bearer/service-account/OIDC-placeholder identity, rejects anonymous/dev/runner-token access, keeps action tools blocked, supports `limit`/`offset` pagination on list-like resource URIs and list tools, returns structured errors for unknown methods/resources/tools, and has route-level tests for request caps, response caps, and in-process per-subject rate limits.
- Remote MCP-specific OAuth depth, distributed rate limiting for multi-replica deployments, per-client scoping, operator exposure guidance, and broader future-resource tenant proof remain future hardening.
- Full tenant filtering for every future remote MCP resource is not proven yet.
- Action MCP is not implemented and remains blocked for apply, sync, rollback, approval, token, secret, runner, host, Git, prune, and delete operations.
- MCP does not make Nivora production-ready.
