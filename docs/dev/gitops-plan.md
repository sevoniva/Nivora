# GitOps Planning

Phase 6.1 supports local GitOps planning, local working tree updates, local commit generation, and guarded rollback by revision for Argo CD-style targets.

```bash
go run ./cmd/nivora gitops plan --local examples/deployments/argocd-plan.yaml
```

The command builds a GitOpsChangePlan from the deployment spec. Planning does not contact Git, Gitea, GitHub, GitLab, or Argo CD.

GitOps plans can either include `target.repoURL` directly or reference an existing repository catalog record with `target.repositoryId`.
When `repositoryId` is used, Nivora resolves the repository URL and default branch from catalog metadata only.
It does not contact the SCM provider, does not clone the repository, and does not resolve or return CredentialRef secret values during planning.

The catalog-backed form requires the server API because local mode does not have the repository catalog.
Create a repository record first, then plan with `--local=false` and a project scope:

```bash
go run ./cmd/nivora repository create \
  --project-id project-placeholder \
  --name gitops-config \
  --url https://example.com/platform/gitops.git \
  --provider generic

go run ./cmd/nivora gitops plan examples/deployments/argocd-plan-catalog.yaml \
  --local=false \
  --project-id project-placeholder
```

To test local working tree writes:

```bash
mkdir -p tmp/gitops
cp -R examples/gitops/apps tmp/gitops/
go run ./cmd/nivora gitops write --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm
```

Local writes are intentionally explicit. There is no remote commit or push in Phase 2.3.

To commit local working tree changes:

```bash
go run ./cmd/nivora gitops commit --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm
```

Remote push is disabled by default. A push requires both `--push` and `--allow-push`, and should only be used with explicitly configured credentials outside the repository.

To plan a guarded rollback by Git revision:

```bash
go run ./cmd/nivora gitops rollback --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --revision <git-revision> --confirm
```

## Current Limits

- No GA production Argo CD automation.
- No remote Git provider integration.
- No remote push by default.
- No Helm or Kustomize engine.
- No credentials in examples.
- The project remains a hardened beta-candidate foundation and is not production-ready.
