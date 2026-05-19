# OIDC Auth Foundation

Phase 7.0 adds the backend foundation for enterprise identity without hardcoding a provider.

## Modes

```yaml
auth:
  enabled: true
  mode: oidc
  oidc:
    issuer: https://issuer.example
    client_id: nivora
    jwks_url: https://issuer.example/.well-known/jwks.json
    scopes:
      - openid
      - profile
      - email
    groups_claim: groups
    username_claim: preferred_username
```

`dev` and `token` modes remain available for local development and automation. OIDC provider validation is behind the auth use case provider interface; concrete provider adapters must not leak tokens or client secrets into logs, audit records, examples, or command output.

## Current Limitations

- No frontend login flow is implemented in this phase.
- No provider is hardcoded; Keycloak, Okta, Entra ID, and other providers require configuration and future adapter hardening.
- Refresh tokens, browser sessions, SSO logout, and group synchronization remain future work.
- The project remains early-stage and not production-ready.

## Secret Handling

Client secrets, token signing keys, and provider credentials must come from environment variables, SecretRef/CredentialRef records, or an external secret provider. Do not place secret values in config files.
