# Production config unsafe operation review

## Evidence Used

- Resources: `nivora://capabilities/current`, `nivora://system/runtime`
- Tool: `nivora_status`
- Prompt: `mcp_safe_operation_check`

## Facts

- MCP can show runtime summary and capability status.
- The status response reports `productionReady=false`.

## Inference

- Unsafe flags require config validation evidence outside MCP status alone.

## Unknowns

- Live Helm/Compose values, mounted secrets, and restore-drill evidence are unknown.

## Blocked Actions

- Do not remote host deploy, sync Argo CD, or apply Kubernetes through MCP.

## Safe Next Checks

- Run config validation and install smoke scripts.
- Review production profile docs.

## Permissions

- Requires `project.read`.

## Safety Notes

- MCP runtime summary alone cannot certify an installation for production.
