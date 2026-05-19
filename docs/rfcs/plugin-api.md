# RFC: Plugin API Foundation

Status: Draft foundation

## Context

Nivora needs external adapter extensibility for SCM providers, artifact registries, cloud inventory, executors, secret providers, scanners, and notifications. The extension model must keep domain and usecase packages independent from transport details and unsafe dynamic loading.

## Decision

Nivora uses manifest-described plugins and an external process protocol foundation.

Current plugin API version:

```text
v1alpha1
```

Current manifest API version:

```text
nivora.io/plugin/v1alpha1
```

Supported protocol identifiers:

- `builtin`
- `http`
- `grpc`

Required external lifecycle operations:

- health
- capabilities
- validate config
- execute

The protocol skeleton is defined in `api/proto/plugin.proto`.

## Compatibility

Each manifest declares:

- plugin API version
- supported Nivora version range
- supported transport protocols

Nivora rejects unsupported plugin API versions and unsupported protocols during manifest validation.

## Non-Goals

- No Go `plugin` dynamic loading.
- No marketplace.
- No automatic plugin installation.
- No production plugin process supervisor in this phase.
- No secret values in manifests, logs, audit, or normal API responses.

## Future Work

- Authenticated plugin transport.
- Sandboxed execution policies.
- Plugin process supervision.
- Capability-specific request schemas.
- Plugin audit envelopes.
- Cancellation, retry, timeout, and streaming log contracts.
