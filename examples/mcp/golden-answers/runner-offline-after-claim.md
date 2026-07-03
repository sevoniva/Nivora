# Runner offline after job claim

## Evidence Used

- Resources: `nivora://runners/summary`, `nivora://system/runtime`
- Tool: `nivora_get_runner_summary`
- Prompt: `runner_fleet_health_review`

## Facts

- MCP can report runner status counts and runtime warnings visible to the subject.
- MCP cannot operate the runner host.

## Inference

- A stuck claim can be suspected only when stale heartbeat or lease evidence is present.

## Unknowns

- Host OS state, container runtime state, and network reachability are unknown.

## Blocked Actions

- Do not rotate tokens or register runners through MCP.

## Safe Next Checks

- Check runner host logs outside MCP.
- Review token rotation audit through normal APIs.

## Permissions

- Requires `project.read`.

## Safety Notes

- Shell executor is not an OS-level sandbox.
