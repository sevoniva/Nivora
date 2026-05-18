# Rollback Plan

Phase 2.4 creates a rollback plan baseline for Kubernetes YAML DeploymentRuns.

```bash
go run ./cmd/nivora deployment rollback-plan --local examples/deployments/yaml-rollback-plan.yaml
```

Rollback plans include the current manifest snapshot, target, resources, strategy, warnings, and an `executable` flag. The default value is `false`.

Nivora does not perform destructive rollback in this phase. It does not delete resources, patch live objects, or apply previous manifests by default. Rollback execution requires future design, explicit confirmation, and adapter hardening.
