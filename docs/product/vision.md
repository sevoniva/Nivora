# Product Vision

Nivora's long-term vision is to be an open-source DevOps delivery control plane that gives teams one auditable model for pipelines, releases, deployments, artifacts, policies, runners, and delivery targets.

Nivora is a delivery control plane rather than another CI tool. A CI tool usually focuses on turning source changes into jobs and build outputs. Nivora's intended scope is broader: it should coordinate how source changes become immutable Artifacts, how those Artifacts become Releases, how Releases become DeploymentRuns, and how approvals, policies, logs, events, and audit records follow the delivery lifecycle.

## Why a Control Plane

Delivery crosses many systems:

- Git providers trigger change.
- CI runners execute jobs.
- Artifact Registries hold build outputs.
- Policy tools evaluate risk.
- Approval systems gate sensitive environments.
- Deployment tools change hosts, clusters, or GitOps state.
- Observability systems explain what happened.
- Audit records preserve accountability.

Nivora should provide a consistent control surface across those systems. It should not hide them completely. Operators still need to know whether a deployment used a host group, Kubernetes YAML, Helm, Kustomize, Argo CD, a cloud target, or a webhook target.

## Integrate Rather Than Replace

Nivora should integrate with mature tools through Ports and Adapters. The core should define stable concepts and workflows. Adapters should implement concrete behavior for Git providers, Artifact Registries, cloud providers, Executors, secret stores, event buses, object stores, notification systems, and policy engines.

In the current Phase 0 / Phase 0.5 / Phase 0.6 state, these integrations are not implemented. The repository only defines boundaries, placeholder packages, and planning documents.

