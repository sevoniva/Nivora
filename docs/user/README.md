# User Guide

This guide is for people using Nivora's current backend and CLI foundations. Nivora is a hardened beta-candidate foundation and is not production-ready. The examples here are local, guarded, and self-contained unless marked otherwise.

## Core Workflows

| Workflow | Start here | Example |
| --- | --- | --- |
| Run a shell pipeline | [Pipeline](#pipeline) | `examples/pipelines/simple-shell.yaml` |
| Plan a deployment dry-run | [Deployment](#deployment) | `examples/deployments/yaml-dry-run.yaml` |
| Create or inspect a release | [Release](#release) | `examples/releases/simple-release.yaml` |
| Inspect an artifact reference | [Artifact](#artifact) | `examples/artifacts/oci-image.yaml` |
| Request or decide approval | [Approval](#approval) | `examples/approvals/deployment-approval.yaml` |
| Evaluate a policy gate | [Security](#security) | `examples/security/policy-latest-warn.yaml` |

## Pipeline

Current PipelineRun support is shell-focused and local-friendly.

```sh
nivora pipeline run --local examples/pipelines/simple-shell.yaml
nivora pipeline logs <pipeline-run-id>
nivora pipeline timeline <pipeline-run-id>
```

Useful docs:

- [PipelineRun concept](../concepts/pipeline-run.md)
- [Local PipelineRun development](../dev/local-pipeline-run.md)
- [Runtime acceptance matrix](../dev/runtime-acceptance.md)

## Deployment

Current DeploymentRun support focuses on YAML planning, dry-run, explicit guarded apply foundations, resource inventory, health summaries, snapshots, diffs, and rollback plans.

```sh
nivora deployment plan --local examples/deployments/yaml-dry-run.yaml
nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml
```

Apply is never the default. Guarded apply examples require explicit flags and should be used only in disposable environments.

Useful docs:

- [DeploymentRun concept](../concepts/deployment-run.md)
- [YAML deployment dry-run](../dev/deployment-yaml-dry-run.md)
- [Kubernetes deployment operations](../operations/kubernetes-deployment.md)

## Release

Release represents versioned delivery intent. ReleaseExecution coordinates one or more target DeploymentRuns.

```sh
nivora release plan --file examples/releases/multi-target-release.yaml --local
nivora release deploy --file examples/releases/sequential-release.yaml --local
```

For a Release already saved on a Nivora server, safe noop/webhook planning can be started from the Release ID:

```sh
nivora release plan <release-id> --environment dev --target audit-only --local=false
nivora release deploy <release-id> --environment dev --target audit-only --local=false
```

Release ID mode is server-backed. Targets that need a full Deployment spec, such as Kubernetes, Argo CD, or host targets, still use `--file` and remain guarded.

Useful docs:

- [Release concept](../concepts/release.md)
- [Release orchestration architecture](../architecture/release-orchestration.md)
- [Multi-target release development](../dev/multi-target-release.md)

## Artifact

Artifact references should be immutable where possible. Digest references are preferred over mutable tags.

```sh
nivora artifact inspect registry.example.com/team/demo:1.0.0
nivora artifact resolve registry.example.com/team/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
```

Useful docs:

- [Artifact concept](../concepts/artifact.md)
- [Artifact model](../architecture/artifact-model.md)
- [Artifact registry operations](../operations/artifact-registries.md)

## Approval

Approval, change window, and notification foundations model human governance. They do not implement a full ITSM workflow.

```sh
nivora approvals list
nivora change-window evaluate --env prod
```

Useful docs:

- [Approval concept](../concepts/approval.md)
- [Change window concept](../concepts/change-window.md)
- [Change management operations](../operations/change-management.md)

## Security

Security gates can scan, evaluate policy, and produce allow/warn/deny/approval-required decisions through current foundations.

```sh
nivora security scan manifest examples/security/manifest-privileged-warning.yaml --local
nivora policy evaluate --subject registry.example.com/demo/app:latest
```

Useful docs:

- [Policy concept](../concepts/policy.md)
- [Security scans](../dev/security-scans.md)
- [Security threat model](../security/threat-model.md)
