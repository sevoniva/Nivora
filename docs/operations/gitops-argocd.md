# GitOps and Argo CD Operations

Phase 6.1 hardens the GitOps runtime foundation around local working tree operations, revision tracking, guarded sync, and rollback planning.

Nivora still does not replace Argo CD, and this is not GA production automation.

## Supported in Phase 6.1

- Build a `GitOpsChangePlan` from a deployment spec.
- Update image references in a local GitOps working tree.
- Generate a local diff.
- Create a local Git commit for changed files.
- Keep Git push disabled unless explicitly requested and allowed.
- Read Argo CD application status through the configured provider.
- Request Argo CD sync only when sync, allow-sync, and confirmation are all present.
- Watch limited Argo CD sync and health status.
- Plan and execute a guarded local rollback by Git revision checkout.
- Record logs, events, audit records, and DeploymentRun timeline entries.

## Safety Rules

- `gitops.sync` defaults to `false`.
- `gitops.push` defaults to `false`.
- `gitops.allowPush` defaults to `false`.
- Argo CD sync requires `gitops.sync=true`, `gitops.allowSync=true`, and API/CLI confirmation.
- Git push requires `gitops.push=true`, `gitops.allowPush=true`, and API/CLI confirmation.
- Rollback by revision requires `gitops.rollback=true`, `gitops.rollbackRevision`, a working tree, and confirmation.
- No Git or Argo CD credentials should be stored in specs, examples, or logs.

## Local Commands

```bash
go run ./cmd/nivora gitops plan --local examples/deployments/argocd-plan.yaml
go run ./cmd/nivora gitops write --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm
go run ./cmd/nivora gitops commit --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm
go run ./cmd/nivora gitops rollback --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --revision <git-revision> --confirm
go run ./cmd/nivora argocd status --app demo-springboot
go run ./cmd/nivora argocd sync --app demo-springboot --confirm --allow-sync
```

The local Argo CD commands use the configured provider. The default development provider is deterministic and does not contact a real Argo CD server.

## API Surface

- `POST /api/v1/deployments/gitops/plan`
- `POST /api/v1/deployments/gitops/commit`
- `POST /api/v1/deployments/gitops/rollback`
- `GET /api/v1/integrations/argocd/applications/{name}/status`
- `GET /api/v1/integrations/argocd/applications/{name}/resources`
- `POST /api/v1/integrations/argocd/applications/{name}/sync`
- `POST /api/v1/deployments/{id}/sync`

## Current Limitations

- Production Argo CD application management is not implemented.
- Remote Git provider authentication is future work.
- Push is guarded and not part of default examples.
- Rollback uses Git revision checkout only; it does not force Argo CD sync by default.
- Multi-cluster GitOps, SSO, RBAC integration, and advanced sync options remain future work.
