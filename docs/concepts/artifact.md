# Artifact

An Artifact is a build output or package reference intended to be immutable. In Phase 2.5, Nivora can parse and inspect artifact references, optionally resolve OCI image digests through generic registry APIs, bind them to a Release, and carry artifact warnings into deployment planning.

## Why It Exists

Release audit depends on knowing exactly what was delivered. Digests, signed artifacts, and immutable versions are preferred over mutable tags.

An image tag such as `app:1.0.0` is useful, but it can still be moved by a registry operator. A digest reference such as `app@sha256:...` is stronger because it identifies content. Nivora therefore treats digest-backed references as immutable and emits warnings for `latest` or references that do not include a tag or digest.

## Relationships

- May be produced by a PipelineRun.
- Stored in an Artifact Registry.
- Referenced by a Release.
- Bound to a ReleaseArtifact for DeploymentRun planning.
- Verified against manifest image references during YAML deployment planning.
- Evaluated by security scanners or Policy gates in future phases.

## Current Implementation

Phase 2.5 supports a small artifact model, OCI image reference parsing, generic URI-style references, immutability warnings, and generic OCI digest resolution. Harbor is treated as an OCI-compatible registry endpoint in this phase.

The current backend also has a foundation artifact registry catalog at `/api/v1/artifact-registries` and `nivora artifact registry`. It records registry metadata, explicit `insecure` settings for local HTTP registries, capabilities, and `CredentialRef` values. It never stores or returns registry passwords or tokens through registry records.

Artifacts bound to releases are queryable through `/api/v1/artifacts`, `/api/v1/artifacts/{id}`, `/api/v1/artifacts/{id}/releases`, and `nivora artifact list/get/releases`. This inventory is derived from ReleaseArtifact bindings that Nivora already knows about. It is not a full registry crawl.

Nivora does not implement Harbor management APIs, Nexus, JFrog, ECR, ACR, TCR, signing, scanning, or full registry administration.

## Common Confusion

An Artifact is not just a tag. A tag can move. A digest identifies content.

Artifact inspection is not artifact scanning. Trivy, Cosign, SBOM handling, and registry-specific metadata are future phases.
