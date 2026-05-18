# GitOps Examples

These examples are safe local fixtures for GitOps planning and Phase 2.6 guarded Argo CD sync modeling.

- `apps/demo-springboot/dev/deployment.yaml` is a local working tree file that can be copied into a temporary directory for `nivora gitops write`.
- Argo CD sync is disabled by default.
- Guarded sync requires both `--confirm` and `--allow-sync`.
- The repository URL in deployment examples is a placeholder.
- No credentials are required or included.

Example:

```bash
mkdir -p tmp/gitops
cp -R examples/gitops/apps tmp/gitops/
go run ./cmd/nivora gitops plan --local examples/deployments/argocd-plan.yaml
go run ./cmd/nivora gitops deploy --local examples/deployments/argocd-status-read.yaml
go run ./cmd/nivora gitops write --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm
```

Guarded sync remains explicit:

```bash
go run ./cmd/nivora gitops deploy --local examples/deployments/argocd-sync-guarded.yaml --confirm --allow-sync
```
