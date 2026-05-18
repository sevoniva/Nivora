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

## Phase 0.5

Guardrails and validation only.

Allowed:

- AI and contributor guardrail cleanup
- architecture verification scripts
- secret verification scripts
- CI check hardening
- Makefile and local development polish
- documentation corrections
- structured placeholder API responses

Not allowed:

- real CI/CD execution
- real cloud integrations
- real Kubernetes or Argo CD integrations
- production workflow runtime
- frontend

## Phase 0.6

Public planning and collaboration docs only.

Allowed:

- project charter
- product vision and problem statement
- concept docs
- target architecture docs
- detailed roadmap docs
- community and governance docs
- RFC process and template
- documentation style guide

Not allowed:

- Phase 1 PipelineRun execution
- real cloud integrations
- real Kubernetes or Argo CD integrations
- real Git provider or Artifact Registry integrations
- frontend

## Phase 1

Minimal pipeline execution.

Target flow:

```text
Git webhook -> PipelineRun -> Runner -> Executor -> Logs -> Status -> Audit
```

## Phase 2

Release and deployment foundation. Phase 2.0 is limited to YAML deployment planning and non-destructive dry-run validation.

Target capabilities:

- YAML DeploymentRun planning/dry-run foundation
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
