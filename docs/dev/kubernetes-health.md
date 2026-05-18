# Kubernetes Health

Phase 2.4 adds lightweight health evaluation for Kubernetes YAML DeploymentRuns.

```bash
go run ./cmd/nivora deployment health --local examples/deployments/yaml-health-dry-run.yaml
```

The default local runtime does not query Kubernetes. It evaluates desired resources conservatively:

- workloads such as Deployment, StatefulSet, DaemonSet, Job, and CronJob start as `Progressing`
- Service, ConfigMap, Secret metadata, Ingress, Namespace, ServiceAccount, Role, and bindings are `Healthy`
- unknown resources are `Unsupported`

Live condition-based health is future adapter work. Kubernetes clusters are optional and are not required for tests or CI.
