# Delivery Engine Production Boundary

Current maturity: **guarded beta/foundation surfaces, not GA production CD**.

Nivora intentionally separates planning, dry-run, guarded execution, and rollback planning. The delivery engines are not a Kubernetes operator, not a full Argo CD replacement, and not a host deployment automation platform by default.

## Boundary Matrix

| Surface | Current State | Default Behavior | Required Guard | Evidence | Production Boundary |
|---|---|---|---|---|---|
| Kubernetes dry-run | beta | allowed as plan/dry-run | explicit target/spec | deployment CLI/API tests | Does not require a live cluster in CI |
| Kubernetes apply | foundation | disabled | `allowKubernetesApply` plus confirmation | guarded apply tests and docs | Not default; no prune/delete by default |
| Kubernetes prune/delete | blocked by default | not exposed as normal path | separate future design | MCP denied tools and deployment safety policy | Do not implement as implicit rollback |
| Kubernetes rollback plan | beta | plan only | none for read/plan | rollback plan tests | Execution remains guarded |
| Kubernetes rollback execution | foundation | disabled | confirmation and safe restore strategy | guarded rollback tests | No default destructive delete |
| GitOps plan/diff | foundation | local plan/diff | explicit local working tree | GitOps tests/docs | No production Git provider write integration |
| GitOps push | blocked by default | disabled | explicit future write config | guarded GitOps docs | No automatic push from MCP |
| Argo CD status | foundation | read-only | credentials/config if real adapter | fake/noop tests | No app lifecycle management |
| Argo CD sync | foundation | disabled | `sync=true`, `allowSync=true`, confirmation | sync guard tests | No production Argo automation claim |
| Host deployment plan | foundation | plan/dry-run/noop | explicit host target | host tests/docs | Remote execution disabled |
| Remote host deploy | experimental | disabled | `allowRemoteHostDeploy`, confirmation, CredentialRef | host safety tests/docs | SSH is not a sandbox |
| OCI artifact inspect | beta | local parse/metadata | none for local inspect | artifact parser tests | Network resolution optional |
| OCI digest resolution | foundation | optional | registry config and credentials | fake registry tests | No full Harbor admin API |

## Enterprise Rules

- Apply, sync, rollback execution, host deploy, Git push, Kubernetes prune, Kubernetes delete, approval decisions, secret retrieval, and token rotation must not be exposed through MCP action tools.
- Tests must not require Kubernetes, Argo CD, Harbor, cloud providers, or real SSH.
- Examples must be safe, credential-free, and explicit about dry-run/noop behavior.
- Production install profiles must keep unsafe delivery switches disabled unless the operator explicitly enables them.

## Required Next Hardening

| Area | Next Work |
|---|---|
| Kubernetes | Namespace/context restrictions, fake rollout watch matrix, optional live smoke |
| GitOps | Local repo revision rollback proof, guarded push design |
| Argo CD | Read-only status contract and sync-watch fake adapter coverage |
| Host | Fake SSH batch/canary/rollback tests, runner isolation guidance |
| Artifact | Registry credential integration proof with SecretProvider and redaction tests |
