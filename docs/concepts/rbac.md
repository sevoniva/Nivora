# RBAC

RBAC means role-based access control. Nivora uses it to decide whether an authenticated subject may perform an action on a resource.

In Phase 3.2, RBAC is intentionally small:

- roles are built in
- permissions are action strings
- decisions return allow or deny with a reason
- selected HTTP routes check permissions

This is not production SSO. OIDC, Keycloak, external groups, durable membership storage, and advanced policy engines are future work.
