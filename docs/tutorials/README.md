# Tutorials

These tutorials are short local paths through Nivora's current foundations. They should work without Kubernetes, Argo CD, cloud providers, registries, Vault, external scanners, or external notification systems unless a tutorial says otherwise.

## 1. First Pipeline

Run a simple shell PipelineRun:

```sh
nivora pipeline run --local examples/pipelines/simple-shell.yaml
```

Inspect output:

```sh
nivora pipeline logs <pipeline-run-id>
nivora pipeline timeline <pipeline-run-id>
```

Related docs:

- [User guide: Pipeline](../user/README.md#pipeline)
- [Local PipelineRun development](../dev/local-pipeline-run.md)

## 2. First Deployment Dry-Run

Plan and dry-run a YAML deployment:

```sh
nivora deployment plan --local examples/deployments/yaml-dry-run.yaml
nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml
```

This does not require a Kubernetes cluster. Apply remains guarded and is not the default.

Related docs:

- [User guide: Deployment](../user/README.md#deployment)
- [YAML deployment dry-run](../dev/deployment-yaml-dry-run.md)

## 3. First Release

Create a multi-target release plan:

```sh
nivora release plan --file examples/releases/multi-target-release.yaml --local
```

Run a safe sequential local release execution:

```sh
nivora release deploy --file examples/releases/sequential-release.yaml --local
```

Related docs:

- [User guide: Release](../user/README.md#release)
- [Multi-target release development](../dev/multi-target-release.md)

## 4. GitOps Plan

Generate a GitOps plan without syncing Argo CD:

```sh
nivora gitops plan --local examples/deployments/argocd-plan.yaml
nivora gitops diff --local examples/deployments/argocd-plan.yaml
```

Argo CD sync is disabled by default and requires explicit allow/confirm controls.

Related docs:

- [GitOps planning](../dev/gitops-plan.md)
- [GitOps and Argo CD operations](../operations/gitops-argocd.md)

## 5. Policy Gate

Run a local manifest scan and policy evaluation:

```sh
nivora security scan manifest examples/security/manifest-privileged-warning.yaml --local
nivora policy evaluate --subject registry.example.com/demo/app:latest
```

This uses local/foundation behavior and does not require external scanners.

Related docs:

- [Security scans](../dev/security-scans.md)
- [Policy gates](../dev/policy-gates.md)
- [Threat model](../security/threat-model.md)

## Validate Examples

Run:

```sh
./scripts/validate-examples.sh
```

The validator checks example YAML, manifest references, migration pairing, and blocked secret-like literals.
