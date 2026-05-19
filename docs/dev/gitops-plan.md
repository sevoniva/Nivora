# GitOps Planning

Phase 6.1 supports local GitOps planning, local working tree updates, local commit generation, and guarded rollback by revision for Argo CD-style targets.

```bash
go run ./cmd/nivora gitops plan --local examples/deployments/argocd-plan.yaml
```

The command builds a GitOpsChangePlan from the deployment spec. Planning does not contact Git, Gitea, GitHub, GitLab, or Argo CD.

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
- The project remains early-stage and not production-ready.
