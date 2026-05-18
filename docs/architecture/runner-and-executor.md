# Runner and Executor

Runners belong to the Execution Plane. Executors are the mechanisms a Runner uses to perform work.

## Runner Responsibilities

- Register with the Control Plane.
- Send heartbeats.
- Receive job assignments.
- Prepare execution context.
- Invoke Executors.
- Stream logs.
- Report status.
- Handle cancellation.

## Executor Responsibilities

- Implement a specific execution mechanism.
- Prepare, run, cancel, and expose logs for assigned work.
- Keep integration-specific behavior out of use cases and domain models.

## Runner vs Executor

The Runner is the long-running process. The Executor is a strategy used by the Runner. A single Runner may support multiple Executors over time.

## Controlled Execution

Execution should be conservative by default. Privileged execution, remote access, container isolation, Kubernetes jobs, Argo CD sync, and cloud operations require explicit design before implementation.

## Future Remote Runner Protocol

Future phases should define a protocol for registration, heartbeat, job assignment, log streaming, cancellation, and status reporting. Protocol changes should use an RFC.

## Current State

Phase 0 includes placeholder runner registration and heartbeat logs plus a basic shell Executor for controlled local commands.

