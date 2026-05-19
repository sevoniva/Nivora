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

## OIDC Mode

Phase 7.0 introduces the OIDC provider interface and config shape:

```yaml
auth:
  enabled: true
  mode: oidc
  oidc:
    issuer: https://issuer.example
    client_id: nivora
    jwks_url: https://issuer.example/.well-known/jwks.json
```

This is still a backend foundation. It does not implement a frontend login flow, refresh token lifecycle, or provider-specific setup.

## Service Accounts

Use service accounts and API tokens for automation. Tokens are hashed in storage and raw values are returned only once:

```bash
nivora auth service-account create --name ci --role developer
nivora auth token create --subject-id <service-account-id>
```
