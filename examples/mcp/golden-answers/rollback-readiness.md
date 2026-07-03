# Rollback readiness review

## Evidence Used

- Resources: `nivora://deployments/{id}`, `nivora://deployments/{id}/resources`, `nivora://deployments/{id}/diff`
- Tool: `nivora_explain_deployment_risk`
- Prompt: `diagnose_deployment_run`

## Facts

- Deployment plan, resource inventory, and diff summary can support a rollback-readiness review.
- The explanation tool is plan-only and must return `mutated=false`.

## Inference

- Rollback readiness is partial unless stored baseline and current live state both support it.

## Unknowns

- Live cluster state and whether the rollback baseline still matches current resources are unknown.

## Blocked Actions

- Do not execute rollback through MCP.

## Safe Next Checks

- Inspect the stored rollback plan through normal APIs.
- Require guarded confirmation outside MCP for any rollback execution.

## Permissions

- Requires `project.read`; risk explanation requires `deployment.create`.

## Safety Notes

- Rollback planning evidence is not the same as rollback execution.
