# Nivora

<p align="center">
  <strong>🌐 Languages</strong>:
  <a href="README.md">English</a> |
  <a href="README.zh-CN.md">中文</a> |
  <a href="README.ja-JP.md">日本語</a> |
  <a href="README.ko-KR.md">한국어</a> |
  <a href="README.es-ES.md">Español</a>
</p>

> Plano de control de entrega backend-first para pipelines, lanzamientos, despliegues, runners, puertas de política, aprobaciones y registros de auditoría.

**Nivora** es un plano de control de entrega DevOps de código abierto bajo la organización `sevoniva`.

El proyecto registra la intención y el estado de entrega a través de pipelines, lanzamientos, artefactos, despliegues, runners, decisiones de política, aprobaciones, registros, eventos y registros de auditoría. Está diseñado para rodear las herramientas existentes, no para reemplazarlas.

Nivora **no** es Jenkins, Argo CD, Kubernetes, Harbor, un plano de control en la nube ni un escáner. Esos sistemas permanecen separados; Nivora modela y audita cómo el trabajo de entrega se mueve a través de ellos.

Madurez actual: **fundación beta-candidata endurecida**. Nivora **no está listo para producción**. El repositorio tiene fundamentos de backend funcionales, almacenes respaldados por PostgreSQL para áreas de ejecución principales y metadatos del catálogo del plano de control, operaciones de despliegue protegidas, pruebas RBAC, activos de empaquetado y scripts de verificación. El uso en producción aún necesita más validación en torno al aislamiento de runners, pruebas de instalación/restauración en vivo, integraciones externas y operaciones a escala de producción.

Los documentos futuros `v1.0.0` son listas de verificación de planificación, no prueba de que se ha alcanzado GA. La fuente actual de verdad es [Capability Status](docs/status/CAPABILITY_STATUS.md), con contexto histórico de auditoría en [Implementation Audit](docs/status/IMPLEMENTATION_AUDIT.md).

El seguimiento de preparación empresarial se encuentra en [Enterprise Production Baseline](docs/status/ENTERPRISE_PRODUCTION_BASELINE.md), [Enterprise Readiness Matrix](docs/status/ENTERPRISE_READINESS_MATRIX.md), [Enterprise Production Readiness Review](docs/status/ENTERPRISE_PRODUCTION_READINESS_REVIEW.md) y [Enterprise Risk Register](docs/status/ENTERPRISE_RISK_REGISTER.md). Estos documentos son evidencia de endurecimiento de lanzamiento, no aprobación de producción.

## Current Status

| Área | Estado |
|---|---|
| PipelineRun runtime | Implementado para ejecución local en shell con registros/eventos/auditoría más lecturas de metadatos de artefacto/caché/anotación/resumen; no es un motor de flujo de trabajo completo |
| DeploymentRun runtime | Parcial; existen fundamentos de ejecución en seco YAML, apply protegido, inventario, salud, diff, auditoría y persistencia en PostgreSQL |
| Release y ReleaseExecution | Parcial; existen fundamentos de orquestación secuencial y persistencia en PostgreSQL |
| Catálogo de release targets | Fundación; `/api/v1/release-targets` y `nivora target` gestionan metadatos de objetivos con persistencia en PostgreSQL en modo servidor configurado y operaciones inseguras deshabilitadas por defecto |
| Catálogo de repositorios / inteligencia | Fundación; catálogo de metadatos de repositorios, `nivora repository create --file` basado en archivos, snapshotting local/genérico de solo lectura, detección estática de lenguaje/build/test/package, resúmenes DevOps de solo planificación, almacenamiento de snapshot/inteligencia respaldado por PostgreSQL en modo servidor/MCP configurado, `nivora repository inspect/snapshot/analyze/devops-plan`, y herramientas MCP de lectura/planificación de repositorios existen; las escrituras externas SCM siguen siendo trabajo futuro |
| Nivora Workflow | Fundación; parser de `.nivora/workflows/*.yaml`, validador, planificador DAG/matriz, sugerencias de artefacto/caché, intención de seguridad/lanzamiento/despliegue de solo planificación, conversión de definición de Pipeline, registros de planes almacenados, metadatos protegidos de WorkflowRun, `nivora workflow validate/plan/run/cancel/reconcile/retry`, y superficies API/MCP de solo planificación existen; WorkflowRun puede encolar/cancelar/reintentar registros PipelineRun vinculados, reconciliar estado desde el estado PipelineRun vinculado y registrar metadatos de artefacto/caché, pero no es un motor de flujo de trabajo completo |
| Protocolo runner | Parcial; existen tokens, latidos, claim, registros, estado y perfiles de aislamiento; el aislamiento a nivel de SO sigue siendo trabajo del operador |
| Kubernetes YAML | Fundación experimental de apply/reversión protegida; sin comportamiento destructivo por defecto |
| GitOps / Argo CD | Fundación experimental de planificación/estado/sync protegido; sin automatización Argo de producción |
| Artefacto / OCI | Parcial; parsing OCI, fundación de digest y catálogo de registro respaldado por PostgreSQL; sin integración completa de producto de registro |
| DevSecOps / política | Fundación; rutas de escáner noop/falso, reglas integradas y catálogo de política respaldado por PostgreSQL; sin integración de producción Trivy/Cosign/SBOM |
| Secretos / credenciales | Parcial; metadatos, redacción, esqueletos de proveedor; el ciclo de vida del proveedor en producción sigue siendo trabajo futuro |
| Auth / RBAC | Parcial; fundación local/token/OIDC y pruebas de rutas; SSO empresarial completo sigue siendo trabajo futuro |
| Aprobaciones / ventanas de cambio / notificaciones | Fundación; solo backend, sin flujo de trabajo ITSM |
| Multi-cloud | Solo inventario placeholder/fundación; sin despliegue en la nube |
| Despliegue en host | Superficie experimental de plan/ejecución en seco/noop y SSH protegido |
| Consola web | UI mínima experimental que consume APIs de backend |
| Plano de control MCP | Fundación; acceso AI de solo lectura y solo planificación por stdio local más JSON-RPC remoto experimental opt-in de solo lectura, herramientas de plan de repositorio/flujo de trabajo, lecturas agregadas de eventos/registros, herramientas de acción denegada, rechazo de tokens de runner, auditoría respaldada por cumplimiento y 31 escenarios de operador validados con respuestas doradas; MCP remoto no está ampliamente expuesto ni listo para producción |
| Índice de capacidad de integración | Fundación; `/api/v1/integrations` de solo lectura etiqueta capacidades de adaptador integradas, esqueleto, noop, fundación y experimentales |
| Empaquetado | Parcial; Docker Compose, Helm, valores similares a producción y pruebas de humo existen |
| Observabilidad / auditoría | Parcial; existen fundamentos de métricas, centro de recuperación de ejecución, diagnóstico de producción, índice de API de visualización de solo lectura, libros de jugadas y exportación de auditoría/evidencia; la retención/exportación en producción aún necesita endurecimiento |

Enfoque actual:

```text
mantener el estado público preciso
mantener ejemplos y documentación alineados con el comportamiento implementado
estabilizar CI, empaquetado y rutas de demo local
continuar el endurecimiento de ejecución, instalación, restauración, runner y auditoría
convertir verificaciones orientadas al operador en flujos de trabajo de producto repetibles
```

Referencias de estado:

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
- [Production Diagnostics](docs/operations/production-doctor.md)
- [Upgrade Guide](docs/operations/upgrade.md)
- [Release Automation](docs/operations/release-automation.md)
- [Changelog](CHANGELOG.md)

## Por qué existe Nivora

El estado de entrega suele estar dividido entre varios sistemas.

| Área | Herramientas comunes |
|---|---|
| Control de fuentes | GitHub, GitLab, Gitea |
| Ejecución CI | Jenkins, GitLab CI, GitHub Actions, Tekton |
| Almacenamiento de artefactos | Harbor, Nexus, JFrog, registros OCI, S3 |
| Entrega Kubernetes | kubectl, Helm, Kustomize |
| GitOps | Argo CD |
| Despliegue en host | SSH, systemd, scripts |
| Objetivos en la nube | AWS, Aliyun, Tencent Cloud |
| Seguridad | Trivy, Cosign, herramientas SBOM, motores de política |
| Observabilidad | OpenTelemetry, Prometheus, registros |
| Proceso humano | aprobaciones, ventanas de cambio, auditoría de lanzamiento |

El problema no son las herramientas individuales. El problema es que la intención de entrega, el estado de ejecución, la auditoría, la política, la trazabilidad de artefactos y el contexto de reversión suelen almacenarse por separado.

Nivora proporciona un modelo de plano de control backend para ese estado.

## Posicionamiento del producto

Nivora es un **plano de control de entrega**. No es solo una herramienta CI, y no es solo una herramienta CD.

Coordina:

```text
código fuente
-> ejecución de pipeline
-> selección de artefacto
-> evaluación de política
-> aprobación
-> despliegue
-> verificación
-> reversión
-> auditoría
-> línea temporal
```

Nivora busca responder preguntas operativas como:

- ¿Qué commit produjo este lanzamiento?
- ¿Qué artefacto fue desplegado?
- ¿Quién aprobó el despliegue en producción?
- ¿Qué runner ejecutó el trabajo?
- ¿Qué puertas de política pasaron o fallaron?
- ¿Qué entorno recibió el lanzamiento?
- ¿Qué cambió entre dos despliegues?
- ¿Qué registros, eventos y registros de auditoría pertenecen a esta entrega?
- ¿Se puede revertir este despliegue de forma segura?
- ¿Qué sistemas externos participaron en la entrega?

## Mapa de valor de Nivora

Este diagrama muestra los límites previstos entre sistemas externos, el plano de control de Nivora, los mecanismos de ejecución y los registros de entrega.

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

## Qué es Nivora

Nivora es un plano de control de entrega. Coordina:

- Ejecución de pipeline
- Planificación de lanzamiento
- Ejecución de despliegue
- Asignación de runners
- Selección de executors
- Trazabilidad de artefactos
- Evaluación de política
- Flujo de aprobación
- Registros de auditoría
- Eventos de ejecución
- Línea temporal de entrega
- Modelos de lectura de API de visualización

Nivora comienza como un **monolito modular** con múltiples binarios:

```text
nivora-server
nivora-worker
nivora-runner
nivora CLI
```

Esto mantiene el proyecto comprensible mientras preserva un camino hacia la futura extracción de servicios.

## Qué no es Nivora

Nivora no es:

- un clon de Jenkins
- un reemplazo de Argo CD
- una plataforma exclusiva de Kubernetes
- un sistema específico para un proveedor de nube
- un proyecto frontend-first
- una herramienta de automatización de caja negra
- una declaración de que cada integración modelada ha completado la validación en producción

Nivora debe integrarse con los sistemas existentes a través de puertos y adaptadores explícitos.

## Arquitectura objetivo

La arquitectura objetivo separa el **plano de control** del **plano de ejecución**.

El plano de control posee el estado, la orquestación, las políticas, la auditoría, las APIs y la configuración de integración. El plano de ejecución posee la ejecución de trabajos, registros, latidos y resultados de ejecución.

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

## Principios de arquitectura

### El plano de control y el plano de ejecución están separados

El plano de control posee la API, el estado, la orquestación, la política, la auditoría, la configuración de integración y la línea temporal de eventos. El plano de ejecución posee la ejecución de trabajos, registros, latidos y reporte de resultados de ejecución.

El servidor API no debe ejecutar directamente trabajos de despliegue.

### Runner y Executor son diferentes

```text
Runner = quién ejecuta
Executor = cómo ocurre la ejecución
```

| Runner | Executor |
|---|---|
| Local Runner | Shell Executor |
| Host Runner | SSH Executor |
| Kubernetes Runner | Kubernetes Job Executor |
| GitOps Runner | Argo CD Executor |
| Cloud Runner | Webhook / Cloud Adapter |

Esta separación permite que Nivora soporte muchos entornos de ejecución sin reescribir la lógica central de orquestación.

### GitOps es un modo de despliegue

Nivora soporta GitOps, pero GitOps no es todo el producto.

Los futuros modos de despliegue incluyen despliegue en host, YAML de Kubernetes puro, Helm, Kustomize, GitOps con Argo CD, entrega basada en webhook y entrega específica para proveedores de nube.

### Puertos y adaptadores primero

Los sistemas externos deben integrarse a través de interfaces estables:

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

Los casos de uso principales deben depender de capacidades, no de proveedores concretos.

### Los artefactos deben ser inmutables

Un lanzamiento debe apuntar a artefactos inmutables siempre que sea posible: digest de imagen, versión inmutable, artefacto firmado y referencia SBOM. Evitar etiquetas `latest`, reconstrucciones implícitas durante el despliegue y mutación de artefactos sin seguimiento.

### La auditoría no es opcional

Las acciones importantes de entrega deben ser auditables: pipeline iniciado, trabajo asignado, artefacto seleccionado, aprobación concedida o rechazada, despliegue iniciado, reversión ejecutada, violación de política detectada, runner registrado y credencial utilizada.

Los registros de auditoría no deben contener valores secretos.

### Sin falsa preparación para producción

Nivora debe ser explícito sobre lo que existe hoy y lo que es arquitectura objetivo. Las fases tempranas no deben afirmar preparación para producción, integraciones completas, planificación duradera ni garantías de seguridad que no hayan sido implementadas y verificadas.

## Flujo de entrega de extremo a extremo

Este es el flujo a largo plazo para el que Nivora está diseñado. Las fases tempranas implementan solo el subconjunto PipelineRun basado en shell: parsing de definiciones, creación de ejecuciones encoladas, ejecución local de runner, registros, eventos, registros de auditoría, reintento, timeout, cancelación y consultas de línea temporal.

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

## Modelo de ejecución de PipelineRun

Esta es la primera fundación de ejecución que Nivora está construyendo. La implementación actual está limitada a la ejecución mínima de PipelineRun basada en shell.

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

## Modelo de estados de PipelineRun

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

## Modelo de Runner y Executor

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

## Modelo de despliegue

La ejecución de despliegue es arquitectura objetivo. No está implementada como un motor de despliegue completo de producción en la fase actual.

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

## Modelo de integración

Todos los sistemas externos deben conectarse a través de puertos y adaptadores. Los nombres de adaptadores a continuación son direcciones de integración objetivo a menos que estén explícitamente documentados como implementados.

El endpoint de solo lectura `/api/v1/integrations` expone el índice actual de capacidad de adaptadores/plugins. Es solo metadatos: no configura proveedores, llama a servicios externos ni devuelve credenciales. Los adaptadores esqueleto, noop, solo fundación y experimentales están etiquetados como tales.

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

## Modelo de observabilidad y auditoría

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

## Conceptos principales

| Concepto | Significado |
|---|---|
| Aplicación | Un producto o servicio gestionado por Nivora |
| Entorno | Un contexto de entrega como dev, staging, prod o un grupo objetivo personalizado |
| ReleaseTarget | Un objetivo de despliegue concreto como grupo de hosts, clúster Kubernetes, aplicación Argo CD, objetivo en la nube o webhook |
| Pipeline | Una definición reutilizable de etapas, trabajos y pasos |
| PipelineRun | Una ejecución de un Pipeline |
| StageRun | Registro de ejecución de una etapa |
| JobRun | Registro de ejecución de un trabajo |
| StepRun | Registro de ejecución de un paso |
| Release | Una intención de entrega versionada, generalmente vinculada a artefactos inmutables |
| DeploymentRun | Una ejecución de un lanzamiento o plan de despliegue contra un objetivo |
| Runner | Un componente que recibe y ejecuta trabajos |
| Executor | Un mecanismo usado por un Runner para ejecutar trabajo |
| Artefacto | Una salida de build como imagen, jar, binario, chart o paquete |
| Registro de artefactos | Un sistema que almacena artefactos |
| Política | Una puerta que puede permitir, denegar o requerir aprobación |
| AuditLog | Registro duradero de acciones importantes |
| Evento | Señal de ejecución emitida durante el ciclo de vida de entrega |
| LogChunk | Segmento ordenado de stdout, stderr o registro del sistema |

## Estructura del repositorio

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

| Directorio | Propósito |
|---|---|
| `cmd/` | Solo puntos de entrada de binarios |
| `internal/domain/` | Conceptos de dominio puro y estados |
| `internal/usecase/` | Orquestación de negocio |
| `internal/ports/` | Interfaces de capacidad externa |
| `internal/adapters/` | Implementaciones de puertos |
| `internal/infra/` | Infraestructura técnica |
| `internal/api/` | Transporte HTTP / gRPC |
| `api/` | Definiciones OpenAPI, AsyncAPI, proto |
| `docs/` | Arquitectura, roadmap, conceptos, docs de comunidad |
| `examples/` | Pipelines de ejemplo y especificaciones de despliegue |

## Inicio rápido

### Requisitos previos

- Go
- Make
- Docker, opcional para compose local
- PostgreSQL, opcional según el modo de ejecución

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

Documentación de empaquetado:

- [Docker Compose install](docs/operations/install-docker-compose.md)
- [Kubernetes install](docs/operations/install-kubernetes.md)
- [Configuration](docs/operations/configuration.md)
- [Performance and load testing](docs/operations/performance.md)
- [Backup and restore](docs/operations/backup-restore.md)
- [HA and disaster recovery](docs/operations/ha-disaster-recovery.md)

### Pruebas de humo

```bash
make smoke-local
make smoke-api
```

### Ejecutar servidor

```bash
make run-server
```

### Ejecutar Web UI

```bash
make run-web
```

La consola web reside en `web/` y consume las APIs existentes de ejecución, visualización, artefacto, política, evidencia, plugin y metadatos de integración. Es una fundación mínima de Phase 6.4, no un producto frontend completo.

Si el backend no es accesible, la consola ahora se detiene en una única página de diagnóstico de conexión en lugar de mostrar cada tarjeta del panel como un fallo de fetch. Iníciela con `make run-web` o ejecute Vite desde `web/` para que las dependencias se resuelvan desde el paquete web incluido.

### Comprobación de salud

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/version
curl http://localhost:8080/api/v1/system/runtime
curl http://localhost:8080/api/v1/system/diagnostics
curl http://localhost:8080/metrics
```

`/readyz` y `/api/v1/system/diagnostics` incluyen verificaciones ligeras de dependencias para la base de datos, almacén de objetos, bus de eventos, recuperación de outbox y postura de reconexión de runners.

### Ejecutar Worker

```bash
make run-worker
```

### Ejecutar Runner

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

## Desarrollo local

Nivora soporta desarrollo local a través del Makefile, docker-compose, un almacén de objetos local, un bus de eventos en memoria, el executor de shell y pipelines de ejemplo.

Este repositorio usa un proxy Go neutral por defecto en las herramientas locales:

```bash
GOPROXY=https://proxy.golang.org,direct
```

Los desarrolladores en China pueden sobrescribirlo sin cambiar los valores por defecto del proyecto:

```bash
GOPROXY=https://goproxy.cn,direct make verify
```

o:

```bash
export GOPROXY=https://goproxy.cn,direct
make verify
```

## Pipeline de ejemplo

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

Ejecútelo localmente:

```bash
go run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml
```

## Ejemplo de ejecución en seco de despliegue YAML

La fundación actual de Phase 2 soporta planificación de despliegue YAML no destructiva y validación de ejecución en seco, más apply local noop explícito para pruebas de ejecución. Renderiza manifiestos estáticos, valida su forma básica, crea un DeploymentPlan, registra el inventario de recursos, verifica las imágenes del manifiesto contra los artefactos vinculados, registra datos de registros/eventos/auditoría/línea temporal y no aplica recursos a un clúster por defecto.

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

Ejecútelo localmente:

```bash
go run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml
go run ./cmd/nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml
```

El apply local explícito requiere un comando separado y confirmación:

```bash
go run ./cmd/nivora deployment apply --local examples/deployments/yaml-apply-local.yaml --confirm
```

La ruta de apply local por defecto usa el cliente de manifiesto noop seguro. La semántica de apply de Kubernetes en producción, Helm, Kustomize, Argo CD, proveedores de nube, despliegue remoto en host e integraciones de registro siguen siendo trabajo futuro.

## Ejemplo de ejecución en seco de despliegue en host

Phase 8.1 endurece la fundación segura de despliegue en host. Puede construir un plan para desplegar un paquete binario en directorios de lanzamiento versionados, cambiar enlaces simbólicos, comprobar salud HTTP/TCP/comando, ejecutar lotes y preparar reversión protegida de enlaces simbólicos. La ejecución por defecto usa un executor de host noop y no ejecuta SSH remoto.

```bash
go run ./cmd/nivora deployment host plan --file examples/deployments/host-dry-run.yaml --local
go run ./cmd/nivora deployment host run --file examples/deployments/host-dry-run.yaml --local
```

El despliegue remoto en host permanece deshabilitado a menos que se configure explícitamente un transporte adaptador con referencias de credenciales, confirmación y flags de permiso.

## Ejemplo de lanzamiento multi-objetivo

Phase 2.7 añade una fundación local de ReleasePlan / ReleaseExecution. Puede planificar un Release a través de múltiples objetivos y ejecutar objetivos seguros secuencialmente a través de DeploymentRuns a nivel de objetivo u objetivos placeholder.

```bash
go run ./cmd/nivora release plan --file examples/releases/multi-target-release.yaml --local
go run ./cmd/nivora release deploy --file examples/releases/sequential-release.yaml --local
```

Los comandos de lanzamiento y despliegue respaldados por servidor están protegidos por RBAC. Use `--token-env NIVORA_AUTH_TOKEN` para llamadas al servidor en lugar de pasar valores de token directamente.

Esto no es un motor de flujo de trabajo de producción. La ejecución paralela, las aprobaciones duraderas, los objetivos en host/nube y la automatización GitOps de producción siguen siendo trabajo futuro.

Ejecute un PipelineRun mínimo en shell a través de la API:

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

Los grupos de API no implementados devuelven respuestas estructuradas, no datos falsos:

```json
{
  "code": "not_implemented",
  "message": "This endpoint is reserved for a future phase.",
  "path": "/api/v1/integrations"
}
```

## Eventos

Nivora usa envolturas de eventos estilo CloudEvents.

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

Las definiciones OpenAPI residen en `api/openapi/openapi.yaml`. Las definiciones AsyncAPI residen en `api/asyncapi/asyncapi.yaml`.

Los grupos de API principales incluyen:

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
/api/v1/runner-groups
/api/v1/runners
/api/v1/approvals
/api/v1/policies
/api/v1/audit-logs
/api/v1/events
/api/v1/logs
/api/v1/timeline
/api/v1/integrations
/api/v1/visualization
```

La inspección agregada de ejecución también tiene puntos de entrada CLI:

```bash
nivora events search --pipeline-run-id <pipeline-run-id> --limit 50
nivora logs search --pipeline-run-id <pipeline-run-id> --contains "error"
nivora timeline search --pipeline-run-id <pipeline-run-id> --limit 50
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

Consulte [ROADMAP.md](ROADMAP.md) y [docs/roadmap/overview.md](docs/roadmap/overview.md) para más detalles.

## Mapa de contribución

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

Antes de contribuir, lea:

- [AGENTS.md](AGENTS.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [PROJECT_CHARTER.md](PROJECT_CHARTER.md)
- [docs/README.md](docs/README.md)
- [docs/rfcs/README.md](docs/rfcs/README.md)
- [docs/architecture/architecture-contract.md](docs/architecture/architecture-contract.md)
- [docs/architecture/module-boundaries.md](docs/architecture/module-boundaries.md)
- [docs/engineering/testing-policy.md](docs/engineering/testing-policy.md)
- [docs/engineering/dependency-policy.md](docs/engineering/dependency-policy.md)

Expectativas básicas:

- mantenga los cambios pequeños
- preserve los límites de arquitectura
- no añada abstracciones especulativas
- no incluya secretos en commits
- no afirme preparación para producción
- actualice la documentación cuando la arquitectura cambie
- actualice OpenAPI / AsyncAPI cuando el comportamiento público cambie
- añada pruebas para cambios de comportamiento

## Automatización para contribuyentes

Las herramientas de codificación automatizadas y los contribuyentes humanos usan las mismas reglas del repositorio. El archivo de instrucciones canónico es [AGENTS.md](AGENTS.md).

Los archivos de instrucciones específicos de herramientas deben apuntar a `AGENTS.md` en lugar de definir comportamientos contradictorios. Todos los cambios deben preservar los límites de arquitectura, los límites de fase, la política de dependencias, la política de pruebas, la línea base de seguridad y la consistencia de la documentación.

## Verificación

Ejecute la suite completa de verificación:

```bash
make verify
```

Las comprobaciones esperadas incluyen:

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

## Seguridad

Nivora no debe incluir ni exponer secretos en commits.

No incluya en commits tokens, contraseñas, claves privadas, kubeconfigs, credenciales de nube, credenciales de registro ni credenciales falsas de aspecto realista. Los valores secretos no deben registrarse, ser devueltos por APIs normales, almacenarse en registros de auditoría, incrustarse en ejemplos o incrustarse en pruebas.

Consulte [SECURITY.md](SECURITY.md) y [docs/engineering/security-baseline.md](docs/engineering/security-baseline.md).

Phase 3.0 añade fundamentos locales de DevSecOps:

```bash
go run ./cmd/nivora security scan artifact registry.example.com/demo/app:latest --local
go run ./cmd/nivora security scan manifest examples/security/manifest-privileged-warning.yaml --local
go run ./cmd/nivora policy evaluate --subject registry.example.com/demo/app:latest
```

Estos comandos usan fundamentos de escáner noop/amigable y puertas de política integradas. Trivy, Cosign, generación SBOM, OPA, Kyverno, Gatekeeper y la automatización de seguridad en producción siguen siendo trabajo futuro.

Phase 3.1 añade metadatos de SecretRef y Credential:

```bash
go run ./cmd/nivora secret put --name local-registry-token --value-env NIVORA_TOKEN --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora secret provider validate --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora credential create --file examples/credentials/registry-credential.yaml --token-env NIVORA_AUTH_TOKEN
```

Los valores secretos se aceptan solo en los límites de creación y rotación y no son devueltos por APIs normales. Los comandos respaldados por servidor deben usar `--token-env` para que los tokens API no queden en el historial del shell; las rutas de desarrollo en proceso pueden usar `--local` donde el comando lo soporte. El proveedor integrado es solo para desarrollo. Phase 7.1 añade fundamentos de adaptador Vault y Kubernetes Secret más placeholders de KMS en la nube; el almacenamiento externo de secretos en producción sigue siendo trabajo futuro.

Phase 7.0 endurece los fundamentos locales de auth y RBAC:

```bash
go run ./cmd/nivora auth whoami
go run ./cmd/nivora auth users
go run ./cmd/nivora auth roles
go run ./cmd/nivora auth permissions
go run ./cmd/nivora project members add <project-id> --user-id <user-id> --role developer
go run ./cmd/nivora auth service-account create --name ci --role developer
go run ./cmd/nivora auth token create --subject-id <service-account-id>
```

La autenticación de desarrollo no es autenticación de producción. El modo de token estático lee valores de token desde variables de entorno. OIDC es trabajo de fundación backend configurado por proveedor; el SSO completo de navegador y las operaciones del ciclo de vida del proveedor siguen siendo trabajo futuro.

Los diagnósticos del sistema se pueden leer a través de CLI o HTTP:

```bash
go run ./cmd/nivora system runtime
go run ./cmd/nivora system diagnostics
```

Phase 7.2 añade fundamentos de multi-tenencia y cuotas:

```bash
go run ./cmd/nivora quota view --scope-type project --scope-id demo --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora usage summary --scope-type project --scope-id demo --token-env NIVORA_AUTH_TOKEN
```

Los tokens API con alcance pueden restringirse a límites de tipo org/proyecto/entorno, y los modelos de lectura de cuotas exponen fundamentos de concurrencia, runner, artefacto, almacenamiento de registros y rate-limit. La aplicación distribuida persistente de cuotas sigue siendo trabajo futuro.

Phase 7.3 añade fundamentos de auditoría de cumplimiento y evidencia:

```bash
go run ./cmd/nivora audit search --subject <subject-id>
go run ./cmd/nivora evidence list --subject-type pipelineRun --subject-id <pipeline-run-id> --token-env NIVORA_AUTH_TOKEN
go run ./cmd/nivora evidence export pipelineRun <pipeline-run-id> --format markdown --token-env NIVORA_AUTH_TOKEN
```

Los paquetes de evidencia recopilan contexto seguro de lanzamiento, artefacto, aprobación, política, seguridad, despliegue, referencia de registro, evento y auditoría. Los valores similares a secretos se redactan antes de la exportación; el almacenamiento externo inmutable de auditoría y los trabajos de aplicación de retención siguen siendo trabajo futuro.

## Documentación

| Documento | Propósito |
|---|---|
| [PROJECT_CHARTER.md](PROJECT_CHARTER.md) | Propósito y principios del proyecto |
| [ROADMAP.md](ROADMAP.md) | Roadmap de alto nivel |
| [docs/README.md](docs/README.md) | Índice de documentación |
| [docs/architecture/](docs/architecture/overview.md) | Modelo de arquitectura |
| [docs/concepts/](docs/concepts/overview.md) | Conceptos principales |
| [docs/product/](docs/product/vision.md) | Planificación de producto |
| [docs/community/](docs/community/governance.md) | Contribución y gobernanza |
| [docs/rfcs/](docs/rfcs/README.md) | Proceso RFC |
| [docs/adr/](docs/adr/0001-use-go-as-primary-language.md) | Registros de decisiones de arquitectura |
| [AGENTS.md](AGENTS.md) | Reglas de automatización y contribución |

## Estrella polar de diseño

Nivora se está construyendo para hacer que los sistemas de entrega sean más coherentes. No asume una herramienta, una nube, una ejecución o un modelo de despliegue.

El objetivo a largo plazo es proporcionar un plano de control de entrega donde:

```text
los pipelines son repetibles
los lanzamientos se basan en artefactos
los despliegues son auditables
las políticas son explícitas
los runners están aislados
las integraciones son reemplazables
los eventos son observables
la reversión es trazable
```

Nivora comienza pequeño. El primer hito no es soportar todas las herramientas. El primer hito es construir la fundación correcta.

## Licencia

Nivora está licenciado bajo la Apache License 2.0. Consulte [LICENSE](LICENSE).
