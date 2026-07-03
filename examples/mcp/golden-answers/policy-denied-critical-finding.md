# Policy denial due to critical finding

## Evidence Used

- Resource: `nivora://security/summary`
- Tool: `nivora_evaluate_policy_local`
- Prompt: `policy_gate_review`

## Facts

- Local policy evaluation can identify manifest risks such as privileged containers.
- The local evaluator is non-persistent and must return `mutated=false`.

## Inference

- High or critical findings can justify deny, warn, or approval-required decisions depending on policy configuration.

## Unknowns

- Scanner provenance and persisted policy-result history may be missing.

## Blocked Actions

- Do not apply deployments or approve requests through MCP.

## Safe Next Checks

- Read persisted security summary.
- Review release or deployment policy evidence.

## Permissions

- Requires `deployment.create`.

## Safety Notes

- Do not claim the local MCP evaluator stored a policy result.
