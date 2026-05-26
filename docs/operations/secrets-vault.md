# Vault Secret Provider

Phase 7.1 adds a Vault provider foundation for Nivora's `SecretProvider` port. The current adapter models configuration and provider validation, but it does not require Vault in CI and does not claim production-ready Vault storage.

## Current Behavior

- The builtin provider remains the default development provider.
- The Vault adapter exposes provider identity, mount metadata, and capability status.
- Secret values are never returned through normal APIs.
- Vault-specific values such as tokens must come from future SecretRef/CredentialRef or runtime configuration, never committed files.

## Validation

Provider validation reports whether a Vault address was configured. It does not contact a live Vault server yet, so it is safe for default tests and local development without Vault.

## Future Work

- Vault token and AppRole configuration through secure references.
- Real read/write/rotate/delete support.
- Lease and renewal handling.
- Provider-specific integration tests that are optional and never required by baseline CI.

Nivora remains a hardened beta-candidate foundation and is not production-ready.
