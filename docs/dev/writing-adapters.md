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

You can inspect the built-in registry locally:

```bash
nivora plugins capabilities artifact-oci --local
```

## Integration Boundary Metadata

Every built-in adapter that appears in the integration index must declare explicit boundary metadata. The integration index is used by API, CLI, web, and MCP read paths, so vague entries can make skeletons look more complete than they are.

Required metadata keys:

| Key | Allowed values | Purpose |
|---|---|---|
| `maturity` | `foundation`, `partial`, `experimental`, `placeholder` | Current implementation maturity. Do not use production language. |
| `adapterKind` | `foundation`, `noop`, `skeleton`, `fake`, `placeholder` | What kind of implementation backs the entry. |
| `boundary` | `read-only`, `metadata-only`, `plan-only`, `guarded-action`, `noop`, `development-only` | The strongest behavior the entry should advertise by default. |
| `credentialMode` | `none`, `credential_ref_only`, `secret_ref_only` | How credentials are represented. Values are references only, never secret values. |
| `networkAccess` | `none`, `optional` | Whether network calls can happen when explicitly configured. Baseline tests must not require network access. |
| `safe` | `true` | Built-in entries must remain safe by default. |
| `defaultMutation` | `false` | Built-in entries must not mutate external systems by default. |

Use `credential_ref_only` for adapters that can use credentials later through a trusted use case or adapter boundary. Use `secret_ref_only` only for secret-provider entries. Do not put credential values, registry passwords, bearer tokens, kubeconfigs, private keys, or local endpoints in metadata.

The current built-in integration boundary matrix lives in `docs/status/EXTERNAL_INTEGRATION_BOUNDARY_MATRIX.md`. Update it when adding or changing a built-in adapter.

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
