# Evidence bundle request through MCP

## Evidence Used

- Resources: `nivora://audit/search`, `nivora://releases/executions/{id}/timeline`
- Tools: `nivora_search_audit`, `nivora_get_release_execution`
- Prompts: `audit_incident_summary`, `release_readiness_review`

## Facts

- MCP can cite available audit metadata and release execution timeline data.
- Available MCP evidence is not automatically a complete compliance bundle.

## Inference

- The operator can use MCP to identify missing evidence references before exporting a bundle through governed API paths.

## Unknowns

- It is unknown whether all approvals, policy results, deployment plans, security findings, logs, and audit-chain checks are present.

## Blocked Actions

- Do not retrieve secrets or approve requests through MCP.

## Safe Next Checks

- Query persisted evidence bundles through the governed API.
- Verify audit-chain integrity before treating evidence as compliance-ready.

## Permissions

- Requires `audit.read` for audit evidence and `project.read` for release execution context.

## Safety Notes

- Do not claim a complete compliance bundle from partial MCP evidence.
