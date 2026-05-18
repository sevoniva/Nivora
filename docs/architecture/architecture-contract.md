# Nivora Architecture Contract

This document defines architecture rules that must not be violated without an explicit architecture decision record.

## Core Positioning

Nivora is a DevOps delivery control plane.

It is not:

- a Jenkins clone
- an Argo CD replacement
- a Kubernetes-only tool
- a frontend-first platform
- a cloud-provider-specific product

Nivora coordinates CI/CD, GitOps, multi-cloud deployment, runners, artifacts, policies, approvals, audit, and visualization APIs.

## Primary Architecture

Nivora starts as a modular monolith with multiple binaries:

- `nivora-server`
- `nivora-worker`
- `nivora-runner`
- `nivora` CLI

Service extraction is allowed only after stable module boundaries exist.

## Layering

Allowed dependency direction:

```text
cmd
-> internal/app
-> internal/api
-> internal/usecase
-> internal/domain
```

Adapters may depend on ports.
Use cases may depend on ports.
Domain must not depend on ports, adapters, infra, API, or app packages.

## Domain Purity

The domain layer must contain:

- entities
- value objects
- statuses
- domain errors
- simple invariants

The domain layer must not contain:

- SQL
- HTTP
- JSON transport assumptions
- Kubernetes client logic
- cloud SDK logic
- Argo CD logic
- queue client logic
- object storage logic

## Ports and Adapters

Ports define external capabilities.

Examples:

- `SCMProvider`
- `ArtifactProvider`
- `CloudProvider`
- `Executor`
- `WorkflowRuntime`
- `SecretProvider`
- `NotificationProvider`
- `PolicyEngine`
- `EventBus`
- `ObjectStore`

Adapters implement ports.

Examples:

- GitHub SCM adapter
- GitLab SCM adapter
- OCI artifact adapter
- S3 object store adapter
- Argo CD executor adapter
- shell executor adapter

## Runner Separation

The control plane must not directly execute deployment tasks.

The runner is responsible for:

- receiving jobs
- executing jobs
- streaming logs
- sending heartbeats
- reporting status

The server is responsible for:

- API
- authorization
- orchestration requests
- state
- audit
- integration configuration

## Deployment Modes

GitOps is one deployment mode, not the only deployment mode.

Supported future deployment modes:

- host deployment
- Kubernetes YAML apply
- Helm
- Kustomize
- Argo CD
- cloud-provider-specific deployment
- webhook deployment

Do not design APIs that assume Kubernetes or Argo CD is always present.

## Artifact Immutability

Deployment should target immutable artifact references.

Prefer:

- digest
- immutable version
- signed artifact

Avoid:

- mutable `latest` tag
- rebuilding during deployment
- implicit artifact lookup without audit trail

## Auditability

Important operations must be auditable:

- create/update/delete project
- modify credentials
- trigger pipeline
- approve deployment
- deploy
- rollback
- change policy
- register runner

Audit records must not contain secret values.

## Non-Negotiable Rule

If a change violates this document, create or update an ADR first.
