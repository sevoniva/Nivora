# GitOps and Argo CD Guardrails

Nivora GitOps and Argo CD paths are guarded foundations. Sync, write, commit, push, prune, and force all require explicit opt-in. This is not production Argo automation.

## Guarded Sync Requirements

Argo CD sync requires ALL of these gates to pass:

| Gate | Default | How to Enable |
|---|---|---|
| `GitOps.Sync` | false | Set `sync: true` in deployment spec |
| `GitOps.AllowSync` | false | Set `allowSync: true` in deployment spec |
| Confirmed | false | Set `confirmed: true` in API/CLI request |
| RBAC permission | required | User must have `deployment.create` |
| Production config | blocked | `runtime.allow_argo_sync` must be true |
| Provider gate | provider-specific | `NoopProvider.AllowSync` must be true |

If any gate fails, sync is skipped with an event (`devops.argocd.sync.skipped`) and a warning in the plan.

## Working Tree Safety

| Gate | Default | Description |
|---|---|---|
| `WriteToWorkingTree` | false | Manifest files are NOT written to disk by default |
| Working tree path | must be explicit | Path traversal rejected by `safePath` |
| `Commit` | false | No git commit unless explicitly enabled |
| `Push` | false | No git push unless explicitly enabled |
| `AllowPush` | false | Push requires explicit `allowPush` flag |
| `Prune` | false | No resource pruning by default |
| `Force` | false | No force push/sync by default |
| `Rollback` | false | No rollback without explicit request |

## Route Security

All GitOps/Argo mutation routes require `deployment.create` permission. Runner tokens, viewers, and auditors cannot access sync routes (verified in RBAC test suite with 100+ sub-tests).

## Current Limitations

- Argo CD provider is a NoopProvider — no real Argo CD API calls.
- Git provider integration is skeleton/generic — no real push.
- This is NOT production Argo automation. It is a guarded foundation.
