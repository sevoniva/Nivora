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
