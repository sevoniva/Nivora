# YAML Deployment Dry-Run

Phase 2.0 adds a minimal DeploymentRun foundation for static YAML manifests. It is intentionally non-destructive.

## What It Does

- Parses a `Deployment` spec.
- Creates a minimal Release intent and DeploymentRun.
- Renders static YAML manifest files.
- Validates `apiVersion`, `kind`, and `metadata.name`.
- Builds a DeploymentPlan and resource summary.
- Runs a PolicyEngine pre-check through the local allow-all placeholder.
- Runs non-destructive dry-run validation through the local no-op manifest client.
- Records logs, events, audit records, and timeline entries.

## What It Does Not Do

- It does not apply manifests to Kubernetes by default.
- It does not implement Argo CD.
- It does not implement Helm or Kustomize.
- It does not implement cloud providers or host SSH deployment.
- It does not require kind, Harbor, Nexus, Gitea, GitLab, or Argo CD.
- It is not production-ready.

## Local Commands

```bash
go run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml
go run ./cmd/nivora deployment run --local examples/deployments/yaml-dry-run.yaml
make deployment-plan-local
make deployment-run-local
make verify-deployment
```

## API Shape

With `make run-server` running, post the same deployment definition as JSON:

```bash
curl -X POST http://localhost:8080/api/v1/deployments/plan \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Deployment",
    "metadata": {"name": "demo-yaml-deployment"},
    "spec": {
      "application": "demo-springboot",
      "environment": "dev",
      "target": {"type": "kubernetes-yaml", "name": "dev-kind", "namespace": "default"},
      "manifests": ["examples/yaml/deployment.yaml", "examples/yaml/service.yaml"],
      "options": {"dryRun": true, "apply": false}
    }
  }'
```

The repository provides YAML examples for CLI usage. Convert the YAML spec to JSON before posting directly to the HTTP API.

## Optional Local Kubernetes Validation

Local kind clusters may exist for manual exploration, but they are not required for tests or CI. Do not commit kubeconfigs, credentials, cluster endpoints, or local passwords. Any future kubectl-based dry-run adapter must stay optional and configurable.
