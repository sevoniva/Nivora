# Release

A Release is a versioned delivery intent.

## Why It Exists

Releases connect Artifacts to delivery decisions. They provide a stable object for promotion, approval, audit, rollback, and deployment planning.

## Relationships

- Usually belongs to an Application.
- References one or more ReleaseArtifacts.
- Can produce one or more DeploymentRuns.
- May require approvals or Policy checks.

## Current Implementation

Phase 2.5 supports explicit Release creation and ReleaseArtifact binding with optional OCI digest resolution. The current implementation validates release definitions, parses artifact references, records immutability warnings, can resolve digest identity through generic OCI registry APIs when configured, emits release/artifact events, and records audit entries through the configured runtime store.

Phase 2.7 adds a ReleasePlan and ReleaseExecution foundation. A Release can now be planned across multiple ReleaseTargets and executed sequentially through target-level DeploymentRuns or safe placeholder targets. The orchestration layer owns aggregate status only; each DeploymentRun still owns target execution details.

For a saved server-side Release, `nivora release plan <release-id> --environment <env> --target <name> --local=false --token-env NIVORA_AUTH_TOKEN` and `nivora release deploy <release-id> --environment <env> --target <name> --local=false --token-env NIVORA_AUTH_TOKEN` create a minimal ReleaseOrchestration request for safe noop/webhook targets. Targets that need a Deployment spec, such as Kubernetes, Argo CD, or host targets, still use an orchestration file and guarded execution flags.

Release evidence can be generated through `POST /api/v1/releases/{id}/evidence` or `nivora release evidence <release-id> --token-env NIVORA_AUTH_TOKEN`. The evidence bundle reuses the compliance evidence store and includes available release, artifact, event, and audit references without secret values.

Release records are still a partial foundation. Promotion workflows, advanced approval rules, full registry management APIs, and production release governance remain future work.

## Common Confusion

A Release is not just a Git tag and not just a container tag. It should identify exactly what is intended for delivery.

A Release is also not a DeploymentRun. The Release says what should be delivered; the DeploymentRun records one execution of that intent against an Environment or ReleaseTarget.
