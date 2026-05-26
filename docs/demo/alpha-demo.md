# Alpha Demo Guide

This guide demonstrates the `v0.1.0-alpha.1` foundation without requiring Kubernetes, Argo CD, cloud accounts, registries, or external scanners.

## What This Demo Shows

- Project verification.
- Server, worker, and runner startup commands.
- Local PipelineRun execution.
- API smoke path for logs and timeline.
- Release planning and local release execution.
- Deployment YAML dry-run and resource inventory.
- Policy gate example.
- GitOps plan example.

This is an alpha demo, not production-ready validation.

## 1. Verify The Repository

```sh
make verify
```

This runs formatting, tidy check, vet, tests, binary builds, architecture guardrails, secret scan, local runtime smoke paths, host/deployment/release/security checks, and web build.

## 2. Start Local Processes

Use separate terminals if you want to inspect process logs:

```sh
make run-server
make run-worker
make run-runner
```

This historical alpha demo still uses local/in-memory defaults for many workflows. Cross-process durability is a foundation, not a production scheduler.

## 3. Run A Simple Pipeline

```sh
make pipeline-run-local
```

Expected result: a `Succeeded` PipelineRun with stdout logs from `examples/pipelines/simple-shell.yaml`.

## 4. Exercise The API Smoke Path

```sh
./scripts/smoke-api.sh
```

The script starts a temporary server, creates a PipelineRun, checks logs and timeline, creates a DeploymentRun dry-run, and checks plan/resources/logs/timeline.

## 5. Create A Release Plan

```sh
make release-plan-local
```

This uses `examples/releases/multi-target-release.yaml` and produces a ReleasePlan with target-level DeploymentPlans.

## 6. Run A Local Release Execution

```sh
make release-deploy-local
```

Expected result: a local sequential ReleaseExecution using safe/noop foundations.

## 7. Run Deployment YAML Dry-Run

```sh
make deployment-dry-run-local
```

Expected result: a `Succeeded` dry-run DeploymentRun with manifest render/validation logs and resource inventory.

## 8. Inspect Resource Inventory And Diff

```sh
go run ./cmd/nivora deployment plan --local examples/deployments/yaml-health-dry-run.yaml
```

The output includes manifest count, target information, artifact warnings, and a diff summary. Live cluster diff is not required for this demo.

## 9. Run Policy Gate Example

```sh
make security-scan-local
make policy-evaluate-local
```

The manifest example produces warnings from the local security foundation. No external scanner is required.

## 10. Run GitOps Plan Example

```sh
make gitops-plan-local
make gitops-diff-local
```

These commands show GitOps planning and diff behavior without requiring Gitea, GitHub, GitLab, or Argo CD.

## Optional Packaging Checks

```sh
make helm-template
make helm-lint
make docker-build
```

Docker builds require external base image registries to be reachable. Helm checks require Helm to be installed.

## Demo Limitations

- No production Kubernetes apply is demonstrated.
- No destructive rollback is demonstrated.
- No real cloud, Git provider, registry, external scanner, notification, SSO, Vault, or ITSM integration is required.
- The web UI is a minimal foundation, not a complete product UI.
