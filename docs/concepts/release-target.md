# Release Target

A Release Target is the concrete destination for a DeploymentRun.

## Why It Exists

Environments describe delivery context. Release Targets describe where work would happen inside that context, such as a host group, Kubernetes namespace, Argo CD application, cloud target, or webhook target.

## Current Implementation

Phase 2.1 supports the `kubernetes-yaml` target type for static manifest dry-run and explicit no-op local apply. The target may include a Kubernetes context name, but no context is hardcoded and no kubeconfig is stored.

Phase 2.4 adds resource inventory and health output for Kubernetes YAML targets. Cluster access remains optional; the default local runtime evaluates desired resources without requiring a kubeconfig.

Phase 2.7 uses ReleaseTargets in ReleasePlan and ReleaseExecution records. The catalog and orchestration foundation recognize `kubernetes-yaml`, `argocd`, `host`, `noop`, and `webhook` targets, while real execution remains constrained by each target's guarded runtime.

Release orchestration can now reference an existing catalog target by `targetId`. The server resolves the saved target metadata, verifies it is enabled, and preserves project/environment scope in the ReleasePlan. Metadata-only catalog targets can plan safe `noop` and `webhook` targets directly. Executable target types such as `kubernetes-yaml`, `argocd`, and `host` still require an inline Deployment spec; catalog metadata alone is not enough to run them.

The current catalog foundation also exposes ReleaseTarget metadata through:

- `GET /api/v1/release-targets`
- `POST /api/v1/release-targets`
- `GET /api/v1/release-targets/{id}`
- `PATCH /api/v1/release-targets/{id}`
- `DELETE /api/v1/release-targets/{id}`
- `POST /api/v1/release-targets/{id}/validate`

The matching CLI entry is `nivora target`. Target records include `projectId`, `environmentId`, `targetType`, optional `configRef` and `credentialRef`, labels, metadata, and explicit safety flags. `allowApply`, `allowSync`, and `allowRemoteHostDeploy` default to `false`. Deleting a target disables it instead of hard-deleting it, and disabled targets fail validation.

Phase 3.5 adds the `host` target foundation for VM and bare-metal delivery. It supports planning and noop/local execution only by default; real remote SSH deployment remains future work.

## Common Confusion

A Release Target is not always a Kubernetes cluster. GitOps, host, cloud, and webhook targets are different delivery modes with different adapters and safety requirements.

The catalog is still foundation-level. When the server is configured with the PostgreSQL runtime store, ReleaseTarget metadata can be persisted with the rest of the catalog. Full target lifecycle policy, environment promotion rules, and production-grade external target execution remain future work.
