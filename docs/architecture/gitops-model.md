# GitOps Model

GitOps is one deployment mode in Nivora, not the whole product. Phase 2.3 adds a safe foundation for planning GitOps changes and modeling Argo CD status/sync behavior without claiming production automation.

## Current Scope

Phase 2.3 supports:

- `argocd` ReleaseTarget fields in deployment specs
- GitOps change plans
- local working tree reads/writes when explicitly confirmed
- simple image reference update planning
- a noop Argo CD provider for status and guarded sync tests
- logs, events, audit records, and timeline entries for GitOps DeploymentRuns

It does not implement production Argo CD sync, Git provider authentication, remote push, Helm rendering, Kustomize rendering, or multi-cluster GitOps operations.

## Flow

```mermaid
flowchart LR
    Release["Release or DeploymentSpec"] --> Target["GitOpsTarget / ArgoCDTarget"]
    Target --> Plan["GitOpsChangePlan"]
    Plan --> Diff["Local diff"]
    Diff --> Worktree["Optional local working tree update"]
    Plan --> Status["Optional Argo CD status read"]
    Plan --> Sync["Optional guarded sync request"]
    Sync --> Timeline["Logs, events, audit, timeline"]
```

## Safety Defaults

- `sync` defaults to `false`.
- `writeToWorkingTree` defaults to `false`.
- Local writes require CLI confirmation.
- Sync requires explicit confirmation and allow flags.
- Credentials are referenced by name only and are not stored in specs.

Future real adapters must stay behind ports and must not leak Argo CD or Git client types into domain or use case packages.
