# Local Auth Mode

Phase 3.2 adds local auth foundations for development.

## Dev Mode

`auth.mode: dev` uses a configured local user:

```yaml
auth:
  enabled: false
  mode: dev
  dev_user: local-admin
```

When auth is disabled, Nivora still attaches a local owner subject so existing local development commands continue to work.

## Token Mode

Token mode reads the expected token from an environment variable:

```yaml
auth:
  enabled: true
  mode: token
  static_token_env: NIVORA_AUTH_TOKEN
```

Requests should use:

```bash
Authorization: Bearer $NIVORA_AUTH_TOKEN
```

Do not place token values in config files, examples, logs, audit records, or command output.

## OIDC Placeholder

`oidc-placeholder` exists only to reserve the direction. It does not implement OIDC, Keycloak, SSO, group mapping, refresh tokens, or browser login.
