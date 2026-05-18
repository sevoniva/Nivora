# System Context

Nivora sits between users, delivery systems, execution targets, and observability or security tools.

```mermaid
flowchart LR
  Users["Platform engineers\nDevOps engineers\nSREs\nBackend teams\nSecurity reviewers"] --> Nivora["Nivora Control Plane"]
  Contributors["Open-source contributors\nAI coding agents"] --> Repo["sevoniva/nivora repository"]
  Repo --> Nivora
  Git["Git providers"] --> Nivora
  Nivora --> Registry["Artifact registries"]
  Nivora --> Clouds["Cloud providers"]
  Nivora --> K8s["Kubernetes clusters"]
  Nivora --> Argo["Argo CD"]
  Nivora --> Hosts["Host groups"]
  Nivora --> Policy["Policy and security tools"]
  Nivora --> Obs["Observability systems"]
  Nivora --> Runners["Runners"]
```

## Users

Users interact with the Control Plane through APIs and the CLI. Future visualization APIs may support a frontend, but frontend work is not part of current phases.

## External Systems

Nivora should integrate with Git providers, Artifact Registries, cloud providers, Kubernetes clusters, Argo CD, host groups, policy tools, scanners, notification systems, object stores, and observability systems through Ports and Adapters.

## Current State

Phase 0 / Phase 0.6 does not implement real external integrations. It reserves interfaces, package boundaries, API specs, and documentation for future phases.

