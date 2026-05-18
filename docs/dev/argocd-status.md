# Argo CD Status

Phase 2.6 adds a read-only Argo CD status foundation. The default local runtime uses a deterministic noop provider, so tests and examples do not require a real Argo CD server.

```bash
go run ./cmd/nivora argocd status --app demo-springboot
go run ./cmd/nivora argocd resources --app demo-springboot
go run ./cmd/nivora gitops deploy --local examples/deployments/argocd-status-read.yaml
```

Status read is intentionally safer than sync. A failed status read can be recorded as a warning for plan-only flows unless the deployment spec requires status.

Optional local Argo CD validation must use environment variables or config for URLs and credentials. Do not commit credentials or local endpoints.
