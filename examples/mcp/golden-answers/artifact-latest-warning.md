# Artifact latest tag warning

## Evidence Used

- Resource: `nivora://capabilities/current`
- Tools: `nivora_inspect_artifact`, `nivora_inspect_artifact_reference`
- Prompt: `policy_gate_review`

## Facts

- Local artifact inspection can parse references and identify mutable tag or digest-pinned form.
- The artifact inspection tools must return `mutated=false`.

## Inference

- A `latest` tag is weak evidence for immutable release identity.

## Unknowns

- Registry manifest digest, media type, and size are unknown unless resolved through configured registry APIs.

## Blocked Actions

- Do not retrieve registry secrets or rotate tokens through MCP.

## Safe Next Checks

- Use normal artifact resolve flow with configured credentials if available.

## Permissions

- Requires `deployment.create`.

## Safety Notes

- Do not claim digest resolution when only local parsing was performed.
