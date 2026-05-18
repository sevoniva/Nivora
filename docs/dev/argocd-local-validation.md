# Argo CD Local Validation

Phase 2.6 includes a noop Argo CD provider for tests and local modeling. It does not require a local Argo CD instance.

Optional manual validation may use environment variables:

```bash
export NIVORA_LOCAL_ARGOCD_URL='<url>'
export NIVORA_LOCAL_ARGOCD_USERNAME='<username>'
export NIVORA_LOCAL_ARGOCD_PASSWORD='<password>'
```

Do not commit actual values.

Current CLI behavior is local/noop:

```bash
go run ./cmd/nivora argocd status --app demo-springboot
go run ./cmd/nivora argocd resources --app demo-springboot
```

Sync is guarded and disabled by default:

```bash
go run ./cmd/nivora argocd sync --app demo-springboot --confirm --allow-sync
```

The Phase 2.6 noop provider records the request but does not call a remote Argo CD API. Production sync, credentials management, RBAC, multi-cluster handling, and application history remain future work.
