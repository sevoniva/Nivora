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

Phase 2.5 supports explicit Release creation and ReleaseArtifact binding with optional OCI digest resolution. The current implementation validates release definitions, parses artifact references, records immutability warnings, can resolve digest identity through generic OCI registry APIs when configured, emits release/artifact events, and writes audit records in the in-memory runtime.

Phase 2.7 adds a ReleasePlan and ReleaseExecution foundation. A Release can now be planned across multiple ReleaseTargets and executed sequentially through target-level DeploymentRuns or safe placeholder targets. The orchestration layer owns aggregate status only; each DeploymentRun still owns target execution details.

Release records are still an early foundation. Durable database persistence, promotion workflows, approval rules, advanced credential handling, registry management APIs, and production release governance remain future work.

## Common Confusion

A Release is not just a Git tag and not just a container tag. It should identify exactly what is intended for delivery.

A Release is also not a DeploymentRun. The Release says what should be delivered; the DeploymentRun records one execution of that intent against an Environment or ReleaseTarget.
