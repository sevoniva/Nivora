# Plugin System Foundation

Phase 7.4 stabilizes how Nivora describes extension points without enabling unsafe dynamic loading.

Nivora does not use Go `plugin` loading as the primary extension mechanism. Built-in adapters remain ordinary Go packages behind Ports. Future external plugins should run as separately configured processes and communicate through an explicit HTTP or gRPC protocol.

## Model

A plugin manifest describes:

- `name`
- `type`
- `version`
- `protocol`
- `endpoint`
- `capabilities`
- `compatibility`
- `lifecycle`
- `status`
- `metadata`

The current manifest API version is `nivora.io/plugin/v1alpha1`. The current plugin API version is `v1alpha1`.

Supported plugin types:

- `scm`
- `artifact`
- `cloud`
- `executor`
- `secret`
- `notification`
- `policy`
- `scanner`
- `gitops`

## Capability Registry

The registry lists built-in adapter metadata and capabilities. It is static and in-process for built-ins.

Examples:

- `executor-shell` exposes shell execution capability for minimal PipelineRuns.
- `artifact-oci` exposes artifact inspection and digest resolution capabilities.
- `executor-argocd` exposes GitOps planning and guarded Argo CD status/sync modeling.
- `cloud-aws`, `cloud-aliyun`, and `cloud-tencent` expose inventory skeleton capabilities.
- `notification-noop`, `scanner-noop`, and `secret-builtin` expose safe local foundations.

The registry validates manifest shape and compatibility. It does not load code, install packages, download artifacts, or execute untrusted binaries.

## External Protocol Direction

The protocol skeleton lives at `api/proto/plugin.proto`.

Required future external plugin operations:

- `Health`
- `Capabilities`
- `ValidateConfig`
- `Execute`

The `Execute` method is only a placeholder shape. Before any external plugin can execute delivery behavior, the project needs an RFC covering authentication, authorization, sandboxing, version compatibility, audit, timeouts, retries, cancellation, log handling, and secret access.

## Safety Rules

- No unsafe dynamic loading.
- No marketplace behavior in core.
- No credentials in plugin manifests.
- Plugin endpoints must be explicit configuration, not hardcoded local defaults.
- Domain models must not import plugin client implementations.
- Use cases should depend on stable Ports, not plugin transport details.

## Current API

- `GET /api/v1/plugins`
- `GET /api/v1/plugins/{name}`
- `GET /api/v1/plugins/{name}/capabilities`
- `POST /api/v1/plugins/validate`

These endpoints expose metadata only.
