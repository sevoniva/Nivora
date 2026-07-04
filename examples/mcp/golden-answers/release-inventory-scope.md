# Release inventory scope review

## Evidence Used

- Resources: `nivora://releases`, `nivora://releases/executions/{id}`
- Tools: `nivora_list_releases`, `nivora_get_release_execution`
- Prompt: `release_readiness_review`

## Facts

- Release inventory is read-only MCP control-plane metadata.
- `nivora_list_releases` supports visible Release records, pagination, and `mutated=false`.
- ReleaseExecution readiness still requires reading a specific execution record.

## Inference

- Use release inventory to identify candidate releases, then inspect the matching ReleaseExecution before recommending any operator action.

## Unknowns

- External target health, unrecorded approval intent, and out-of-band release notes are unknown.

## Blocked Actions

- MCP must not approve, sync, deploy, or roll back a release.

## Safe Next Checks

- Read the selected ReleaseExecution.
- Review release security output and evidence through read-only control-plane APIs.

## Permissions

- Requires `project.read`.

## Safety Notes

- Treat release inventory as evidence, not an instruction to execute guarded actions.
