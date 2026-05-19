# Writing Adapters

Adapters are Nivora's primary extension mechanism. They implement Ports while keeping vendor SDKs, network clients, credentials, and protocol details out of domain and usecase packages.

Phase 7.4 stabilizes a plugin metadata registry, but it does not replace the Ports and Adapters model.

## Before Adding an Adapter

Confirm:

- The Port already exists or the new Port has an accepted design.
- Domain packages do not need provider-specific types.
- Tests can run without external services.
- Credentials are referenced through SecretRef or CredentialRef.
- OpenAPI or AsyncAPI changes are included if public behavior changes.
- The adapter does not claim production readiness before hardening exists.

## Built-In Adapter Metadata

Built-in adapters should have plugin metadata when they expose a meaningful capability. Metadata should describe what the adapter can do, not overstate maturity.

Good capability names:

- `artifact.resolve_digest`
- `cloud.inventory`
- `executor.shell`
- `gitops.plan`
- `notification.noop`
- `security.scan_noop`

## External Plugins

External plugins are future work. The Phase 7.4 protocol skeleton is intentionally small:

- health
- capabilities
- validate config
- execute request placeholder

Do not add dynamic loading, marketplace installation, or remote execution behavior without an RFC.

For manifest authoring, see `docs/dev/plugin-authoring.md`.

## Secrets

Never put secret values in:

- plugin manifests
- adapter metadata
- logs
- audit records
- examples

Use SecretRef or CredentialRef and resolve values only inside trusted use cases or adapter boundaries.
