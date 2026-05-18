# Kubernetes Resource Inventory

Phase 2.4 records desired Kubernetes resource inventory for static YAML DeploymentRuns.

```bash
go run ./cmd/nivora deployment resources --local examples/deployments/yaml-health-dry-run.yaml
go run ./cmd/nivora deployment snapshot --local examples/deployments/yaml-health-dry-run.yaml
```

The local runtime does not require a cluster. It records resource kind, API version, group, version, namespace, name, labels, annotations, source file, document index, and desired hash.

Secret values are redacted. Only Secret metadata is used for inventory.
