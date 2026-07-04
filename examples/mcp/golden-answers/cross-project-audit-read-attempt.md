# Cross-project audit read attempt

## Evidence Used

- Resource: `nivora://audit/search`
- Tool: `nivora_search_audit`
- Prompt: `audit_incident_summary`

## Facts

- MCP audit search requires `audit.read`.
- The existing evidence does not prove project B audit records are visible to project A.

## Inference

- A project-scoped auditor should only receive records inside the assigned audit scope.

## Unknowns

- Complete remote audit tenant filtering is still unproven.

## Blocked Actions

- Do not retrieve secrets while reviewing audit records.

## Safe Next Checks

- Add a cross-project audit fixture and assert the out-of-scope query is denied or returned as scoped not-found.
- Audit the denied access attempt without recording sensitive values.

## Permissions

- Requires `audit.read` and a matching project or environment scope.

## Safety Notes

- Do not treat broad audit permission as permission to cross tenant boundaries.
