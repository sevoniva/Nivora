# Policy result summary review

## Evidence Used

- Resource: `nivora://policy/results/summary`
- Resource: `nivora://security/summary`
- Tool: `nivora_get_policy_result_summary`
- Prompt: `policy_gate_review`

## Facts

- MCP can summarize persisted policy gate decisions and the reasons attached to scan records.
- The summary is read-only and must return `mutated=false`.

## Inference

- Warn, deny, or approval-required decisions should be treated as governance evidence before moving a release or deployment forward.

## Unknowns

- MCP does not know whether a separate approval, waiver, or remediation was completed unless those records are present.

## Blocked Actions

- Do not approve or reject requests through MCP.

## Safe Next Checks

- Read approval records for the subject.
- Check release status and evidence bundle references.

## Permissions

- Requires `project.read`.

## Safety Notes

- MCP can explain policy posture but cannot override policy gates.
