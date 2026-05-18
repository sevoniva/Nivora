# Kubernetes Resource Model

Phase 2.4 improves Kubernetes YAML DeploymentRun observability without turning Nivora into a production Kubernetes controller.

## Current Scope

Nivora renders static YAML manifests, extracts a desired resource inventory, creates a manifest snapshot, builds a lightweight health evaluation, generates a desired-state diff summary, and creates a non-destructive rollback plan.

Supported inventory kinds include common workloads and platform resources such as Deployment, StatefulSet, DaemonSet, Job, CronJob, Service, ConfigMap, Secret metadata, Ingress, Namespace, ServiceAccount, Role, RoleBinding, ClusterRole, and ClusterRoleBinding.

Secret values are not stored or logged. Secret manifest content is redacted before it is kept in the in-memory manifest snapshot.

## Health

Health is lightweight in Phase 2.4. The default local runtime does not query a Kubernetes cluster, so workload resources are usually `Progressing`, simple metadata resources are `Healthy`, and unsupported resources are `Unsupported`.

Future Kubernetes adapters may observe live status, UID, resourceVersion, conditions, and rollout details through a safe adapter boundary.

## Rollback

Rollback is a plan, not an action. Phase 2.4 creates a `manifest-restore` rollback plan with `executable: false`. It does not delete, patch, or apply cluster resources.
