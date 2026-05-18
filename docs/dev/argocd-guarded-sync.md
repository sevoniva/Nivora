# Argo CD Guarded Sync

Phase 2.6 models guarded Argo CD sync requests. Sync is disabled by default and is not production-grade automation.

A sync request requires all of:

- `gitops.sync: true`
- `gitops.allowSync: true`
- CLI/API confirmation, such as `--confirm`
- CLI/API allow flag, such as `--allow-sync`
- no policy denial

```bash
go run ./cmd/nivora gitops deploy --local examples/deployments/argocd-sync-guarded.yaml --confirm --allow-sync
go run ./cmd/nivora argocd sync --app demo-springboot --confirm --allow-sync
```

`prune` defaults to false. `force` is rejected in this phase. Watch behavior is limited to modeled sync and health status from the provider.

Nivora does not replace Argo CD. Production sync automation, credentials, RBAC, SSO, multi-cluster operations, and application management remain future work.
