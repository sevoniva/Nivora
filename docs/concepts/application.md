# Application

An Application represents a deliverable product or system owned by a Project. It may contain one or more Services.

## Why It Exists

Applications give Nivora a stable unit for Releases and deployment history. They help answer what is being delivered across PipelineRuns, Artifacts, and DeploymentRuns.

## Relationships

- Belongs to a Project.
- May contain Services.
- May have Repositories, Pipelines, Releases, and DeploymentRuns.

## Common Confusion

An Application is not necessarily one repository or one Kubernetes workload. It is a delivery-level concept.

