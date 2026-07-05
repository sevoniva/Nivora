# Repository DevOps readiness review

## Evidence Used

- Resource: `nivora://capabilities/current`
- Tool: `nivora_repository_inspect`
- Prompt: `repository_devops_readiness_review`

## Facts

- Local repository inspection can capture metadata about files, detected language signals, framework signals, package managers, build/test/package command candidates, deployment hints, and a recommended Nivora Workflow draft.
- The inspection path is static and must return `mutated=false`.
- Detected commands are candidates only; they were not executed by MCP or repository intelligence.

## Inference

- The repository has enough static evidence to review a plan-only workflow draft and begin a controlled CI planning path.
- It is not ready for release or deployment execution until runtime, artifact, policy, approval, and deployment-plan evidence is present.

## Unknowns

- Whether the detected build and test commands pass on a scoped runner is unknown.
- Artifact digest identity, policy results, approval state, runner label fit, and deployment dry-run evidence are unknown.

## Blocked Actions

- Do not apply deployments, push Git changes, retrieve secrets, run scanners, create releases, or execute workflow jobs through MCP.

## Safe Next Checks

- Review the recommended workflow draft.
- Run `nivora_workflow_validate` and `nivora_workflow_plan` as plan-only checks.
- Use guarded workflow run APIs only after runner labels, permissions, and unsafe executor policy are reviewed.

## Permissions

- Requires `project.read` for repository inspection and prompt rendering.

## Safety Notes

- Static repository intelligence is evidence for planning, not proof of production readiness.
- MCP must not expose repository credentials, raw tokens, token hashes, private keys, kubeconfigs, or Authorization headers.
