# Tenant scope uncertainty review

## Evidence Used

- Resources: `nivora://capabilities/current`, `nivora://audit/search`
- Tools: `nivora_get_capability_status`, `nivora_search_audit`
- Prompt: `audit_incident_summary`

## Facts

- MCP enforces subject authentication and permissions.
- Audit search requires `audit.read`.
- Runner tokens are rejected from MCP paths.

## Inference

- Full tenant-filtered remote MCP remains a blocker unless each resource and tool applies explicit scope filters.

## Unknowns

- Cross-project filtering for every MCP resource and plan-only tool is not proven by local stdio behavior alone.

## Blocked Actions

- Do not retrieve secrets through MCP.

## Safe Next Checks

- Add route-specific tenant fixture tests before remote MCP exposure.
- Review the MCP tenant scope document.

## Permissions

- Requires `project.read` for capability status and `audit.read` for audit evidence.

## Safety Notes

- Do not claim complete multi-tenant MCP isolation until remote-scope enforcement is tested.
