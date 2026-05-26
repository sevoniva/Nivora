# Problem Statement

Delivery systems are fragmented because each tool owns a different part of the path from source change to running software.

Git providers know commits and pull requests. CI systems know jobs and logs. Artifact Registries know images and packages. Kubernetes release tools know manifests and rollout state. Argo CD knows GitOps application sync. Host deployment tools know machines and scripts. Cloud providers know regions, clusters, hosts, and registries. Approval systems know human decisions. Policy engines know rules. Audit systems know accountability.

The result is a delivery process where no single system can answer simple questions consistently:

- What Artifact was approved?
- Which PipelineRun produced it?
- Which Release used it?
- Which DeploymentRun changed the environment?
- Which Policy gates passed or failed?
- Who approved it?
- Which Runner executed the work?
- What logs and events explain the outcome?
- What target should be rolled back?

## Fragmentation Areas

- Git providers: webhooks, commits, branches, tags, and commit statuses differ by vendor.
- CI runners: job execution, cancellation, and logs are often CI-specific.
- Artifact Registries: tags are mutable, digests are not always modeled, and metadata is inconsistent.
- Kubernetes release methods: YAML apply, Helm, Kustomize, operators, and custom scripts all expose different state.
- Argo CD and GitOps: GitOps is useful, but it is not the only delivery mode.
- Host deployment: many teams still deploy to host groups, systemd services, or SSH targets.
- Cloud providers: regions, clusters, hosts, and registries vary by provider.
- Approvals: approval gates are often external to the actual deployment record.
- Audit: audit trails are split across Git, CI, chat, deployment tooling, and cloud logs.
- Policy gates: policy checks may run before build, before release, before deploy, or during verification.

## Why a Unified Control Plane Helps

A unified Control Plane can normalize delivery intent and lifecycle state while leaving execution details to specialized tools. Nivora should make the core questions explicit: what is being delivered, where it is going, who approved it, which policies applied, which Runner executed it, and what happened.
