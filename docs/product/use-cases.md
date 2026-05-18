# Use Cases

These use cases describe intended direction. Phase 0 / Phase 0.6 does not implement real delivery workflows.

## Git Webhook to Pipeline Execution

A Git provider sends a webhook. The Control Plane creates a PipelineRun, schedules jobs to a Runner, collects logs, records status transitions, and emits audit events.

## Local Shell Executor for Early Testing

A Runner executes a controlled shell command through the shell Executor. This is useful for early development and tests, not a full production execution model.

## Host Deployment

A Release is deployed to a host group Release Target through a future host or SSH Executor. The DeploymentRun records status, logs, approval, policy checks, and rollback context.

## Kubernetes YAML Deployment

A DeploymentRun applies Kubernetes YAML to a Kubernetes Release Target through a future YAML apply Executor. Nivora should record intent, target, result, and audit without making Kubernetes the only model.

## Helm and Kustomize Deployment

A DeploymentRun renders and applies Helm or Kustomize output through future Executor Adapters. Rendering, diff, approval, and verification should be explicit.

## Argo CD GitOps Release

A Release updates or syncs a GitOps target through a future Argo CD Adapter. GitOps is one deployment mode, not the entire product.

## Artifact-Based Release

A Release references immutable Artifact digests from an Artifact Registry. Promotion should not depend on mutable tags or rebuilding during deployment.

## Multi-Cloud Target Inventory

CloudProvider Adapters can list regions, clusters, hosts, and registries. Nivora should use that inventory to map Release Targets without becoming cloud-provider-specific.

## Approval and Policy Gate

Policy checks and approvals can pause or reject a PipelineRun or DeploymentRun. Policy is a gate, not just documentation.

## Release Audit

Nivora should connect who triggered a release, what Artifact was used, what policies ran, who approved it, where it deployed, which Runner executed it, and what happened.

## Rollback

Rollback should be modeled as an auditable operation with reason, target, prior version or Artifact, status, logs, and verification.

## Future Visualization APIs

Future visualization APIs should expose pipeline DAGs, deployment timelines, environment topology, runner status, audit timelines, and security findings. Frontend work is not part of current phases.

