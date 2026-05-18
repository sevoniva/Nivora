# Release

A Release is a versioned delivery intent.

## Why It Exists

Releases connect Artifacts to delivery decisions. They provide a stable object for promotion, approval, audit, rollback, and deployment planning.

## Relationships

- Usually belongs to an Application.
- References immutable Artifacts.
- Can produce one or more DeploymentRuns.
- May require approvals or Policy checks.

## Common Confusion

A Release is not just a Git tag and not just a container tag. It should identify exactly what is intended for delivery.

