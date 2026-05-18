# Artifact

An Artifact is a build output or package reference intended to be immutable.

## Why It Exists

Release audit depends on knowing exactly what was delivered. Digests, signed artifacts, and immutable versions are preferred over mutable tags.

## Relationships

- May be produced by a PipelineRun.
- Stored in an Artifact Registry.
- Referenced by a Release.
- Evaluated by security scanners or Policy gates in future phases.

## Common Confusion

An Artifact is not just a tag. A tag can move. A digest identifies content.

