# Target Architecture

This document describes the target architecture. The current Phase 0 / Phase 0.6 implementation only contains the skeleton, package boundaries, minimal HTTP routes, placeholder Adapters, and documentation.

## Target Shape

Nivora starts as a modular monolith with separate binaries:

- `nivora-server`: API and Control Plane entry point.
- `nivora-worker`: background workflow and event processing.
- `nivora-runner`: Execution Plane process.
- `nivora`: CLI.

Service extraction is a future option only after stable module boundaries exist.

## Control Plane and Execution Plane

```mermaid
flowchart TB
  subgraph CP["Control Plane"]
    API["API Server"]
    Worker["Worker"]
    Workflow["Workflow Orchestrator"]
    Integrations["Integration Manager"]
    Policy["Policy Engine"]
    Audit["Audit Service"]
    DB[("PostgreSQL")]
  end

  subgraph EP["Execution Plane"]
    Runner["Runner"]
    Executors["Executors"]
    Host["Host targets"]
    K8s["Kubernetes targets"]
    GitOps["GitOps targets"]
    Cloud["Cloud targets"]
  end

  subgraph EXT["External Systems"]
    Git["Git providers"]
    Registry["Artifact registries"]
    Clouds["Cloud providers"]
    ObjectStore["Object storage"]
    Obs["Observability systems"]
    Scanners["Security scanners"]
  end

  API --> Workflow
  Worker --> Workflow
  Workflow --> Policy
  Workflow --> Audit
  Workflow --> DB
  Integrations --> Git
  Integrations --> Registry
  Integrations --> Clouds
  Runner --> Executors
  Executors --> Host
  Executors --> K8s
  Executors --> GitOps
  Executors --> Cloud
  Runner --> API
  Audit --> DB
  Workflow --> ObjectStore
  Worker --> Obs
  Policy --> Scanners
```

## Ports and Adapters

Ports define capabilities such as SCMProvider, ArtifactProvider, CloudProvider, Executor, WorkflowRuntime, SecretProvider, NotificationProvider, PolicyEngine, EventBus, and ObjectStore. Adapters implement those capabilities.

The domain layer must remain independent from transport, persistence, queue, cloud, Kubernetes, Argo CD, and vendor SDKs.

## Event-Driven Direction

Nivora should move toward durable events for PipelineRun, DeploymentRun, Runner, Artifact, Policy, and Audit activity. Phase 0 only includes an in-memory EventBus Adapter and AsyncAPI skeleton.

## Audit and Policy

Audit and Policy are not add-ons. They are part of the delivery lifecycle and must be considered in workflow, API, storage, and runner design.

