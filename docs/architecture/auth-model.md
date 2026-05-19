# Auth Model

Phase 3.2 adds a minimal authentication and RBAC foundation for local development and future integration work.

## Current Implementation

Nivora supports three auth modes:

- `dev`: creates a local development subject, usually `local-admin`.
- `token`: accepts a bearer token whose expected value is read from an environment variable.
- `oidc-placeholder`: reserved for future OIDC and Keycloak integration.

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

The Phase 3.2 middleware protects selected sensitive routes, especially Secret and Credential APIs. Future phases should expand route coverage as resource ownership and persistence mature.

## Non-Goals

- No password authentication.
- No frontend login UI.
- No production OIDC or Keycloak integration.
- No enterprise policy engine such as Casbin yet.
- No token values in logs, audit records, examples, or docs.
