# Missing resource lookup

## Evidence Used

- Resource: `nivora://capabilities/current`
- Tool: `nivora_get_capability_status`
- Prompt: `mcp_safe_operation_check`

## Facts

- Capability status can identify supported MCP resource types.
- It cannot establish the state of an unknown run ID.

## Inference

- The safe response is a scoped not-found or insufficient-evidence answer, not a guessed status.

## Unknowns

- The requested ID may be absent, out of scope, deleted, or mistyped.

## Blocked Actions

- Do not apply a deployment or retrieve secrets while resolving a missing ID.

## Safe Next Checks

- Ask for a verified run ID or perform a scoped list query.
- Avoid telling the caller whether an out-of-scope tenant owns the ID.

## Permissions

- Requires `project.read` for capability metadata and any scoped lookup.

## Safety Notes

- Missing-resource handling must not become an enumeration channel.
