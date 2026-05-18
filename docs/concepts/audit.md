# Audit

Audit is the durable record of important delivery actions.

## Why It Exists

Delivery systems affect production systems and sensitive credentials. Nivora must preserve who did what, when, to which target, with which Artifact, under which Policy and approval context.

## Relationships

- Records important PipelineRun, Release, DeploymentRun, Policy, approval, credential, and runner actions.
- Must not contain secret values.
- Should correlate with events and logs.

## Common Confusion

Audit is not optional logging. Logs explain behavior; audit records preserve accountability.

