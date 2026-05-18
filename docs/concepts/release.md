# Release

A Release is a versioned delivery intent.

## Why It Exists

Releases connect Artifacts to delivery decisions. They provide a stable object for promotion, approval, audit, rollback, and deployment planning.

## Relationships

- Usually belongs to an Application.
- References immutable Artifacts.
- Can produce one or more DeploymentRuns.
- May require approvals or Policy checks.

## Current Implementation

Phase 2.0 creates a minimal in-memory Release intent from a deployment spec before creating a DeploymentRun. Release persistence, promotion workflows, approval rules, and artifact registry resolution remain future work.

## Common Confusion

A Release is not just a Git tag and not just a container tag. It should identify exactly what is intended for delivery.
