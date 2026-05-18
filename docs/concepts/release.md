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

Phase 2.2 adds explicit Release creation and ReleaseArtifact binding. The current implementation validates release definitions, parses artifact references, records immutability warnings, resolves digest identity only when already present in the reference, emits release/artifact events, and writes audit records in the in-memory runtime.

Release records are still an early foundation. Durable database persistence, promotion workflows, approval rules, registry-backed digest resolution, and production release governance remain future work.

## Common Confusion

A Release is not just a Git tag and not just a container tag. It should identify exactly what is intended for delivery.

A Release is also not a DeploymentRun. The Release says what should be delivered; the DeploymentRun records one execution of that intent against an Environment or ReleaseTarget.
