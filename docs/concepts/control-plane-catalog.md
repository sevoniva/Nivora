# Control Plane Catalog

Nivora now has a foundation catalog for the core enterprise objects that other delivery workflows refer to:

- organizations
- projects
- applications
- environments
- SCM repositories

These APIs turn the control-plane hierarchy into addressable backend resources. They are still a foundation layer, not a complete enterprise tenant lifecycle.

## Hierarchy

```text
Organization
  Project
    Application
    Environment
    Repository
```

Projects require an existing organization. Applications, environments, and repositories require an existing project.

## API Shape

The catalog exposes basic CRUD-style routes:

- `GET /api/v1/orgs`
- `POST /api/v1/orgs`
- `GET /api/v1/orgs/{id}`
- `PATCH /api/v1/orgs/{id}`
- `DELETE /api/v1/orgs/{id}`

The same shape exists for:

- `/api/v1/projects`
- `/api/v1/applications`
- `/api/v1/environments`
- `/api/v1/repositories`

`DELETE` disables the resource by setting `enabled=false`. It does not hard-delete state.

Repository records also expose:

- `POST /api/v1/repositories/{id}/validate`

This validates metadata shape, enabled state, supported provider names, URL shape, and branch presence. It does not contact GitHub, GitLab, Gitea, or other SCM services, and it does not resolve or return CredentialRef values.

## CLI Shape

The CLI mirrors the API:

```bash
nivora org create --name platform
nivora project create --org-id org-123 --name delivery
nivora application create --project-id project-123 --name api
nivora environment create --project-id project-123 --name prod
nivora repository create --project-id project-123 --name service --url https://example.com/team/service.git
nivora repository validate repo-123 --token-env NIVORA_AUTH_TOKEN
```

List, get, update, and disable commands are available for each resource group.

## Current Limits

- The default runtime wiring uses an in-memory catalog store.
- PostgreSQL catalog persistence is available when `database.runtime_store: postgres` is configured.
- Fine-grained tenant lifecycle workflows are future work.
- Repository catalog entries store metadata and CredentialRef ids only; validation does not call GitHub, GitLab, Gitea, or other SCM APIs.
- RBAC protects the routes, but full enterprise SSO remains future work.
- Nivora is still a beta-candidate foundation, not production-ready.
