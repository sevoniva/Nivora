# Runbook: Policy Gate Denied

Use this when a policy gate denies a release, deployment, or security evaluation.

## Signals

- `nivora_policy_denial_total` increases.
- Events include `devops.policy.gate.denied`.
- DeploymentRun or ReleaseExecution stops with a policy reason.

## Triage

```sh
nivora deployment events <deployment-run-id>
nivora deployment security <deployment-run-id>
```

For security scans:

```sh
nivora security scan artifact <reference> --local
nivora policy evaluate --subject <reference>
```

## Recovery

1. Read the policy result and findings.
2. Fix the artifact, manifest, or deployment input.
3. Re-run planning or dry-run before any apply.
4. If an approval override exists in a future workflow, capture the approval evidence.

## Escalation Notes

Do not bypass policy by editing stored runtime state. Keep denial events and audit records for evidence.
