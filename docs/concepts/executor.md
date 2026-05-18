# Executor

An Executor implements a specific execution mechanism.

## Why It Exists

Different delivery steps need different execution mechanisms. Shell, SSH, Kubernetes Job, YAML apply, Helm, Argo CD, and webhook execution should not be hard-coded into core use cases.

## Relationships

- Used by a Runner.
- Implements the Executor Port.
- Should keep integration-specific behavior inside an Adapter package.

## Common Confusion

An Executor is not a full workflow engine. It runs assigned work through a specific mechanism.

## Phase 1 Behavior

Phase 1 supports the shell Executor for safe test commands such as `echo` and `printf`. Privileged execution and remote execution are future work.

## Phase 1.5 Behavior

The shell Executor returns stdout, stderr, exit code, and errors, and it respects command timeouts through Go context cancellation. Phase 1.5 uses those results to create ordered LogChunks and update StepRun, JobRun, and PipelineRun status.

The shell Executor remains intentionally conservative. Secret injection, privileged execution, remote execution, container isolation, and deployment-specific executors are future work.
