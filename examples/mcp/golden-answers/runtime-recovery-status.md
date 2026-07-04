# Runtime recovery status review

## Evidence Used

- Resource: `nivora://runtime/recovery`
- Resource: `nivora://system/runtime`
- Tool: `nivora_get_runtime_recovery_status`
- Prompt: `mcp_safe_operation_check`

## Facts

- MCP can read recovery summaries for pipeline, deployment, release, and outbox state.
- The runtime recovery tool is read-only and must return `mutated=false`.

## Inference

- Stale work, pending outbox events, or non-terminal executions should be reconciled through guarded runtime commands outside MCP.

## Unknowns

- MCP may not know whether a worker reconciled the state after the snapshot was read.

## Blocked Actions

- Do not execute rollback or register runners through MCP.

## Safe Next Checks

- Run runtime status or guarded reconcile commands outside MCP.
- Check worker logs and event outbox health.

## Permissions

- Requires `project.read`.

## Safety Notes

- MCP can explain recovery posture but must not mutate runtime state.
