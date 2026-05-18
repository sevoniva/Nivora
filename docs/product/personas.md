# Personas

## Platform Engineer

Responsibilities: build and operate internal delivery platforms, define shared workflows, and reduce tool fragmentation.

Pain points: teams use inconsistent pipelines, deployment models, approvals, and audit patterns.

How Nivora should help: provide shared concepts, extension points, and a control plane that can coordinate multiple delivery modes.

What Nivora should not force: one CI vendor, one deployment tool, or Kubernetes-only delivery.

## DevOps Engineer

Responsibilities: maintain CI/CD workflows, runners, release automation, and environment promotion.

Pain points: pipeline state, artifact identity, deployment history, and rollback context are spread across tools.

How Nivora should help: connect PipelineRuns, Artifacts, Releases, DeploymentRuns, logs, and events.

What Nivora should not force: replacing existing Git providers, Artifact Registries, or release tools before adapters are ready.

## SRE

Responsibilities: reliability, release safety, rollback readiness, incident response, and operational visibility.

Pain points: release audit is incomplete and deployment events are hard to correlate with incidents.

How Nivora should help: preserve audit records, deployment timelines, runner heartbeats, and future visualization APIs.

What Nivora should not force: opaque deployment automation that hides target-specific state.

## Backend Team

Responsibilities: ship services safely and understand what is running in each environment.

Pain points: teams need simple delivery workflows without learning every platform detail.

How Nivora should help: expose clear Pipeline, Release, DeploymentRun, Environment, and Artifact concepts.

What Nivora should not force: frontend-first workflows or vendor-specific deployment assumptions.

## Infrastructure Architect

Responsibilities: define delivery architecture across teams, tools, clouds, and compliance needs.

Pain points: hard boundaries between CI, CD, GitOps, cloud, and audit systems create inconsistent governance.

How Nivora should help: provide a modular monolith foundation with Ports and Adapters that can evolve carefully.

What Nivora should not force: premature microservices or one cloud provider.

## Security Reviewer

Responsibilities: review secret handling, policy gates, access control, audit trails, and runner trust boundaries.

Pain points: credentials and approvals are often scattered across CI variables, scripts, and chat approvals.

How Nivora should help: model SecretRefs, Credentials, PolicyResults, AuditLogs, and runner boundaries explicitly.

What Nivora should not force: storing raw secrets in the core database or logging sensitive values.

## Open-Source Contributor

Responsibilities: improve docs, examples, tests, CLI behavior, adapters, and future core features.

Pain points: unclear project direction and hidden architecture constraints make contribution risky.

How Nivora should help: provide clear docs, ADRs, RFCs, guardrails, and phase-based contribution areas.

What Nivora should not force: broad rewrites or unclear ownership boundaries.

