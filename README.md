# Nivora

> Backend-first delivery control plane for pipelines, releases, deployments, runners, policy gates, approvals, and audit records.

**Nivora** is an open-source DevOps delivery control plane under the `sevoniva` organization.

The project records delivery intent and state across pipelines, releases, artifacts, deployments, runners, policy decisions, approvals, logs, events, and audit records. It is designed to sit around existing tools, not replace them.

Nivora is **not** Jenkins, Argo CD, Kubernetes, Harbor, a cloud control plane, or a scanner. Those systems remain separate; Nivora models and audits how delivery work moves through them.

Current maturity: **hardened beta-candidate foundation**. Nivora is **not production-ready**. The repository has working backend foundations, PostgreSQL-backed stores for core runtime areas and control-plane catalog metadata, guarded deployment operations, RBAC tests, packaging assets, and verification scripts. Production use still needs more validation around runner isolation, live install/restore drills, external integrations, and production-scale operations.

Future `v1.0.0` documents are planning checklists, not proof that GA has been reached. The current source of truth is [Capability Status](docs/status/CAPABILITY_STATUS.md), with historical audit context in [Implementation Audit](docs/status/IMPLEMENTATION_AUDIT.md).

Enterprise-readiness tracking lives in [Enterprise Production Baseline](docs/status/ENTERPRISE_PRODUCTION_BASELINE.md), [Enterprise Readiness Matrix](docs/status/ENTERPRISE_READINESS_MATRIX.md), [Enterprise Production Readiness Review](docs/status/ENTERPRISE_PRODUCTION_READINESS_REVIEW.md), and [Enterprise Risk Register](docs/status/ENTERPRISE_RISK_REGISTER.md). These documents are release hardening evidence, not production approval.

## Current Status

| Area | Status |
|---|---|
| PipelineRun runtime | Implemented for local shell execution; not a full workflow engine |
| DeploymentRun runtime | Partial; YAML dry-run, guarded apply, inventory, health, diff, audit, and PostgreSQL persistence foundations exist |
| Release and ReleaseExecution | Partial; sequential orchestration and PostgreSQL persistence foundations exist |
| Release target catalog | Foundation; `/api/v1/release-targets` and `nivora target` manage target metadata with PostgreSQL persistence in configured server mode and unsafe operations disabled by default |
| Runner protocol | Partial; tokens, heartbeat, claim, logs, status, and isolation profiles exist; OS-level sandboxing is still operator work |
| Kubernetes YAML | Experimental guarded apply/rollback foundation; no default destructive behavior |
| GitOps / Argo CD | Experimental planning/status/guarded sync foundation; no production Argo automation |
| Artifact / OCI | Partial; OCI parsing, digest foundation, and PostgreSQL-backed registry catalog; no full registry product integration |
| DevSecOps / policy | Foundation; noop/fake scanner paths, built-in rules, and PostgreSQL-backed policy catalog; no Trivy/Cosign/SBOM production integration |
| Secrets / credentials | Partial; metadata, redaction, provider skeletons; production provider lifecycle remains future work |
| Auth / RBAC | Partial; local/token/OIDC foundation and route tests; complete enterprise SSO remains future work |
| Approvals / change windows / notifications | Foundation; backend-only, no ITSM workflow |
| Multi-cloud | Placeholder/foundation inventory only; no cloud deployment |
| Host deployment | Experimental plan/dry-run/noop and guarded SSH surface |
| Web console | Experimental minimal UI that consumes backend APIs |
| MCP control plane | Foundation; local stdio read-only and plan-only AI access, aggregate event/log reads, denied action tools, runner-token rejection, compliance-backed audit, and 29 validated operator scenarios with golden answers; remote MCP remains no-go |
| Integration capability index | Foundation; read-only `/api/v1/integrations` labels built-in, skeleton, noop, foundation, and experimental adapter capabilities |
| Packaging | Partial; Docker Compose, Helm, production-like values, and smoke checks exist |
| Observability / audit | Partial; diagnostics, metrics, runtime recovery center, production doctor, read-only visualization API index, runbooks, and audit/evidence export foundations; production retention/export still needs hardening |

Current focus:

```text
keep public status accurate
keep examples and docs aligned with implemented behavior
stabilize CI, packaging, and local demo paths
continue runtime, install, restore, runner, and audit hardening
turn operator-facing checks into repeatable product workflows
```

Status references:

- [Alpha Capability Matrix](docs/ALPHA_CAPABILITY_MATRIX.md)
- [Beta Capability Matrix](docs/BETA_CAPABILITY_MATRIX.md)
- [API Inventory](docs/API_INVENTORY.md)
- [Alpha Demo Guide](docs/demo/alpha-demo.md)
- [v0.1.0-alpha.1 Checklist](docs/releases/v0.1.0-alpha.1-checklist.md)
- [v0.5.0-beta Checklist](docs/releases/v0.5.0-beta-checklist.md)
- [v0.5.0-beta Release Notes Draft](docs/releases/v0.5.0-beta-release-notes-draft.md)
- [v1.0.0-rc.1 Checklist](docs/releases/v1.0.0-rc.1-checklist.md)
- [Future v1.0.0 GA Readiness Capability Matrix](docs/releases/v1.0.0-ga-capability-matrix.md)
- [Future v1.0.0 GA Readiness Checklist](docs/releases/v1.0.0-ga-checklist.md)
- [Future v1.0.0 Release Notes Draft](docs/releases/v1.0.0-release-notes.md)
- [Implementation Audit](docs/status/IMPLEMENTATION_AUDIT.md)
- [Capability Status](docs/status/CAPABILITY_STATUS.md)
- [AI Control Plane Product Review](docs/status/AI_CONTROL_PLANE_PRODUCT_REVIEW.md)
- [AI Control Plane Beta Readiness](docs/status/AI_CONTROL_PLANE_BETA_READINESS.md)
- [AI Control Plane Deep Audit](docs/status/AI_CONTROL_PLANE_DEEP_AUDIT.md)
- [AI Operator Journeys](docs/status/AI_OPERATOR_JOURNEYS.md)
- [AI Control Plane Go / No-Go](docs/status/AI_CONTROL_PLANE_GO_NO_GO.md)
- [Remote MCP Readiness Audit](docs/status/REMOTE_MCP_READINESS_AUDIT.md)
- [MCP Enterprise Opening Decision](docs/status/MCP_ENTERPRISE_OPENING_DECISION.md)
- [Enterprise Production Readiness Review](docs/status/ENTERPRISE_PRODUCTION_READINESS_REVIEW.md)
- [Enterprise Next Goals](docs/status/ENTERPRISE_NEXT_GOALS.md)
- [Security Threat Model](docs/security/threat-model.md)
- [MCP Threat Model](docs/security/mcp-threat-model.md)
- [Security Review Checklist](docs/security/security-review-checklist.md)
- [User Guide](docs/user/README.md)
- [Operator Guide](docs/operator/README.md)
- [Developer Guide](docs/developer/README.md)
- [Tutorials](docs/tutorials/README.md)
- [Release Playbook](docs/releases/release-playbook.md)
- [Production-Direction Install](docs/operations/production-install.md)
- [Production Doctor](docs/operations/production-doctor.md)
- [Upgrade Guide](docs/operations/upgrade.md)
- [Release Automation](docs/operations/release-automation.md)
- [Changelog](CHANGELOG.md)

## Why Nivora Exists

Delivery state is often split across several systems.

| Area | Common Tools |
|---|---|
| Source control | GitHub, GitLab, Gitea |
| CI execution | Jenkins, GitLab CI, GitHub Actions, Tekton |
| Artifact storage | Harbor, Nexus, JFrog, OCI registries, S3 |
| Kubernetes delivery | kubectl, Helm, Kustomize |
| GitOps | Argo CD |
| Host deployment | SSH, systemd, scripts |
| Cloud targets | AWS, Aliyun, Tencent Cloud |
| Security | Trivy, Cosign, SBOM tooling, policy engines |
| Observability | OpenTelemetry, Prometheus, logs |
| Human process | approvals, change windows, release audit |

The issue is not the individual tools. The issue is that delivery intent, execution state, audit, policy, artifact traceability, and rollback context are often stored separately.

Nivora provides a backend control-plane model for that state.

## Product Positioning

Nivora is a **delivery control plane**. It is not only a CI tool, and it is not only a CD tool.

It coordinates:

```text
source code
-> pipeline execution
-> artifact selection
-> policy evaluation
-> approval
-> deployment
-> verification
-> rollback
-> audit
-> timeline
```

Nivora aims to answer operational questions such as:

- Which commit produced this release?
- Which artifact was deployed?
- Who approved the production deployment?
- Which runner executed the job?
- Which policy gates passed or failed?
- Which environment received the release?
- What changed between two deployments?
- What logs, events, and audit records belong to this delivery?
- Can this deployment be rolled back safely?
- Which external systems participated in the delivery?

## Nivora Value Map

This diagram shows the intended boundaries between external systems, Nivora's control plane, execution mechanisms, and delivery records.

```mermaid
flowchart LR
    subgraph A["External Delivery Systems"]
        A1["Git Providers<br/>GitHub / GitLab / Gitea"]
        A2["Artifact Registries<br/>Harbor / Nexus / OCI / S3"]
        A3["Delivery Targets<br/>Hosts / Kubernetes / Argo CD / Cloud"]
        A4["Security Tools<br/>Trivy / Cosign / Policy Engines"]
        A5["Human Process<br/>Approval / Change Window / Audit"]
    end

    subgraph B["Nivora Delivery Control Plane"]
        B1["Application & Environment Model"]
        B2["Pipeline & Release Orchestration"]
        B3["Runner & Executor Coordination"]
        B4["Policy Gates & Approval"]
        B5["Artifact & Version Traceability"]
        B6["Audit, Events & Timeline"]
        B7["Open APIs & Future Visualization"]
    end

    subgraph C["Execution Plane"]
        C1["Host Runner"]
        C2["Kubernetes Runner"]
        C3["GitOps Runner"]
        C4["Cloud Runner"]
        C5["Local / Dev Runner"]
    end

    subgraph D["Delivery Records"]
        D1["Repeatable PipelineRun"]
        D2["Auditable DeploymentRun"]
        D3["Immutable Artifact Release"]
        D4["Controlled Rollback"]
        D5["Observable Delivery Timeline"]
        D6["Multi-Target Delivery"]
    end

    A1 --> B
    A2 --> B
    A3 --> B
    A4 --> B
    A5 --> B
    B --> C
    C --> D
    B --> D
```

## What Nivora Is

Nivora is a delivery control plane. It coordinates:

- Pipeline execution
- Release planning
- Deployment execution
- Runner assignment
- Executor selection
- Artifact traceability
- Policy evaluation
- Approval flow
- Audit records
- Runtime events
- Delivery timeline
- Visualization API read models

Nivora starts as a **modular monolith** with multiple binaries:

```text
nivora-server
nivora-worker
nivora-runner
nivora CLI
```

This keeps the project understandable while preserving a path toward future service extraction.

## What Nivora Is Not

Nivora is not:

- a Jenkins clone
- an Argo CD replacement
- a Kubernetes-only platform
- a cloud-provider-specific system
- a frontend-first project
- a black-box automation tool
- a statement that every modeled integration has completed production validation

Nivora should integrate with existing systems through explicit ports and adapters.

## Target Architecture

The target architecture separates the **Control Plane** from the **Execution Plane**.

The control plane owns state, orchestration, policies, audit, APIs, and integration configuration. The execution plane owns job execution, logs, heartbeats, and runtime results.

```mermaid
flowchart TB
    U1["Users / Maintainers"]
    U2["CLI"]
    U3["Future Web UI"]
    U4["Git Webhooks"]

    subgraph CP["Control Plane"]
        API["API Server<br/>REST / OpenAPI"]
        AUTH["AuthN / AuthZ<br/>future OIDC / RBAC"]
        ORCH["Workflow Orchestrator<br/>PipelineRun / DeploymentRun"]
        POLICY["Policy Engine<br/>Gates / Approval / Windows"]
        INTEG["Integration Manager<br/>SCM / Artifact / Cloud / Secret"]
        AUDIT["Audit Service<br/>Who did what, when, why"]
        EVENT["Event Service<br/>CloudEvents-style"]
        LOGIDX["Log Index<br/>LogChunk metadata"]
    end

    subgraph STATE["State & Storage"]
        DB[("PostgreSQL<br/>source of truth")]
        OBJ[("Object Store<br/>S3 / MinIO / local")]
        BUS[("Event Bus<br/>memory now<br/>NATS / Redis later")]
    end

    subgraph EP["Execution Plane"]
        RM["Runner Manager"]
        R1["Host Runner"]
        R2["Kubernetes Runner"]
        R3["GitOps Runner"]
        R4["Cloud Runner"]
        R5["Local Runner"]
        EX1["Shell Executor"]
        EX2["SSH Executor"]
        EX3["Kubernetes Job Executor"]
        EX4["YAML / Helm Executor"]
        EX5["Argo CD Executor"]
        EX6["Webhook Executor"]
    end

    subgraph EXT["External Systems"]
        SCM["SCM<br/>GitHub / GitLab / Gitea"]
        ART["Artifact Registry<br/>Harbor / Nexus / OCI / S3"]
        K8S["Kubernetes<br/>YAML / Helm / Kustomize"]
        ARGO["Argo CD<br/>GitOps sync"]
        HOST["Hosts<br/>VM / Bare Metal"]
        CLOUD["Cloud Providers<br/>AWS / Aliyun / Tencent"]
        SEC["Security Tools<br/>Trivy / Cosign / SBOM"]
        OBS["Observability<br/>OpenTelemetry / Prometheus / Logs"]
    end

    U1 --> API
    U2 --> API
    U3 -. future .-> API
    U4 --> API
    API --> AUTH
    API --> ORCH
    API --> INTEG
    API --> AUDIT
    API --> EVENT
    ORCH --> POLICY
    ORCH --> RM
    ORCH --> DB
    AUDIT --> DB
    EVENT --> BUS
    LOGIDX --> DB
    LOGIDX --> OBJ
    RM --> R1
    RM --> R2
    RM --> R3
    RM --> R4
    RM --> R5
    R1 --> EX1
    R1 --> EX2
    R2 --> EX3
    R2 --> EX4
    R3 --> EX5
    R4 --> EX6
    R5 --> EX1
    INTEG --> SCM
    INTEG --> ART
    INTEG --> CLOUD
    INTEG --> SEC
    EX2 --> HOST
    EX3 --> K8S
    EX4 --> K8S
    EX5 --> ARGO
    EX6 --> CLOUD
    EVENT --> OBS
```

## Architecture Principles

### Control Plane and Execution Plane Are Separate

The control plane owns API, state, orchestration, policy, audit, integration configuration, and event timeline. The execution plane owns job execution, logs, heartbeat, and runtime result reporting.

The API server should not directly execute deployment jobs.

### Runner and Executor Are Different

```text
Runner = who executes
Executor = how execution happens
```

| Runner | Executor |
|---|---|
| Local Runner | Shell Executor |
| Host Runner | SSH Executor |
| Kubernetes Runner | Kubernetes Job Executor |
| GitOps Runner | Argo CD Executor |
| Cloud Runner | Webhook / Cloud Adapter |

This separation allows Nivora to support many execution environments without rewriting the core orchestration logic.

### GitOps Is One Deployment Mode

Nivora supports GitOps, but GitOps is not the whole product.

Future deployment modes include host deployment, raw Kubernetes YAML, Helm, Kustomize, Argo CD GitOps, webhook-based delivery, and cloud-provider-specific delivery.

### Ports and Adapters First

External systems must be integrated through stable interfaces:

```text
SCMProvider
ArtifactProvider
CloudProvider
Executor
WorkflowRuntime
SecretProvider
NotificationProvider
PolicyEngine
EventBus
ObjectStore
```

The core use cases should depend on capabilities, not concrete vendors.

### Artifacts Should Be Immutable

A release should point to immutable artifacts whenever possible: image digest, immutable version, signed artifact, and SBOM reference. Avoid `latest` tags, implicit rebuilds during deployment, and untracked artifact mutation.

### Audit Is Not Optional

Important delivery actions must be auditable: pipeline started, job assigned, artifact selected, approval granted or rejected, deployment started, rollback executed, policy violation detected, runner registered, and credential used.

Audit records must not contain secret values.

### No Fake Production Readiness

Nivora should be explicit about what exists today and what is target architecture. Early phases must not claim production readiness, complete integrations, durable scheduling, or security guarantees that have not been implemented and verified.

## End-to-End Delivery Flow

This is the long-term flow Nivora is designed around. Early phases implement only the shell-based PipelineRun subset: definition parsing, queued run creation, local runner execution, logs, events, audit records, retry, timeout, cancellation, and timeline queries.

```mermaid
flowchart TB
    START["Git Push / Manual Trigger / API Trigger"]
    INGEST["Trigger Ingestion"]
    PLAN["Create PipelineRun"]
    SNAPSHOT["Execution Snapshot"]
    POLICY{"Pre-check Policy Gates"}
    DENIED["Stop and record policy result"]
    QUEUE["Queue PipelineRun"]
    WORKER["Worker Picks Run"]
    RUNTIME["Build Runtime Plan"]
    SELECT{"Select Runner"}
    RUNNER["Runner"]
    EXECUTOR["Executor"]
    LOGS["Capture Logs"]
    STATUS["Persist Status Transitions"]
    EVENTS["Emit Events"]
    AUDIT["Write Audit Records"]
    APPROVAL{"Approval Required?"}
    DEPLOY["Create DeploymentRun"]
    MODE{"Deployment Mode"}
    VERIFY["Verify"]
    ROLLBACK{"Rollback Needed?"}
    RB["Rollback"]
    TIMELINE["Unified Timeline"]

    START --> INGEST --> PLAN --> SNAPSHOT --> POLICY
    POLICY -->|Denied| DENIED --> TIMELINE
    POLICY -->|Allowed| QUEUE --> WORKER --> RUNTIME --> SELECT
    SELECT --> RUNNER --> EXECUTOR --> LOGS --> STATUS --> EVENTS --> AUDIT
    AUDIT --> APPROVAL
    APPROVAL -->|No| DEPLOY
    APPROVAL -->|Yes| DEPLOY
    DEPLOY --> MODE --> VERIFY --> ROLLBACK
    ROLLBACK -->|Yes| RB --> TIMELINE
    ROLLBACK -->|No| TIMELINE
```

## PipelineRun Runtime Model

This is the first execution foundation Nivora is building. Current implementation is limited to minimal shell-based PipelineRun execution.

```mermaid
sequenceDiagram
    autonumber
    participant User as User / CLI / API
    participant API as API Server
    participant UC as PipelineRun Usecase
    participant Repo as Runtime Repositories
    participant Worker as Worker
    participant Runner as Runner
    participant Exec as Executor
    participant Event as EventBus
    participant Audit as AuditLog

    User->>API: POST /api/v1/pipeline-runs
    API->>UC: CreatePipelineRun(spec)
    UC->>Repo: Persist PipelineRun, StageRun, JobRun, StepRun
    UC->>Event: emit pipeline.run.created
    UC->>Audit: record PipelineRun created
    UC-->>API: PipelineRun ID
    Worker->>Repo: Poll queued PipelineRun
    Worker->>Repo: PipelineRun -> Running
    Worker->>Event: emit pipeline.run.started
    Worker->>Audit: record PipelineRun started
    Worker->>Runner: Assign JobRun
    Runner->>Exec: Run step
    Exec-->>Runner: stdout / stderr / exit code
    Runner->>Repo: Persist LogChunks
    Runner->>Repo: StepRun / JobRun status
    Worker->>Repo: PipelineRun final status
    Worker->>Event: emit completed or failed
    Worker->>Audit: record lifecycle result
    User->>API: GET /api/v1/pipeline-runs/{id}/timeline
    API->>Repo: Query ordered runtime events
    API-->>User: Timeline
```

## PipelineRun State Model

```mermaid
stateDiagram-v2
    [*] --> Pending
    Pending --> Queued
    Queued --> Running
    Running --> Paused
    Paused --> Running
    Running --> Succeeded
    Running --> Failed
    Running --> Timeout
    Pending --> Canceled
    Queued --> Canceled
    Running --> Canceled
    Paused --> Canceled
    Failed --> Retrying
    Retrying --> Queued
    Succeeded --> [*]
    Failed --> [*]
    Timeout --> [*]
    Canceled --> [*]
```

## Runner and Executor Model

```mermaid
flowchart TB
    CP["Control Plane"] --> RM["Runner Manager"]
    RM --> LOCAL["Local Runner"]
    RM --> HOST["Host Runner"]
    RM --> K8S["Kubernetes Runner"]
    RM --> GITOPS["GitOps Runner"]
    RM --> CLOUD["Cloud Runner"]

    LOCAL --> SHELL["Shell Executor"]
    HOST --> SSH["SSH Executor"]
    K8S --> KJOB["Kubernetes Job Executor"]
    K8S --> HYAML["Helm / YAML Executor"]
    GITOPS --> ARGO["Argo CD Executor"]
    CLOUD --> WEBHOOK["Webhook / Cloud Executor"]

    SHELL --> RESULT["Execution Result"]
    SSH --> RESULT
    KJOB --> RESULT
    HYAML --> RESULT
    ARGO --> RESULT
    WEBHOOK --> RESULT
    RESULT --> CP
```

## Deployment Model

Deployment execution is target architecture. It is not implemented as a full production deployment engine in the current phase.

```mermaid
flowchart TB
    APP["Application"]
    ENV["Environment"]
    REL["Release"]
    DR["DeploymentRun"]
    TARGET{"ReleaseTarget Type"}

    APP --> ENV --> REL --> DR --> TARGET

    TARGET --> HOST["HostTarget"]
    TARGET --> K8S["KubernetesTarget"]
    TARGET --> HELM["HelmTarget"]
    TARGET --> KUSTOMIZE["KustomizeTarget"]
    TARGET --> ARGO["ArgoCDTarget"]
    TARGET --> CLOUD["CloudTarget"]
    TARGET --> WEBHOOK["WebhookTarget"]

    HOST --> SSH["SSH Executor"]
    K8S --> YAML["YAML Apply Executor"]
    HELM --> HEX["Helm Executor"]
    KUSTOMIZE --> KREN["Kustomize Renderer"]
    ARGO --> AEX["Argo CD Executor"]
    CLOUD --> CAD["Cloud Adapter"]
    WEBHOOK --> WEX["Webhook Executor"]

    SSH --> VERIFY["Verify"]
    YAML --> VERIFY
    HEX --> VERIFY
    KREN --> VERIFY
    AEX --> VERIFY
    CAD --> VERIFY
    WEX --> VERIFY
    VERIFY --> RESULT{"Result"}
    RESULT -->|Healthy| SUCCESS["Deployment Succeeded"]
    RESULT -->|Unhealthy| ROLLBACK["Rollback Plan"]
```

## Integration Model

All external systems should connect through ports and adapters. The adapter names below are target integration directions unless explicitly documented as implemented.

The read-only `/api/v1/integrations` endpoint exposes the current adapter/plugin capability index. It is metadata only: it does not configure providers, call external services, or return credentials. Skeleton, noop, foundation-only, and experimental adapters are labeled as such.

```bash
go run ./cmd/nivora integrations list --local
go run ./cmd/nivora integrations list --server http://localhost:8080
```

```mermaid
flowchart LR
    subgraph CORE["Core Use Cases"]
        PIPE["Pipeline Usecase"]
        DEPLOY["Deployment Usecase"]
        ARTUC["Artifact Usecase"]
        POLICYUC["Policy Usecase"]
        RUNUC["Runner Usecase"]
    end

    subgraph PORTS["Ports"]
        SCM["SCMProvider"]
        ART["ArtifactProvider"]
        CLOUD["CloudProvider"]
        EXEC["Executor"]
        WF["WorkflowRuntime"]
        SECRET["SecretProvider"]
        POLICY["PolicyEngine"]
        BUS["EventBus"]
        OBJ["ObjectStore"]
    end

    subgraph ADAPTERS["Adapters"]
        SCMAD["GitHub / GitLab / Gitea"]
        ARTAD["Harbor / Nexus / OCI / S3"]
        CLOUDAD["AWS / Aliyun / Tencent"]
        EXECAD["Shell / SSH / K8s Job / Argo CD"]
        SECRETAD["Built-in / Vault / K8s Secret"]
        POLICYAD["Built-in / OPA future"]
        BUSAD["Memory / NATS future"]
        OBJAD["Local / MinIO / S3"]
    end

    CORE --> PORTS
    SCM --> SCMAD
    ART --> ARTAD
    CLOUD --> CLOUDAD
    EXEC --> EXECAD
    WF --> BUSAD
    SECRET --> SECRETAD
    POLICY --> POLICYAD
    BUS --> BUSAD
    OBJ --> OBJAD
```

## Observability and Audit Model

```mermaid
flowchart TB
    RUN["PipelineRun / DeploymentRun"]
    RUN --> LOGS["Logs"]
    RUN --> EVENTS["Events"]
    RUN --> AUDIT["AuditLog"]
    RUN --> METRICS["Metrics"]
    RUN --> TRACES["Traces"]

    LOGS --> TL["Unified Timeline"]
    EVENTS --> TL
    AUDIT --> TL
    METRICS --> DASH["Future Dashboards"]
    TRACES --> DASH

    TL --> API["API / CLI / Future Web UI"]
    DASH --> API
```

## Core Concepts

| Concept | Meaning |
|---|---|
| Application | A product or service managed by Nivora |
| Environment | A delivery context such as dev, staging, prod, or a custom target group |
| ReleaseTarget | A concrete deployment target such as host group, Kubernetes cluster, Argo CD application, cloud target, or webhook target |
| Pipeline | A reusable definition of stages, jobs, and steps |
| PipelineRun | One execution of a Pipeline |
| StageRun | Execution record for one stage |
| JobRun | Execution record for one job |
| StepRun | Execution record for one step |
| Release | A versioned delivery intent, usually tied to immutable artifacts |
| DeploymentRun | One execution of a release or deployment plan against a target |
| Runner | A component that receives and executes jobs |
| Executor | A mechanism used by a Runner to execute work |
| Artifact | A build output such as image, jar, binary, chart, or package |
| Artifact Registry | A system that stores artifacts |
| Policy | A gate that can allow, deny, or require approval |
| AuditLog | Durable record of important actions |
| Event | Runtime signal emitted during delivery lifecycle |
| LogChunk | Ordered stdout, stderr, or system log segment |

## Repository Layout

```text
nivora/
  cmd/
    nivora-server/
    nivora-worker/
    nivora-runner/
    nivora/

  internal/
    app/
    domain/
    usecase/
    ports/
    adapters/
    infra/
    api/

  api/
    openapi/
    asyncapi/
    proto/

  configs/
  deployments/
  examples/
  docs/
  scripts/
  test/

  AGENTS.md
  PROJECT_CHARTER.md
  README.md
  ROADMAP.md
  CONTRIBUTING.md
```

| Directory | Purpose |
|---|---|
| `cmd/` | Binary entrypoints only |
| `internal/domain/` | Pure domain concepts and statuses |
| `internal/usecase/` | Business orchestration |
| `internal/ports/` | External capability interfaces |
| `internal/adapters/` | Implementations of ports |
| `internal/infra/` | Technical infrastructure |
| `internal/api/` | HTTP / gRPC transport |
| `api/` | OpenAPI, AsyncAPI, proto definitions |
| `docs/` | Architecture, roadmap, concepts, community docs |
| `examples/` | Example pipelines and deployment specs |

## Quick Start

### Prerequisites

- Go
- Make
- Docker, optional for local compose
- PostgreSQL, optional depending on runtime mode

### Build

```bash
make build
```

### Test

```bash
make test
```

### Verify

```bash
make verify
```

### Package

```bash
make docker-build
make helm-template
make helm-lint
```

Packaging docs:

- [Docker Compose install](docs/operations/install-docker-compose.md)
- [Kubernetes install](docs/operations/install-kubernetes.md)
- [Configuration](docs/operations/configuration.md)
- [Performance and load testing](docs/operations/performance.md)
- [Backup and restore](docs/operations/backup-restore.md)
- [HA and disaster recovery](docs/operations/ha-disaster-recovery.md)

### Smoke Tests

```bash
make smoke-local
make smoke-api
```

### Run Server

```bash
make run-server
```

### Run Web UI

```bash
make run-web
```

The web console lives under `web/` and consumes existing runtime and `/api/v1/visualization/*` backend APIs. It is a minimal Phase 6.4 foundation, not a complete frontend product.

### Health Check

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/version
curl http://localhost:8080/api/v1/system/runtime
curl http://localhost:8080/api/v1/system/diagnostics
curl http://localhost:8080/metrics
```

`/readyz` and `/api/v1/system/diagnostics` include lightweight dependency checks for database, object store, event bus, outbox recovery, and runner reconnect posture.

### Run Worker

```bash
make run-worker
```

### Run Runner

```bash
make run-runner
```

### CLI

```bash
go run ./cmd/nivora version
go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml
go run ./cmd/nivora pipeline get <pipeline-run-id> --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora pipeline logs <pipeline-run-id> --server http://localhost:8080 --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora pipeline timeline <pipeline-run-id> --server http://localhost:8080
go run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml
go run ./cmd/nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml
go run ./cmd/nivora deployment apply --local examples/deployments/yaml-apply-local.yaml --confirm
go run ./cmd/nivora deployment host plan --file examples/deployments/host-dry-run.yaml --local
go run ./cmd/nivora deployment host run --file examples/deployments/host-dry-run.yaml --local
go run ./cmd/nivora release plan --file examples/releases/multi-target-release.yaml --local
go run ./cmd/nivora release deploy --file examples/releases/sequential-release.yaml --local
go run ./cmd/nivora cloud providers --local
go run ./cmd/nivora plugins list --local
go run ./cmd/nivora plugins inspect artifact-oci --local
go run ./cmd/nivora plugins validate --local --file examples/plugins/templates/scanner-plugin.yaml
```

## Local Development

Nivora supports local development through the Makefile, docker-compose, a local object store, a memory event bus, the shell executor, and example pipelines.

This repository uses a neutral default Go proxy in local tooling:

```bash
GOPROXY=https://proxy.golang.org,direct
```

Developers in China can override it without changing project defaults:

```bash
GOPROXY=https://goproxy.cn,direct make verify
```

or:

```bash
export GOPROXY=https://goproxy.cn,direct
make verify
```

## Example Pipeline

```yaml
apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: hello-shell
spec:
  stages:
    - name: build
      jobs:
        - name: echo
          executor: shell
          steps:
            - name: say-hello
              run: echo "hello from nivora"
```

Run it locally:

```bash
go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml
```

## Example YAML Deployment Dry-Run

The current Phase 2 foundation supports non-destructive YAML deployment planning and dry-run validation, plus explicit local no-op apply for runtime testing. It renders static manifests, validates their basic shape, creates a DeploymentPlan, records resource inventory, verifies manifest images against bound artifacts, records logs/events/audit/timeline data, and does not apply resources to a cluster by default.

```yaml
apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo-yaml-deployment
spec:
  application: demo-springboot
  environment: dev
  target:
    type: kubernetes-yaml
    name: dev-kind
    namespace: default
  manifests:
    - examples/yaml/configmap.yaml
    - examples/yaml/deployment.yaml
    - examples/yaml/service.yaml
  options:
    dryRun: true
    apply: false
```

Run it locally:

```bash
go run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml
go run ./cmd/nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml
```

Explicit local apply requires a separate command and confirmation:

```bash
go run ./cmd/nivora deployment apply --local examples/deployments/yaml-apply-local.yaml --confirm
```

The default local apply path uses the safe no-op manifest client. Production Kubernetes apply semantics, Helm, Kustomize, Argo CD, cloud providers, remote host deployment, and registry integrations remain future work.

## Example Host Deployment Dry-Run

Phase 8.1 hardens the safe host deployment foundation. It can build a plan for deploying a binary package to versioned release directories, switching symlinks, checking HTTP/TCP/command health, running batches, and preparing guarded symlink rollback. The default runtime uses a noop host executor and does not execute remote SSH.

```bash
go run ./cmd/nivora deployment host plan --file examples/deployments/host-dry-run.yaml --local
go run ./cmd/nivora deployment host run --file examples/deployments/host-dry-run.yaml --local
```

Remote host deployment remains disabled unless an adapter transport is explicitly configured with credential references, confirmation, and allow flags.

## Example Multi-Target Release

Phase 2.7 adds a local ReleasePlan / ReleaseExecution foundation. It can plan a Release across multiple targets and execute safe targets sequentially through target-level DeploymentRuns or placeholder targets.

```bash
go run ./cmd/nivora release plan --file examples/releases/multi-target-release.yaml --local
go run ./cmd/nivora release deploy --file examples/releases/sequential-release.yaml --local
```

Server-backed release and deployment commands are RBAC-protected. Use `--token-env NIVORA_AUTH_TOKEN` for server calls instead of passing token values directly.

This is not a production workflow engine. Parallel execution, durable approvals, host/cloud targets, and production GitOps automation remain future work.

Run a minimal shell PipelineRun through the API:

```bash
curl -X POST http://localhost:8080/api/v1/pipeline-runs \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Pipeline",
    "metadata": {"name": "hello-shell"},
    "spec": {
      "stages": [{
        "name": "build",
        "jobs": [{
          "name": "echo",
          "executor": "shell",
          "steps": [{"name": "say-hello", "run": "echo hello from nivora"}]
        }]
      }]
    }
  }'
```

Unimplemented API groups return structured responses, not fake data:

```json
{
  "code": "not_implemented",
  "message": "This endpoint is reserved for a future phase.",
  "path": "/api/v1/integrations"
}
```

## Events

Nivora uses CloudEvents-style event envelopes.

```json
{
  "specversion": "1.0",
  "id": "evt_01HX",
  "type": "devops.pipeline.run.started",
  "source": "/projects/example/pipelines/hello-shell",
  "subject": "pipelineRun/pr_123",
  "time": "2026-05-18T10:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "pipelineRunId": "pr_123",
    "status": "Running"
  }
}
```

OpenAPI definitions live under `api/openapi/openapi.yaml`. AsyncAPI definitions live under `api/asyncapi/asyncapi.yaml`.

Core API groups include:

```text
/api/v1/orgs
/api/v1/projects
/api/v1/applications
/api/v1/environments
/api/v1/repositories
/api/v1/artifact-registries
/api/v1/pipelines
/api/v1/pipeline-runs
/api/v1/jobs
/api/v1/releases
/api/v1/deployments
/api/v1/runners
/api/v1/approvals
/api/v1/policies
/api/v1/audit-logs
/api/v1/events
/api/v1/logs
/api/v1/integrations
/api/v1/visualization
```

Aggregate runtime inspection also has CLI entry points:

```bash
nivora events search --pipeline-run-id <pipeline-run-id> --limit 50
nivora logs search --pipeline-run-id <pipeline-run-id> --contains "error"
nivora audit search --subject-id <subject-id> --scope-type project --scope-id <project-id>
```

## Roadmap

```mermaid
flowchart LR
    P0["Phase 0<br/>Backend Skeleton"]
    P05["Phase 0.5<br/>Guardrails"]
    P06["Phase 0.6<br/>Public Planning"]
    P1["Phase 1<br/>Minimal Runtime"]
    P15["Phase 1.5<br/>Durable Runtime Foundation"]
    P16["Phase 1.6<br/>Runtime DX & Acceptance"]
    P2["Phase 2.0<br/>YAML Planning Foundation"]
    P21["Phase 2.1<br/>Kubernetes YAML Runtime"]
    P22["Phase 2.2<br/>Artifact & Release Binding"]
    P23["Phase 2.3<br/>GitOps & Argo CD Foundation"]
    P24["Phase 2.4<br/>Kubernetes Inventory & Rollback Foundation"]
    P25["Phase 2.5<br/>OCI Digest Resolution"]
    P26["Phase 2.6<br/>Argo CD Guarded Sync"]
    P27["Phase 2.7<br/>Release Orchestration"]
    P30["Phase 3.0<br/>DevSecOps Foundation"]
    P31["Phase 3.1<br/>Secret & Credential Foundation"]
    P32["Phase 3.2<br/>Auth & RBAC Foundation"]
    P33["Phase 3.3<br/>Approvals & Change Windows"]
    P34["Phase 3.4<br/>Multi-cloud Inventory"]
    P35["Phase 3.5<br/>Host Deployment Foundation"]
    P36["Phase 3.6<br/>Durable Runner Runtime"]
    P3["Future Phase 3<br/>Multi-cloud & DevSecOps"]
    P40["Phase 4.0<br/>Visualization Backend APIs"]
    P41["Phase 4.1<br/>Web UI Foundation"]
    P4["Future Phase 4<br/>Frontend Visualization"]

    P0 --> P05 --> P06 --> P1 --> P15 --> P16 --> P2 --> P21 --> P22 --> P23 --> P24 --> P25 --> P26 --> P27 --> P30 --> P31 --> P32 --> P33 --> P34 --> P35 --> P36 --> P3 --> P40 --> P41 --> P4
```

See [ROADMAP.md](ROADMAP.md) and [docs/roadmap/overview.md](docs/roadmap/overview.md) for details.

## Contribution Map

```mermaid
flowchart TB
    C["Contributor"]
    C --> G1["Good First Contributions"]
    C --> G2["Intermediate Contributions"]
    C --> G3["Advanced Contributions"]
    C --> G4["Requires RFC"]
    G1 --> D1["Documentation"]
    G1 --> D2["Examples"]
    G1 --> D3["Tests"]
    G1 --> D4["CLI polish"]
    G1 --> D5["API schema cleanup"]
    G2 --> I1["Shell executor improvements"]
    G2 --> I2["Memory event bus"]
    G2 --> I3["Local object store"]
    G2 --> I4["Config validation"]
    G2 --> I5["Pipeline state tests"]
    G3 --> A1["PipelineRun state machine"]
    G3 --> A2["Runner protocol"]
    G3 --> A3["Log streaming"]
    G3 --> A4["Persistence"]
    G3 --> A5["Kubernetes Job executor"]
    G3 --> A6["YAML renderer"]
    G3 --> A7["Argo CD adapter"]
    G3 --> A8["Policy engine"]
    G4 --> R1["Runner protocol changes"]
    G4 --> R2["Workflow runtime changes"]
    G4 --> R3["Database model changes"]
    G4 --> R4["Cloud provider adapters"]
    G4 --> R5["Kubernetes / Argo CD integration design"]
    G4 --> R6["Plugin system changes"]
    G4 --> R7["Security model changes"]
    G4 --> R8["Public API breaking changes"]
```

Before contributing, read:

- [AGENTS.md](AGENTS.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [PROJECT_CHARTER.md](PROJECT_CHARTER.md)
- [docs/README.md](docs/README.md)
- [docs/rfcs/README.md](docs/rfcs/README.md)
- [docs/architecture/architecture-contract.md](docs/architecture/architecture-contract.md)
- [docs/architecture/module-boundaries.md](docs/architecture/module-boundaries.md)
- [docs/engineering/testing-policy.md](docs/engineering/testing-policy.md)
- [docs/engineering/dependency-policy.md](docs/engineering/dependency-policy.md)

Basic expectations:

- keep changes small
- preserve architecture boundaries
- do not add speculative abstractions
- do not commit secrets
- do not claim production readiness
- update docs when architecture changes
- update OpenAPI / AsyncAPI when public behavior changes
- add tests for behavior changes

## Contributor Automation

Automated coding tools and human contributors use the same repository rules. The canonical instruction file is [AGENTS.md](AGENTS.md).

Tool-specific instruction files should point to `AGENTS.md` instead of defining conflicting behavior. All changes must preserve architecture boundaries, phase boundaries, dependency policy, testing policy, security baseline, and documentation consistency.

## Verification

Run the full verification suite:

```bash
make verify
```

Expected checks include:

```text
gofmt check
go mod tidy check
go vet ./...
go test ./...
go build ./cmd/nivora-server
go build ./cmd/nivora-worker
go build ./cmd/nivora-runner
go build ./cmd/nivora
architecture verification
secret scanning
```

## Security

Nivora must not commit or expose secrets.

Do not commit tokens, passwords, private keys, kubeconfigs, cloud credentials, registry credentials, or realistic-looking fake credentials. Secret values must not be logged, returned by normal APIs, stored in audit records, embedded in examples, or embedded in tests.

See [SECURITY.md](SECURITY.md) and [docs/engineering/security-baseline.md](docs/engineering/security-baseline.md).

Phase 3.0 adds local DevSecOps foundations:

```bash
go run ./cmd/nivora security scan artifact registry.example.com/demo/app:latest --local
go run ./cmd/nivora security scan manifest examples/security/manifest-privileged-warning.yaml --local
go run ./cmd/nivora policy evaluate --subject registry.example.com/demo/app:latest
```

These commands use noop/fake-friendly scanner foundations and built-in policy gates. Trivy, Cosign, SBOM generation, OPA, Kyverno, Gatekeeper, and production security automation remain future work.

Phase 3.1 adds SecretRef and Credential metadata:

```bash
go run ./cmd/nivora secret put --name local-registry-token --value-env NIVORA_TOKEN --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora secret provider validate --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora credential create --file examples/credentials/registry-credential.yaml --token-env NIVORA_AUTH_TOKEN
```

Secret values are accepted only at creation and rotation boundaries and are not returned by normal APIs. Server-backed commands should use `--token-env` so API tokens stay out of shell history; in-process development paths can use `--local` where the command supports it. The builtin provider is development-only. Phase 7.1 adds Vault and Kubernetes Secret adapter foundations plus cloud KMS placeholders; production external secret storage remains future work.

Phase 7.0 hardens the local auth and RBAC foundations:

```bash
go run ./cmd/nivora auth whoami
go run ./cmd/nivora auth users
go run ./cmd/nivora auth roles
go run ./cmd/nivora auth permissions
go run ./cmd/nivora project members add <project-id> --user-id <user-id> --role developer
go run ./cmd/nivora auth service-account create --name ci --role developer
go run ./cmd/nivora auth token create --subject-id <service-account-id>
```

Dev auth is not production authentication. Static token mode reads token values from environment variables. OIDC is provider-configured backend foundation work; full browser SSO and provider lifecycle operations remain future work.

System diagnostics can be read through CLI or HTTP:

```bash
go run ./cmd/nivora system runtime
go run ./cmd/nivora system diagnostics
```

Phase 7.2 adds multi-tenancy and quota foundations:

```bash
go run ./cmd/nivora quota view --scope-type project --scope-id demo --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora usage summary --scope-type project --scope-id demo --token-env NIVORA_AUTH_TOKEN
```

Scoped API tokens can be constrained to org/project/environment-style boundaries, and quota read models expose concurrency, runner, artifact, log storage, and rate-limit foundations. Persistent distributed quota enforcement remains future work.

Phase 7.3 adds compliance audit and evidence foundations:

```bash
go run ./cmd/nivora audit search --subject <subject-id>
go run ./cmd/nivora evidence list --subject-type pipelineRun --subject-id <pipeline-run-id> --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora evidence export pipelineRun <pipeline-run-id> --format markdown --token-env NIVORA_AUTH_TOKEN
```

Evidence bundles collect safe release, artifact, approval, policy, security, deployment, log-reference, event, and audit context. Secret-like values are redacted before export; immutable external audit storage and retention enforcement jobs remain future work.

## Documentation

| Document | Purpose |
|---|---|
| [PROJECT_CHARTER.md](PROJECT_CHARTER.md) | Project purpose and principles |
| [ROADMAP.md](ROADMAP.md) | High-level roadmap |
| [docs/README.md](docs/README.md) | Documentation index |
| [docs/architecture/](docs/architecture/overview.md) | Architecture model |
| [docs/concepts/](docs/concepts/overview.md) | Core concepts |
| [docs/product/](docs/product/vision.md) | Product planning |
| [docs/community/](docs/community/governance.md) | Contribution and governance |
| [docs/rfcs/](docs/rfcs/README.md) | RFC process |
| [docs/adr/](docs/adr/0001-use-go-as-primary-language.md) | Architecture decision records |
| [AGENTS.md](AGENTS.md) | Automation and contribution rules |

## Design North Star

Nivora is being built to make delivery systems more coherent. It does not assume one tool, one cloud, one runtime, or one deployment model.

The long-term goal is to provide a delivery control plane where:

```text
pipelines are repeatable
releases are artifact-based
deployments are auditable
policies are explicit
runners are isolated
integrations are replaceable
events are observable
rollback is traceable
```

Nivora starts small. The first milestone is not to support every tool. The first milestone is to build the correct foundation.

## License

Nivora is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
