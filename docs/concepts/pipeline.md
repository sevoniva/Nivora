# Pipeline

A Pipeline is a reusable definition for delivery work.

## Why It Exists

Pipeline definitions describe stages, jobs, steps, inputs, and expected execution behavior. They are versioned so PipelineRuns can point to the exact definition they used.

Nivora exposes a foundation pipeline definition catalog through `/api/v1/pipelines` and `nivora pipeline definition`. The catalog validates the existing Pipeline DSL, stores the latest definition with a version number and `definitionHash`, and supports disabling a definition without hard-deleting it.

## Relationships

- Belongs to a Project.
- Has PipelineVersions.
- Produces PipelineRuns.
- May produce Artifacts that later become Releases.

## Common Confusion

A Pipeline is not an execution. A PipelineRun is one execution of a Pipeline.

## Current Limits

- The default definition catalog is in memory.
- PostgreSQL persistence for pipeline definitions is future work.
- Creating a Pipeline definition does not execute it.
- PipelineRun can still be created directly from an inline definition.
- Nivora is not production-ready.
