# Rollback Plan

Phase 2.4 created a rollback plan baseline for Kubernetes YAML DeploymentRuns. Phase 6.0 adds a guarded manifest-restore rollback path.

```bash
go run ./cmd/nivora deployment rollback-plan --local examples/deployments/yaml-rollback-plan.yaml
```

Rollback plans include the current manifest snapshot, target, resources, strategy, warnings, and an `executable` flag. For confirmed apply runs, the plan may be executable through the configured manifest client.

Nivora does not perform destructive rollback by default. Phase 6.0 rollback requires explicit confirmation, restores manifests through the configured adapter, and does not prune/delete resources.
