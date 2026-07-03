# Backup and restore posture review

## Evidence Used

- Resources: `nivora://capabilities/current`, `nivora://system/runtime`
- Tool: `nivora_get_capability_status`
- Prompt: `mcp_safe_operation_check`

## Facts

- Capability status can report whether backup and restore are documented, partial, or unproven.

## Inference

- Restore confidence remains limited without recent drill evidence.

## Unknowns

- Last successful restore timestamp, object-store restore coverage, and operator runbook execution are unknown.

## Blocked Actions

- Do not retrieve backup credentials through MCP.

## Safe Next Checks

- Run documented restore drill outside MCP.
- Review backup and restore operations docs.

## Permissions

- Requires `project.read`.

## Safety Notes

- Documentation is not the same as verified restore behavior.
