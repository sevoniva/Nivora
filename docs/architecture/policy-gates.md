# Policy Gates

Phase 3.0 introduces policy gates as explicit backend runtime concepts. A gate evaluates scan results and delivery context, then returns one of:

- `allow`
- `warn`
- `deny`
- `require_approval`

The default local runtime uses a noop scanner and simple built-in rules so tests do not require Trivy, Cosign, external registries, Kubernetes clusters, or cloud services.

## Flow

```text
artifact / manifest / deployment plan
-> security scan request
-> SecurityScanner port
-> scanner adapter
-> SecurityScan
-> built-in policy evaluation
-> PolicyResult
-> gate decision
-> events / audit / timeline
```

## Built-In Rules

Phase 3.0 keeps rules intentionally small:

- deny when Critical findings meet the configured threshold
- warn when High findings meet the configured threshold
- warn when an artifact uses a mutable tag such as `latest`
- deny when digest is required but the artifact is not digest-pinned
- warn for simple manifest risks such as `privileged: true`, `hostPath`, or `imagePullPolicy: Always` with `latest`

These rules are not a replacement for OPA, Kyverno, Gatekeeper, Trivy, Cosign, or a production security platform.

## Boundaries

Scanner integrations live behind adapters. Use cases depend on the `SecurityScanner` port. Domain models define scan and finding concepts, but never import scanner implementations.
