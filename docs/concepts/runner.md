# Runner

A Runner is an Execution Plane process that receives assigned work and executes it through Executors.

## Why It Exists

Runners keep execution separate from the Control Plane. This separation is important for security, scaling, cancellation, log streaming, and future remote execution protocols.

## Responsibilities

- Register with the Control Plane.
- Send heartbeats.
- Receive assigned jobs.
- Invoke Executors.
- Stream logs.
- Report status.
- Handle cancellation.

## Common Confusion

A Runner is not an Executor. The Runner is the process that manages job execution. The Executor is the mechanism used to perform a specific kind of work.

## Phase 1 Behavior

Phase 1 uses a local in-process Runner for minimal execution. This keeps the Runner and Executor concepts separate while avoiding a remote runner protocol before it is designed.

## Phase 1.5 Behavior

Phase 1.5 adds minimal runner records with ID, name, status, labels, supported executors, and last heartbeat time. The runtime can register a runner, update heartbeat time, list runners, and select an online runner that supports the shell executor.

This is not a production remote runner protocol. Network registration, job polling, streaming, authentication, and runner isolation remain future work.
