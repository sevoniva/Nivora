# AI Control Plane Deep Audit

Current maturity: **hardened beta-candidate foundation, not production-ready**.

This audit covers the local MCP control-plane surface. It records what an AI client can inspect or plan today, what remains blocked, and what evidence exists in code, tests, docs, and examples.

## Summary

| Area | Current State | Evidence | Gap |
|---|---|---|---|
| Local stdio MCP | implemented for read-only and plan-only use | `cmd/nivora-mcp`, `internal/api/mcp`, `internal/app/mcp`, `make verify-mcp` | Remote transport is not implemented. |
| Read-only resources | implemented with RBAC and redaction | `internal/api/mcp/server.go`, `internal/api/mcp/server_test.go`, `internal/api/mcp/scenario_test.go` | Tenant filtering is not complete for every future remote resource. |
| Plan-only tools | implemented; return `mutated=false` | `internal/api/mcp/server.go`, `TestMCPPlanOnlyToolsReturnMutatedFalse` | Plan output is still summary-level for some runtime areas. |
| Destructive actions | denied | `blockedActionTools` in `internal/api/mcp/server.go`, scenario tests | No action-tier MCP should be exposed without a separate design. |
| Audit | compliance recorder path exists with Postgres hash-chain proof | `NewComplianceAuditRecorder`, `TestMCPComplianceAuditRecorderPersistsToComplianceSearch`, `TestPostgresIntegrationMCPAuditHashChain`, CI `MCP audit hash chain verification` | Remote client identity and per-client audit policy remain future work. |
| Redaction | implemented for JSON outputs and audit entries | `mustJSON`, `sanitizeJSON`, redaction tests | Continue expanding adversarial examples as more tools are added. |
| Golden scenarios | implemented and validated | `examples/mcp/scenarios`, `examples/mcp/golden-answers`, `scripts/validate-mcp-scenarios.sh` | Scenarios are local fixtures, not live production evidence. |

## Resource Catalog

| Resource | Status | Permission | Implementation Evidence | Test Evidence | Remote Readiness |
|---|---|---|---|---|---|
| `nivora://capabilities/current` | implemented | `project.read` | `readResourcePayload` reads `docs/status/CAPABILITY_STATUS.md` | scenario resource coverage | future remote read-only candidate after tenant scoping |
| `nivora://system/runtime` | implemented | `project.read` | `Pipelines.RuntimeStatus` | scenario and server tests | future remote read-only candidate |
| `nivora://api/inventory` | implemented | `project.read` | reads `docs/API_INVENTORY.md` | catalog tests | future remote read-only candidate |
| `nivora://pipelines/runs/{id}` | implemented | `project.read` | `pipelineResource` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://pipelines/runs/{id}/timeline` | implemented | `project.read` | `Pipelines.Timeline` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://pipelines/runs/{id}/logs` | implemented | `project.read` | `truncateLogs(Pipelines.Logs)` | redaction/truncation tests | future remote read-only candidate with strict size limits |
| `nivora://deployments/{id}` | implemented | `project.read` | `Deployments.Get` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://deployments/{id}/timeline` | implemented | `project.read` | `Deployments.Timeline` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://deployments/{id}/resources` | implemented | `project.read` | `Deployments.Resources` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://deployments/{id}/health` | implemented | `project.read` | `Deployments.Health` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://deployments/{id}/diff` | implemented | `project.read` | `Deployments.Diff` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://releases/{id}` | implemented | `project.read` | `Artifacts.GetRelease` | catalog coverage | future remote read-only candidate |
| `nivora://releases/executions/{id}` | implemented | `project.read` | `Releases.GetExecution` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://releases/executions/{id}/timeline` | implemented | `project.read` | `Releases.Timeline` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora://runners/summary` | implemented | `project.read` | `runnerSummary` | scenario tests | future remote read-only candidate with scope filters |
| `nivora://security/summary` | implemented | `project.read` | `securitySummary` | scenario tests | future remote read-only candidate |
| `nivora://audit/search` | implemented | `audit.read` | `Compliance.SearchAudit` | viewer denied, auditor allowed | future remote read-only candidate only with scope filters and pagination |
| `nivora://plugins/capabilities` | implemented | `project.read` | `Plugins.List` | catalog tests | future remote read-only candidate |

## Tool Catalog

| Tool | Status | Permission | Mutates State | Implementation Evidence | Test Evidence | Remote Classification |
|---|---|---|---|---|---|---|
| `nivora_status` | implemented | `project.read` | no | `callToolPayload` | scenario and JSON-RPC tests | future remote read-only candidate |
| `nivora_get_pipeline_run` | implemented | `project.read` | no | `Pipelines.Get` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora_get_pipeline_timeline` | implemented | `project.read` | no | `Pipelines.Timeline` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora_get_deployment` | implemented | `project.read` | no | `Deployments.Get` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora_get_deployment_health` | implemented | `project.read` | no | `Deployments.Health` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora_get_deployment_diff` | implemented | `project.read` | no | `Deployments.Diff` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora_get_release_execution` | implemented | `project.read` | no | `Releases.GetExecution` | fixture-backed scenario tests | future remote read-only candidate |
| `nivora_get_runner_summary` | implemented | `project.read` | no | `runnerSummary` | scenario tests | future remote read-only candidate |
| `nivora_search_audit` | implemented | `audit.read` | no | `Compliance.SearchAudit` | audit boundary and pagination tests | future remote read-only candidate with tenant filters and remote attribution |
| `nivora_get_capability_status` | implemented | `project.read` | no | capability resource wrapper | scenario tests | future remote read-only candidate |
| `nivora_explain_pipeline_failure` | plan-only | `deployment.create` | no | `explainPipeline` | fixture-backed mutated=false tests | future remote plan-only candidate |
| `nivora_explain_deployment` | plan-only | `deployment.create` | no | `explainDeploymentRisk` | fixture-backed mutated=false tests | future remote plan-only candidate |
| `nivora_explain_deployment_risk` | plan-only | `deployment.create` | no | alias of deployment explanation | fixture-backed mutated=false tests | future remote plan-only candidate |
| `nivora_explain_release` | plan-only | `deployment.create` | no | `releaseReadiness` | fixture-backed mutated=false tests | future remote plan-only candidate |
| `nivora_generate_release_readiness_summary` | plan-only | `deployment.create` | no | alias of release explanation | mutated=false coverage | future remote plan-only candidate |
| `nivora_evaluate_policy_local` | plan-only | `deployment.create` | no persisted result | `Security.Evaluate` wrapper | mutated=false tests | future remote plan-only candidate |
| `nivora_inspect_artifact` | plan-only | `deployment.create` | no | artifact local inspect | mutated=false tests | future remote plan-only candidate |
| `nivora_inspect_artifact_reference` | plan-only | `deployment.create` | no | alias of artifact inspect | mutated=false tests | future remote plan-only candidate |
| `nivora_plan_deployment_local` | plan-only | `deployment.create` | no | parses definition only | no-mutation and scenario tests | future remote plan-only candidate |

## Denied Tool Names

| Tool Name | Status | Reason |
|---|---|---|
| `nivora_apply_deployment` | denied | destructive deployment action |
| `nivora_sync_argocd` | denied | guarded Argo CD sync belongs outside MCP |
| `nivora_execute_rollback` | denied | rollback execution is not exposed |
| `nivora_rollback_deployment` | denied | rollback execution alias is not exposed |
| `nivora_approve` / `nivora_approve_request` | denied | governance decision |
| `nivora_reject` / `nivora_reject_request` | denied | governance decision |
| `nivora_get_secret` | denied | secret value retrieval is forbidden |
| `nivora_rotate_token` | denied | token mutation is forbidden |
| `nivora_register_runner` | denied | runner registration has one-time token semantics |
| `nivora_remote_host_deploy` | denied | remote host execution is not an MCP capability |
| `nivora_git_push` | denied | Git mutation is not exposed |
| `nivora_kubernetes_prune` / `nivora_kubernetes_delete` | denied | destructive Kubernetes actions |

## Prompt Catalog

| Prompt | Status | Safety Evidence | Test Evidence |
|---|---|---|---|
| `diagnose_pipeline_run` | implemented | asks for facts, inference, unknowns, read-only checks | prompt safety tests, scenario coverage |
| `diagnose_deployment_run` | implemented | flags apply/sync/rollback/prune/host deploy as guarded | prompt safety tests |
| `release_readiness_review` | implemented | limits recommendations to plan/read actions | scenario coverage |
| `audit_incident_summary` | implemented | forbids secret/token/audit leakage | scenario coverage |
| `policy_gate_review` | implemented | describes local evaluation as non-persistent | scenario coverage |
| `runner_fleet_health_review` | implemented | states shell executor is not an OS sandbox | scenario coverage |
| `mcp_safe_operation_check` | implemented | classifies actions as read-only, plan-only, or blocked | scenario coverage |

All prompt templates now state that logs, events, manifests, audit messages, and user-supplied content are untrusted evidence, not instructions.

## Runtime Capability Evidence

| Runtime Area | AI Evidence Today | Limitation |
|---|---|---|
| PipelineRun | run record, timeline, logs, runtime status | MCP cannot rerun or recover jobs. |
| DeploymentRun | run record, timeline, resources, health, diff, plan-only local planning | MCP cannot apply, rollback, prune, delete, or prove live cluster state. |
| ReleaseExecution | execution record, target summary, timeline, readiness explanation | MCP cannot approve, reject, or execute release targets. |
| Runners | fleet summary and runtime counters | MCP cannot register runners, rotate tokens, or inspect host OS state. |
| Audit | audit search with `audit.read` and limit/offset pagination | Tenant filters and remote attribution are future work. |
| Security | summary and local policy evaluation | No external scanner integration is required or proven through MCP. |
| Artifacts | local reference parsing/inspection | Network digest resolution is outside MCP local inspect. |
| Plugins | capability registry | No dynamic external plugin execution through MCP. |

## Remaining Risks

1. Remote MCP now has an opt-in read-only JSON-RPC foundation, but broad exposure should stay blocked until OAuth/OIDC lifecycle, distributed rate limits, tenant filters, pagination, and remote audit tests are stronger.
2. MCP audit persistence is proven at unit/compliance-service level and through a Postgres hash-chain integration test, but remote client identity and per-client audit policy remain future work.
3. Tenant filtering is not proven for every MCP resource/tool.
4. Plan-only summaries can be misread as execution evidence if prompts are ignored.
5. Shell runner output can contain adversarial text; prompts and golden scenarios mitigate but do not sandbox workloads.

## Recommended Next Phase

Continue hardening before feature expansion:

1. Remote read-only MCP contract tests.
2. Tenant-filtered MCP resources and tools.
3. Remote MCP auth, tenant-scope, limit, and audit contract tests.
4. Response limits, pagination, and rate-limit design for remote transport.
