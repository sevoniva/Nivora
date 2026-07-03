# Artifact missing digest review

## Evidence Used

- Resource: `nivora://capabilities/current`
- Tool: `nivora_inspect_artifact`
- Prompt: `policy_gate_review`

## Facts

- The artifact reference can be parsed locally.
- A tag-only reference is not digest pinned.

## Inference

- Missing digest reduces release immutability confidence.

## Unknowns

- Actual registry digest, media type, and artifact size are unknown.

## Blocked Actions

- Do not retrieve registry credentials through MCP.

## Safe Next Checks

- Resolve the digest through normal registry-integrated artifact APIs.

## Permissions

- Requires `deployment.create`.

## Safety Notes

- Local inspection is not a network registry verification.
