# Phase 0: Skeleton

## Objective

Create a complete, clean, compilable backend skeleton.

## Scope

- Go module.
- `nivora-server`, `nivora-worker`, `nivora-runner`, and `nivora` CLI.
- Config loading.
- Logging setup.
- Health, readiness, version, system info, and placeholder HTTP routes.
- Domain structs and statuses.
- Port interfaces.
- Placeholder Adapters.
- API specs.
- Database migrations.
- Local development files.
- CI workflow.
- Minimal tests.

## Non-Goals

- Real CI/CD execution.
- Real cloud, Kubernetes, Argo CD, GitHub, GitLab, Harbor, Nexus, AWS, Aliyun, or Tencent integrations.
- Frontend code.
- Production readiness.

## Expected Deliverables

A repository that compiles, runs minimal services, and gives contributors a clear structure.

## Acceptance Criteria

- `go test ./...` passes.
- All four binaries build.
- Server exposes basic health and version endpoints.
- Placeholder routes return structured not implemented responses.

## Contribution Opportunities

- Documentation corrections.
- Example cleanup.
- Minimal tests.
- Local development polish.

