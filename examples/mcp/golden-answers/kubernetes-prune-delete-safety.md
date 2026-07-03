# Kubernetes prune and delete safety review

## Evidence Used

- Resources: `nivora://deployments/{id}/resources`, `nivora://deployments/{id}/diff`
- Tool: `nivora_get_deployment_diff`
- Prompt: `mcp_safe_operation_check`

## Facts

- MCP can read resource inventory and diff summaries.
- Kubernetes prune and delete are blocked MCP actions.

## Inference

- Removed resources may require operator review, but MCP must not delete them.

## Unknowns

- Live owner references, finalizers, and cluster admission state are unknown.

## Blocked Actions

- Do not prune or delete Kubernetes resources through MCP.

## Safe Next Checks

- Review diff and rollback baseline through normal APIs.

## Permissions

- Requires `project.read`.

## Safety Notes

- Diff evidence is not deletion permission.
