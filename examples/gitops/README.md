# GitOps Examples

These examples are safe local fixtures for Phase 2.3 GitOps planning.

- `apps/demo-springboot/dev/deployment.yaml` is a local working tree file that can be copied into a temporary directory for `nivora gitops write`.
- Argo CD sync is disabled by default.
- The repository URL in deployment examples is a placeholder.
- No credentials are required or included.

Example:

```bash
mkdir -p tmp/gitops
cp -R examples/gitops/apps tmp/gitops/
go run ./cmd/nivora gitops plan --local examples/deployments/argocd-plan.yaml
go run ./cmd/nivora gitops write --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm
```
