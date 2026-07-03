# Host deployment safety review

## Evidence Used

- Resource: `nivora://capabilities/current`
- Tool: `nivora_plan_deployment_local`
- Prompt: `mcp_safe_operation_check`

## Facts

- Host deployment intent can be parsed without uploading artifacts or running commands.
- Remote host deployment is blocked through MCP.

## Inference

- Host deployment should remain guarded and isolated outside MCP.

## Unknowns

- SSH credential validity, host reachability, disk state, and service health are unknown.

## Blocked Actions

- Do not deploy to remote hosts or retrieve SSH credentials through MCP.

## Safe Next Checks

- Run host dry-run commands outside MCP.
- Review host security and credential scope.

## Permissions

- Requires `deployment.create`.

## Safety Notes

- Shell or SSH execution is not a sandbox guarantee.
