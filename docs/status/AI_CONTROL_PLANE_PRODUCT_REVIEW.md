# AI Control Plane Product Review

Current maturity: hardened beta-candidate foundation, not production-ready.

## Summary

Nivora is a strong fit for AI-assisted operations because it already keeps delivery state in one control-plane model: PipelineRuns, DeploymentRuns, ReleaseExecutions, runners, policy evidence, artifacts, security findings, audit records, capability status, and runtime posture.

The current MCP surface is useful for **inspection, diagnosis, readiness review, and planning**. It is not a safe place for execution authority. Local stdio MCP can be used for maintainer workflows now. Remote MCP should start as read-only only after OAuth/OIDC, tenant filters, rate limits, and remote audit tests exist. Action MCP should stay blocked.

## Operator Questions AI Can Answer Today

| Workflow | Grounded Evidence | Current Usefulness | Limits |
|---|---|---|---|
| Failed PipelineRun diagnosis | PipelineRun, timeline, logs, runtime status, runner summary | useful for local/foundation debugging | cannot prove host OS state or external service state |
| DeploymentRun health and diff review | DeploymentRun, resources, health, diff, timeline | useful for risk review and dry-run explanation | live Kubernetes truth is not guaranteed unless captured by Nivora |
| Release readiness review | ReleaseExecution, target states, timeline, policy/security/approval fields | useful for listing blockers before human action | cannot approve, deploy, roll back, or prove external target health |
| Policy denial explanation | security summary, local policy evaluator, policy prompt | useful for explaining why a finding blocks or warns | local MCP evaluator is non-persistent and must not be treated as stored evidence |
| Runner offline investigation | runner summary and runtime status | useful for first triage | shell executor isolation and host state remain operator responsibilities |
| Suspicious audit activity review | audit search with `audit.read` | useful for actor/action/time grouping | does not prove external-system actions not recorded in Nivora |
| Artifact immutability check | artifact inspect plan-only tool and capability state | useful for latest-tag/digest-pinned explanation | local inspect does not resolve registry digest without configured artifact flow |
| Kubernetes apply safety review | deployment plan/diff/health and denied action tools | useful for explaining why apply/prune/delete stay blocked | cannot execute apply through MCP |
| Argo sync safety review | deployment/GitOps state and denied action tools | useful for explaining guarded sync requirements | cannot execute sync through MCP |
| Host deploy safety review | deployment plan and denied host action tools | useful for explaining remote-host guardrails | cannot run remote host commands through MCP |
| Production install posture review | capability status, runtime summary, docs, smoke script output | useful for checklist-style review | cannot prove live restore, scale, or cluster installation without external drills |

## Where AI Would Overstate

| Overclaim | Why It Is Unsafe | Correct Wording |
|---|---|---|
| "The cluster is healthy." | MCP reads Nivora state, not arbitrary live cluster truth. | "Nivora recorded this health summary; live cluster state is unknown unless captured." |
| "This release is ready to deploy." | Approvals, policy, target health, and rollback evidence may be incomplete. | "These are the visible blockers and unknowns before deployment." |
| "I can apply or roll back this deployment." | Action tools are denied in this phase. | "MCP can explain and plan; execution must use guarded control-plane APIs." |
| "The runner is sandboxed." | Shell executor is hardened but not an OS-level sandbox. | "Runner isolation must be enforced by the operator environment." |
| "This artifact digest was resolved." | Local inspect can parse references without registry resolution. | "The reference is digest-pinned or mutable according to parsing; registry resolution is separate." |
| "Nivora is production-ready." | Status docs list remaining blockers. | "Nivora is a hardened beta-candidate foundation, not production-ready." |

## Scenario Coverage

Golden MCP scenarios live under `examples/mcp/scenarios/`:

- failed pipeline with redacted logs
- deployment with degraded resource
- release waiting for approval
- runner offline after claim
- policy denied due critical finding
- artifact latest warning
- audit incident search
- production config unsafe review

Each scenario records the operator question, fixture assumptions, MCP resources/tools/prompts, safe facts, allowed inference, unknowns, forbidden claims, safe next checks, and blocked actions. `internal/api/mcp/scenario_test.go` loads those files and verifies catalog coverage, prompt safety text, denied action behavior, redaction, and plan-only `mutated=false` output for locally executable plan tools.

## Product Direction

The next product step should be an AI operations layer, not broad new deployment integrations:

1. Keep local stdio MCP stable for maintainers.
2. Add remote read-only MCP only after the RFC blockers are resolved.
3. Keep action MCP blocked until policy, approval, confirmation, rollback evidence, tenant scoping, and independent audit are proven.

