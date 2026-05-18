# ADR 0003: Control Plane and Runner Separation

## Decision

The control plane and runner are separate binaries.

## Rationale

Runners execute untrusted or semi-trusted workloads. Keeping them separate limits blast radius and clarifies trust boundaries.

