# GitOps sync safety review

## Evidence Used

- Resources: `nivora://capabilities/current`, `nivora://deployments/{id}/diff`
- Tool: `nivora_plan_deployment_local`
- Prompt: `mcp_safe_operation_check`

## Facts

- Plan-only parsing can identify GitOps sync intent.
- MCP does not expose Argo CD sync or Git push as allowed tools.

## Inference

- Sync should remain a guarded action outside MCP.

## Unknowns

- Live Argo CD application status, credentials, and Git remote state are unknown.

## Blocked Actions

- Do not sync Argo CD or push Git through MCP.

## Safe Next Checks

- Use read-only Argo CD status paths outside MCP if configured.

## Permissions

- Requires `deployment.create`.

## Safety Notes

- Sync intent in a definition is evidence, not authorization.
