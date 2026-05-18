# Domain Model

The Phase 0 domain model is intentionally small. It includes organizations, users, projects, applications, services, environments, release targets, repositories, artifact registries, credentials, pipelines, runs, releases, deployments, approvals, policies, runners, audit logs, events, and log chunks.

Domain packages use simple Go structs and status enums. They do not import HTTP handlers, database drivers, queue clients, cloud SDKs, or Kubernetes SDKs.

