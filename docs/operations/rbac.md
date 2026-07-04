# RBAC Operations

Phase 7.0 hardens the existing RBAC foundation around organizations, projects, environments, runners, credentials, releases, deployments, approvals, policies, and audit access.

## Roles

- `owner`
- `admin`
- `maintainer`
- `developer`
- `viewer`
- `auditor`

## Important Permissions

- `project.read`
- `project.write`
- `application.read`
- `application.write`
- `environment.read`
- `environment.write`
- `pipeline.run`
- `deployment.create`
- `deployment.approve`
- `deployment.cancel`
- `release.create`
- `credential.manage`
- `runner.manage`
- `policy.manage`
- `audit.read`

Critical mutation APIs for credentials, runners, deployments, releases, policies, approvals, and audit projections are protected by middleware. Route coverage is intentionally explicit so new APIs must choose a permission instead of silently bypassing authorization.

## Service Accounts and API Tokens

Service accounts are scoped automation identities. API tokens are stored as hashes only; raw token values are returned exactly once on creation or rotation.

```bash
nivora auth users
nivora auth roles
nivora auth permissions
nivora auth service-account create --name ci-deployer --role developer --scope-type project --scope-id demo
nivora auth token create --subject-id <service-account-id>
nivora auth token list
nivora auth token rotate <token-id>
nivora auth token revoke <token-id>
```

Use `--token-env` to read an existing admin token from an environment variable. Do not pass token values directly in shell history.

## Membership Operations

Organization, project, and environment membership APIs have matching CLI commands. These commands submit identity metadata only; they never accept token values, passwords, private keys, or kubeconfigs.

```bash
nivora org members list <org-id>
nivora org members add <org-id> --user-id <user-id> --role viewer

nivora project members list <project-id>
nivora project members add <project-id> --user-id <user-id> --role developer

nivora environment members list <environment-id>
nivora environment members add <environment-id> --user-id <user-id> --role maintainer
```

## Audit

Nivora records auth-sensitive actions such as membership changes, service account creation, token creation, token rotation, token revocation, and permission denial. Audit records must identify the action and subject, never raw token values.
