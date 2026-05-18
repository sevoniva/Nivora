# Pipeline

A Pipeline is a reusable definition for delivery work.

## Why It Exists

Pipeline definitions describe stages, jobs, steps, inputs, and expected execution behavior. They are versioned so PipelineRuns can point to the exact definition they used.

## Relationships

- Belongs to a Project.
- Has PipelineVersions.
- Produces PipelineRuns.
- May produce Artifacts that later become Releases.

## Common Confusion

A Pipeline is not an execution. A PipelineRun is one execution of a Pipeline.

