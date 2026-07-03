# Runner token boundary concern

## Evidence Used

- Resources: `nivora://audit/search`, `nivora://runners/summary`
- Tool: `nivora_search_audit`
- Prompt: `runner_fleet_health_review`

## Facts

- Runner tokens are rejected from MCP control-plane resources and tools.
- Audit search requires `audit.read`.

## Inference

- MCP is not a runner administration channel.

## Unknowns

- Whether a specific runner token is revoked requires token metadata outside MCP.

## Blocked Actions

- Do not retrieve secrets, rotate tokens, or register runners through MCP.

## Safe Next Checks

- Review runner-token audit records with an audit-capable subject.
- Use guarded runner-token APIs outside MCP if rotation is needed.

## Permissions

- Requires `audit.read` for audit evidence.

## Safety Notes

- Never reveal runner token values or token hashes.
