# GitOps Planning

Phase 2.3 adds local GitOps planning for Argo CD-style targets.

```bash
go run ./cmd/nivora gitops plan --local examples/deployments/argocd-plan.yaml
```

The command builds a GitOpsChangePlan from the deployment spec. It does not contact Git, Gitea, GitHub, GitLab, or Argo CD.

To test local working tree writes:

```bash
mkdir -p tmp/gitops
cp -R examples/gitops/apps tmp/gitops/
go run ./cmd/nivora gitops write --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm
```

Local writes are intentionally explicit. There is no remote commit or push in Phase 2.3.

## Current Limits

- No production Argo CD sync.
- No remote Git provider integration.
- No Helm or Kustomize engine.
- No credentials in examples.
- The project remains early-stage and not production-ready.
