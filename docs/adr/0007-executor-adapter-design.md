# ADR 0007: Executor Adapter Design

## Decision

Executors are adapters behind a small interface.

## Rationale

Execution targets vary widely. A small port keeps pipeline orchestration separate from shell, SSH, Kubernetes, Helm, Argo CD, and webhook details.

