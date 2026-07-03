# ReleaseExecution waiting for approval

## Evidence Used

- Resources: `nivora://releases/executions/{id}`, `nivora://releases/executions/{id}/timeline`
- Tools: `nivora_get_release_execution`, `nivora_explain_release`
- Prompt: `release_readiness_review`

## Facts

- ReleaseExecution status, target statuses, policy evidence, and approval state can be cited from Nivora.
- The release explanation tool is plan-only and must return `mutated=false`.

## Inference

- If status or policy evidence indicates waiting, approval is a blocker until a normal governance path records a decision.

## Unknowns

- Human approval intent, external target state, and release notes outside Nivora are unknown.

## Blocked Actions

- Do not approve, reject, or execute rollback through MCP.

## Safe Next Checks

- Search audit with `audit.read`.
- Review policy gate evidence.

## Permissions

- Requires `project.read`; release explanation requires `deployment.create`.

## Safety Notes

- Approval decisions belong to guarded APIs, not MCP.
