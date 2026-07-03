# Suspicious audit activity review

## Evidence Used

- Resource: `nivora://audit/search`
- Tool: `nivora_search_audit`
- Prompt: `audit_incident_summary`

## Facts

- Audit entries can be grouped by actor, action, subject, time, scope, and decision.
- Audit search requires `audit.read`.

## Inference

- Incident hypotheses must be labeled as hypotheses.

## Unknowns

- External actions not recorded by Nivora are unknown.

## Blocked Actions

- Do not retrieve secrets or rotate tokens through MCP.

## Safe Next Checks

- Narrow the audit query by actor, action, scope, or correlation ID.

## Permissions

- Requires `audit.read`.

## Safety Notes

- Do not expose raw Authorization headers, token hashes, or secret values from audit metadata.
