# Runbook: Failed DeploymentRun

Use this when a DeploymentRun fails during planning, dry-run, apply, verification, GitOps, or host execution.

## Signals

- DeploymentRun status is `Failed`.
- `nivora_runtime_failure_total` increases.
- Deployment timeline includes `devops.deployment.failed`.

## Triage

```sh
nivora deployment get <deployment-run-id>
nivora deployment logs <deployment-run-id>
nivora deployment events <deployment-run-id>
nivora deployment timeline <deployment-run-id>
```

Inspect related surfaces:

```sh
nivora deployment diff <deployment-run-id>
nivora deployment health <deployment-run-id>
nivora deployment rollback-plan <deployment-run-id>
```

## Recovery

1. Identify whether failure happened before mutation. Dry-run failures usually do not mutate targets.
2. For guarded apply failures, inspect resources and health before retry.
3. If rollback is appropriate and supported for the target:

```sh
nivora deployment rollback <deployment-run-id> --confirm
```

4. Preserve logs, events, and audit records for evidence.

## Escalation Notes

Do not run destructive prune/delete operations as part of this runbook. Nivora rollback is guarded and target-specific.
