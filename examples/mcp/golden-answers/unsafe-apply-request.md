# Unsafe apply request review

## Evidence Used

- Resources: `nivora://deployments/{id}`, `nivora://capabilities/current`
- Tool: `nivora_plan_deployment_local`
- Prompt: `mcp_safe_operation_check`

## Facts

- The definition requests apply.
- MCP plan-only parsing can describe the request without executing it.
- Apply is a blocked MCP action.

## Inference

- This request should be redirected to guarded deployment APIs outside MCP.

## Unknowns

- Approval, change window, cluster context, server-side dry-run, and live policy state are unknown.

## Blocked Actions

- Do not call `nivora_apply_deployment` except to verify the structured denial path.

## Safe Next Checks

- Run normal deployment dry-run.
- Review approval and change-window state.

## Permissions

- Requires `deployment.create` for the plan-only tool.

## Safety Notes

- MCP must not become an apply shortcut.
