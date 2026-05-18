# Release Scope

## Phase 0

Skeleton only.

Allowed:

- binaries
- configuration
- health endpoints
- route placeholders
- domain models
- port interfaces
- placeholder adapters
- docs
- migrations
- local development setup
- minimal tests

Not allowed:

- real cloud integrations
- real Kubernetes integrations
- real Argo CD sync
- production-grade workflow runtime
- frontend

## Phase 1

Minimal pipeline execution.

Target flow:

```text
Git webhook -> PipelineRun -> Runner -> Executor -> Logs -> Status -> Audit
```

## Phase 2

GitOps and production release basics.

Target capabilities:

- Argo CD adapter
- Helm/Kustomize/YAML renderer
- approval
- environment lock
- rollback
- deployment diff

## Phase 3

Multi-cloud and DevSecOps.

Target capabilities:

- AWS
- Aliyun
- Tencent Cloud
- Trivy
- Cosign
- SBOM
- policy gates
- OIDC

## Phase 4

Visualization frontend.

Target capabilities:

- pipeline DAG
- deployment timeline
- environment topology
- audit timeline
