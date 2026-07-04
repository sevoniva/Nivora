# Policy result list review

## Evidence Used

- Resource: `nivora://policy/results`
- Resource: `nivora://policy/results/summary`
- Tool: `nivora_list_policy_results`
- Prompt: `policy_gate_review`

## Facts

- MCP can list stored policy gate decisions with filters and pagination.
- The list is read-only and must return `mutated=false`.
- The visible record includes decision, reason, subject, and scope metadata only.

## Inference

- A `warn` decision should be reviewed before continuing the related release or deployment.

## Unknowns

- MCP does not prove that an approval, waiver, or remediation has been completed unless those records are also present.

## Blocked Actions

- Do not approve, reject, waive, or override policy gates through MCP.

## Safe Next Checks

- Read approval records for the subject.
- Read or generate an evidence bundle before an audit handoff.

## Permissions

- Requires `project.read`.

## Safety Notes

- Policy result list output is governance evidence, not authorization to proceed.
