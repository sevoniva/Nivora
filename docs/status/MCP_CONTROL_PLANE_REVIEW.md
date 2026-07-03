# MCP Control Plane Review

Current maturity: hardened beta-candidate foundation, not production-ready.

## Scope

This review covers the local stdio MCP surface added for AI-assisted inspection, planning, and diagnostics. It does not review a remote MCP server because remote transport and OAuth/OIDC are not implemented in this phase.

## What AI Can Safely Answer Today

| Question | Evidence Source | Confidence |
|---|---|---|
| What is the current Nivora maturity and capability status? | `nivora://capabilities/current`, `nivora_get_capability_status` | high |
| What does the runtime report about configured stores and runner state? | `nivora://system/runtime`, `nivora_status` | medium |
| What happened in a PipelineRun? | PipelineRun resource, timeline, logs, `diagnose_pipeline_run` prompt | medium |
| What does a DeploymentRun show for plan, resources, health, and diff? | DeploymentRun resources, health, diff, timeline, `diagnose_deployment_run` prompt | medium |
| Is a release execution ready for a guarded next step? | ReleaseExecution resource, timeline, `release_readiness_review` prompt | medium |
| What audit records match a subject or action? | `nivora_search_audit`, `nivora://audit/search` | high when subject has `audit.read` |
| What runner fleet issues are visible from control-plane state? | `nivora_get_runner_summary`, `runner_fleet_health_review` prompt | medium |
| Is a requested MCP action safe? | `mcp_safe_operation_check`, denied action contract | high |

## What AI Cannot Prove Today

| Gap | Reason | Risk |
|---|---|---|
| Live Kubernetes cluster truth | MCP only reads Nivora state and existing usecases; it does not query clusters directly. | AI could overstate live health if it ignores unknowns. |
| Production Argo CD sync state beyond modeled status | Real Argo automation remains guarded/foundation-only. | AI must not recommend sync execution through MCP. |
| Secret values or token material | Normal APIs and MCP intentionally never expose them. | AI must ask for SecretRef/CredentialRef metadata only. |
| Full production install readiness | Helm/Compose smoke checks exist, but live production-scale validation is not complete. | AI must keep not-production-ready language. |
| OS-level runner sandbox guarantees | Runner isolation profile docs exist, but shell executor is not a sandbox. | AI must recommend operator isolation checks instead of assuming safety. |
| Remote MCP security | Remote transport and OAuth/OIDC are future work. | Remote MCP should not be opened yet. |

## Most Useful MCP Resources For Operators

1. `nivora://capabilities/current`: anchors AI answers in honest current maturity.
2. `nivora://system/runtime`: quick runtime posture and store configuration.
3. `nivora://deployments/{id}/health`: concise deployment state.
4. `nivora://deployments/{id}/diff`: safe change summary.
5. `nivora://audit/search`: incident and governance evidence.
6. `nivora://runners/summary`: runner fleet triage.
7. `nivora://plugins/capabilities`: integration capability truth.

## Resources That Need Care

| Resource | Concern | Current Guard |
|---|---|---|
| Pipeline logs | Logs can contain accidental sensitive content. | MCP truncates and redacts output. |
| Audit search | Audit can expose operational metadata. | Requires `audit.read`; redaction is applied. |
| Runner summary | Runner metadata can reveal fleet topology. | Requires `project.read`; token hashes are not returned. |
| Capability/API inventory docs | Large text can be noisy. | Useful for grounding, but prompts should cite specific claims. |

## Prompt Quality Notes

The current prompt pack is useful because it forces:

- citations to Nivora resources or tools
- facts vs inference separation
- explicit unknowns
- safe read-only next checks
- no destructive action requests
- no production-ready claims

The next improvement should be smaller, more task-shaped prompts for:

- "deployment failed after apply"
- "runner offline after claim"
- "policy denied release"
- "audit suspicious actor"
- "capability truth check before docs update"

## Remote MCP Decision

Remote MCP should **not** be opened yet.

Required before remote MCP:

- OAuth/OIDC design with scoped MCP clients
- per-resource scope filters
- durable MCP audit in production mode
- rate limits and request body limits
- transport security and deployment docs
- MCP security contract tests for remote auth
- explicit maintainer approval for any future guarded action tier

## Recommended Next Phase

MCP 1.1 should stay local stdio and harden:

- durable MCP audit through compliance stores
- MCP permission matrix tests
- broader redaction tests
- local smoke scripts
- workflow examples
- CI coverage

MCP 1.2 can design remote read-only MCP, but should still exclude action tools.
