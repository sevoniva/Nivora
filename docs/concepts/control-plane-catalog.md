# Control Plane Catalog

Nivora now has a foundation catalog for the core enterprise objects that other delivery workflows refer to:

- organizations
- projects
- applications
- environments

These APIs turn the control-plane hierarchy into addressable backend resources. They are still a foundation layer, not a complete enterprise tenant lifecycle.

## Hierarchy

```text
Organization
  Project
    Application
    Environment
```

Projects require an existing organization. Applications and environments require an existing project.

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

`DELETE` disables the resource by setting `enabled=false`. It does not hard-delete state.

## CLI Shape

The CLI mirrors the API:

```bash
nivora org create --name platform
nivora project create --org-id org-123 --name delivery
nivora application create --project-id project-123 --name api
nivora environment create --project-id project-123 --name prod
```

List, get, update, and disable commands are available for each resource group.

## Current Limits

- The default runtime wiring uses an in-memory catalog store.
- PostgreSQL catalog persistence is future work.
- Fine-grained tenant lifecycle workflows are future work.
- RBAC protects the routes, but full enterprise SSO remains future work.
- Nivora is still a beta-candidate foundation, not production-ready.
