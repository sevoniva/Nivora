# Auth Model

Phase 3.2 adds a minimal authentication and RBAC foundation for local development and future integration work. Phase 7.0 hardens that foundation with service accounts, hashed API tokens, broader permission checks, and an OIDC provider interface.

## Current Implementation

Nivora supports three auth modes:

- `dev`: creates a local development subject, usually `local-admin`.
- `token`: accepts either the configured static bearer token or a hashed API token issued to a service account.
- `oidc`: validates a bearer token through an OIDC provider port. Concrete providers are configuration-driven and must not hardcode an issuer.

Auth can still be disabled for local development. Disabled auth uses a local owner subject so existing smoke tests and local workflows remain simple.

## RBAC

The RBAC evaluator maps roles to permissions:

- `owner`
- `admin`
- `maintainer`
- `developer`
- `viewer`
- `auditor`

Permissions are explicit action strings such as `project.read`, `deployment.create`, `credential.manage`, `runner.manage`, `policy.manage`, and `audit.read`.

The middleware protects selected sensitive routes, including Secret and Credential APIs, runner administration, release/deployment mutation, policy evaluation, approvals, and audit timeline access. Future phases should continue to add explicit permissions as APIs mature.

## Service Accounts and API Tokens

Service accounts represent automation identities. API tokens are stored as hashes only, can expire, can be rotated, and can be revoked. Raw token values are returned only on creation or rotation.

## Non-Goals

- No password authentication.
- No frontend login UI.
- No full browser SSO login flow.
- No hardcoded OIDC/Keycloak provider.
- No enterprise policy engine such as Casbin yet.
- No token values in logs, audit records, examples, or docs.
