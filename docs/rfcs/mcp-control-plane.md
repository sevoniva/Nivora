# RFC: MCP Control Plane Foundation

Status: accepted for foundation implementation.

Nivora is a good fit for Model Context Protocol because it already models delivery state across PipelineRuns, DeploymentRuns, releases, runners, policy gates, security findings, audit records, and capability status. MCP lets AI tools inspect that control-plane state without scraping logs or calling broad HTTP APIs directly.

This first MCP phase is intentionally conservative:

- **Read-only** resources expose current state.
- **Plan-only** tools explain failures, risks, readiness, and local policy/deployment plans.
- **Guarded action** tools are future work and are not exposed in this phase.

Nivora is still a hardened beta-candidate foundation and is not production-ready.

## Exposed Resources

- `nivora://capabilities/current`
- `nivora://system/runtime`
- `nivora://api/inventory`
- `nivora://pipelines/runs/{id}`
- `nivora://pipelines/runs/{id}/timeline`
- `nivora://pipelines/runs/{id}/logs`
- `nivora://deployments/{id}`
- `nivora://deployments/{id}/timeline`
- `nivora://deployments/{id}/resources`
- `nivora://deployments/{id}/health`
- `nivora://deployments/{id}/diff`
- `nivora://releases/{id}`
- `nivora://releases/executions/{id}`
- `nivora://releases/executions/{id}/timeline`
- `nivora://runners/summary`
- `nivora://security/summary`
- `nivora://audit/search`
- `nivora://plugins/capabilities`

## Exposed Tools

Read-only tools:

- `nivora_status`
- `nivora_get_pipeline_run`
- `nivora_get_pipeline_timeline`
- `nivora_get_deployment`
- `nivora_get_deployment_health`
- `nivora_get_deployment_diff`
- `nivora_get_release_execution`
- `nivora_get_runner_summary`
- `nivora_search_audit`
- `nivora_get_capability_status`

Plan-only tools:

- `nivora_explain_pipeline_failure`
- `nivora_explain_deployment`
- `nivora_explain_deployment_risk`
- `nivora_explain_release`
- `nivora_generate_release_readiness_summary`
- `nivora_evaluate_policy_local`
- `nivora_inspect_artifact`
- `nivora_inspect_artifact_reference`
- `nivora_plan_deployment_local`

Plan-only tools must return `mutated=false` when they produce planning, explanation, local policy evaluation, local artifact inspection, or local deployment-plan output.

## Blocked Actions

These operations are deliberately not MCP tools in this phase:

- deployment apply
- Argo CD sync
- rollback execution
- approval approve/reject
- secret retrieval
- token create/rotate/revoke
- runner registration
- remote host deployment
- Git push
- Kubernetes prune/delete

Requests for action-shaped tool names, including common aliases such as `nivora_approve_request`, `nivora_reject_request`, and `nivora_rollback_deployment`, return `mcp_action_not_allowed` with a future gate explanation.

## Auth And Scope

MCP does not bypass Nivora RBAC.

- Local stdio mode can use a configured local subject in local/dev environments.
- Production MCP requires token-backed identity.
- Runner tokens are rejected.
- Audit resources require `audit.read`.
- Plan-only tools require existing delivery permissions such as `deployment.create`.

Remote MCP with OAuth/OIDC is future work. The proposed next step is remote read-only MCP, documented in `remote-mcp-read-only.md`. Remote action MCP remains blocked.

## Audit And Redaction

MCP records operation-level audit/log events for resource reads, tool calls, denied tool calls, and rendered prompts. Local tests can use an in-memory recorder. Runtime wiring uses the compliance service recorder, so PostgreSQL runtime mode persists MCP audit through the existing hash-chained compliance audit path. Audit/log payloads must not contain raw tokens, token hashes, secret values, private keys, kubeconfigs, cloud keys, or Authorization headers.

Golden operator scenarios live in `examples/mcp/scenarios/` and are validated by MCP scenario tests. They cover current useful workflows and the claims AI must not make.

## Dependency Decision

This phase uses a minimal internal stdio JSON-RPC adapter instead of adding a new MCP SDK dependency. The protocol surface is limited to initialize, resources, tools, and prompts. A future phase can replace or extend the transport with the official SDK after dependency review.

## Non-Goals

- No production-ready claim.
- No remote MCP server.
- No marketplace behavior.
- No unsafe dynamic plugin loading.
- No execution authority through MCP.
