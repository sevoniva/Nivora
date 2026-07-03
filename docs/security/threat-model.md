# Security Threat Model

This threat model is the Phase 9.2 pre-GA security review baseline. It is intentionally conservative. Nivora still must not be described as production-ready until maintainers close the security checklist and release checklist for the target release.

## Security Goals

- Protect credentials, runner tokens, API tokens, kubeconfigs, registry secrets, cloud keys, SSH keys, webhook secrets, and OIDC material.
- Keep control-plane state auditable across PipelineRun, DeploymentRun, ReleaseExecution, approvals, policies, and runner activity.
- Keep runner execution isolated from control-plane authority.
- Require explicit confirmation for risky actions such as Kubernetes apply, Argo CD sync, Git push, host remote deploy, pruning, and rollback execution.
- Prevent accidental production-insecure defaults.
- Avoid leaking secrets through logs, audit records, events, examples, diagnostics, release notes, or error responses.

## Trust Boundaries

| Boundary | Trusted side | Untrusted or less-trusted side | Main risk |
| --- | --- | --- | --- |
| API client to server | Server auth/RBAC middleware and use cases | CLI, automation clients, future UI, external callers | Unauthorized mutation or data access |
| Server to database | Repository layer and migrations | Database transport/configuration | Data tampering, weak backup, migration rollback risk |
| Server to secret provider | Trusted internal use cases through `SecretProvider` | Secret backend, operator config | Secret exposure or unauthorized use |
| Server to runner | Runner protocol and token validation | Runner host/process and executed jobs | Runner compromise or malicious job logs/status |
| MCP client to control plane | MCP adapter, auth/RBAC evaluation, redaction, audit | AI clients and prompts over stdio or future remote transport | Oversharing state, leaking secrets, or turning planning into unsafe action |
| Runner to executor | Runner process | Shell, SSH, host, Kubernetes, future executors | Privilege escalation and command injection |
| Deployment plan to target | Deployment use cases and guarded adapters | Kubernetes, Argo CD, GitOps repos, hosts, cloud targets | Unsafe mutation or rollback |
| Artifact reference to registry | Artifact provider port | OCI/Harbor-compatible registries | Mutable tags, poisoned artifacts, credential leaks |
| Audit/event/log export | Redaction and evidence boundaries | Operators, external systems, future plugins | Secret leakage or audit tampering |

## Threats And Controls

### Control Plane Compromise

Threat:

- An attacker gains server process access, admin API access, or database write capability.

Impact:

- Runtime state, releases, deployments, runner registration, approvals, audit records, and credentials metadata may be modified.
- If a configured secret provider is reachable from the compromised process, secret values may be retrievable by trusted internal code paths.

Current controls:

- Auth/RBAC foundation protects selected critical APIs.
- `configs/production.example.yaml` enables auth and token mode instead of dev anonymous mode.
- Secret values are represented by `SecretRef`/`CredentialRef` metadata and are not returned by normal APIs.
- Audit records exist for important lifecycle actions.
- Guarded operations require explicit flags or request fields.

Required hardening before GA:

- Complete RBAC coverage review for all mutation APIs.
- Use production-grade external secret storage and rotate credentials after suspected compromise.
- Protect database access and backups outside the application.
- Add tamper-evident audit storage or append-only export if compliance requires it.

### Runner Compromise

Threat:

- A runner host, token, or process is compromised.

Impact:

- The attacker may claim compatible jobs, append misleading logs, update job status, observe job payloads, or execute malicious commands inside the runner environment.

Current controls:

- Runner mutation endpoints require runner tokens.
- Runner tokens are stored as hashes and raw tokens are returned only once at registration/rotation.
- Runner tokens are accepted only on runner protocol endpoints and cannot authenticate control-plane admin APIs.
- Runner claim uses identity, capabilities, labels, leases, heartbeat, max concurrency, and offline detection.
- Runner protocol does not expose the full domain model.
- Privileged execution and autoscaling are not enabled by default.

Required hardening before GA:

- Run untrusted workloads in a dedicated sandbox outside the server process.
- Scope runner groups by project/environment where configured.
- Rotate runner tokens after host compromise.
- Treat runner logs and status updates as untrusted input.

### MCP Client Misuse

Threat:

- An AI client connected through MCP asks for too much context, requests unsafe operations, or attempts to obtain secret material.

Impact:

- Oversharing control-plane state could expose operational details. If MCP gained action authority too early, an AI client could become an unsafe deployment actor.

Current controls:

- MCP is disabled by default and local stdio-only in the foundation phase.
- MCP exposes read-only resources and plan-only tools.
- Action-shaped tools return `mcp_action_not_allowed`.
- Runner tokens cannot authenticate to MCP.
- MCP output redacts secret-like keys and strings.
- MCP audit records are routed through the compliance service; PostgreSQL mode persists them through the existing hash-chained audit path.

Required hardening before remote MCP:

- Add remote MCP OAuth/OIDC design.
- Add tenant-aware scope filters to every resource URI.
- Keep apply, sync, rollback, approve/reject, token mutation, secret retrieval, and runner registration behind a separate future guarded-action design.

### Credential Leakage

Threat:

- Credentials leak through config, examples, shell history, logs, diagnostics, API responses, audit records, release notes, or command errors.

Impact:

- Git, registry, Kubernetes, Argo CD, cloud, SSH, notification, scanner, or signer access may be compromised.

Current controls:

- Secret scan runs through `./scripts/verify-no-secrets.sh`.
- Example validation blocks common secret-like literals.
- Redaction helpers cover token, password, secret, private key, kubeconfig, authorization, access key, client secret, refresh token, ID token, session, and bearer markers.
- CLI docs prefer `--value-env`, `--token-env`, `--username-env`, and `--password-env`.

Required hardening before GA:

- Review every new API response schema for secret-value fields.
- Treat any secret in logs or audit as a security bug.
- Use external secret providers for real deployments.

### Malicious Pipeline

Threat:

- A pipeline definition intentionally runs harmful shell commands, emits misleading logs, consumes excessive resources, or attempts to exfiltrate environment values.

Impact:

- Runner host compromise, data loss in runner workspace, secret exposure, or denial of service.

Current controls:

- Shell execution is a minimal foundation and runner/executor are separate from the server.
- Cancellation, timeout, retry, logs, events, and audit foundations exist.
- Runner tokens and labels provide a first scheduling boundary.

Required hardening before GA:

- Do not execute untrusted pipelines on shared privileged runners.
- Add stronger workspace isolation, resource limits, and secret-masking behavior before broad multi-tenant use.
- Keep runner hosts disposable where possible.

### Malicious Deployment Manifest

Threat:

- A manifest requests privileged containers, hostPath mounts, sensitive namespace access, destructive resources, or unsafe image references.

Impact:

- Kubernetes cluster compromise, namespace escape, workload disruption, or deployment of malicious images.

Current controls:

- Kubernetes apply is explicit and not default.
- Dry-run and planning are the default examples.
- Manifest inventory, warnings, and lightweight security checks exist.
- Policy gates can warn or deny based on configured findings.

Required hardening before GA:

- Add environment-specific policy enforcement for privileged pods, hostPath, namespace scope, resource limits, and image immutability.
- Use server-side dry-run and reviewed target restrictions before apply.
- Keep pruning and destructive rollback guarded.

### Supply Chain Artifact Issue

Threat:

- A mutable tag changes after release approval, an artifact is poisoned, or a dependency is compromised.

Impact:

- A release may deploy code different from the reviewed intent.

Current controls:

- OCI references are parsed and normalized.
- Digest-pinned artifacts are preferred.
- Mutable tags such as `latest` produce warnings.
- ReleaseArtifact can bind resolved digests where available.
- Dependency review keeps the Go dependency set small.

Required hardening before GA:

- Require digest-bound artifacts for protected environments.
- Add signature verification and SBOM validation before trusted release promotion.
- Review dependency updates and generated artifacts before release.

### Audit Tampering

Threat:

- A malicious actor modifies or deletes audit records, events, or logs after the fact.

Impact:

- Investigation, compliance, rollback, and incident response evidence may be unreliable.

Current controls:

- Audit, events, logs, timelines, and evidence bundle foundations exist.
- Important lifecycle actions emit audit records.
- SHA-256 hash-chained audit records in `compliance_audit_records` across all 9 audit-producing stores (pipeline, deployment, release, release_execution, auth, credential, security, approval, cloud).
- Verify API (`GET /api/v1/audit/verify`) validates chain integrity and detects tampering.
- Full audit hash chain coverage verified via integration tests (7 audit chain tests).
- Secret values should not appear in audit records.

Required hardening before GA:

- Store audit records in a restricted database role or append-only external sink.
- Back up audit/event/log data.
- Add tamper-evidence or signed evidence bundles if required.

### Tenant Isolation Failure

Threat:

- A user, token, runner, credential, release, deployment, or audit query crosses org/project/environment boundaries.

Impact:

- Unauthorized data exposure or mutation across teams.

Current controls:

- Domain models include org, project, environment, membership, roles, permissions, scopes, runner groups, credential scopes, and quotas foundations.
- API tokens and service accounts carry subject/scope metadata.

Required hardening before GA:

- Complete cross-tenant negative tests for critical APIs.
- Enforce scope on credentials, runners, releases, deployments, artifacts, policy results, audit, and evidence exports.
- Avoid shared runners for untrusted tenant workloads unless isolation is proven.

## Secure Defaults Review

| Area | Expected default |
| --- | --- |
| Production-shaped config auth | Enabled, token or future OIDC mode; dev anonymous mode must not be production default |
| Production runtime store | PostgreSQL; memory mode is rejected for `production` / `prod` configs |
| Local shell executor | Allowed for local/dev only; rejected by production config validation |
| Insecure OCI registry | Disabled unless explicitly configured |
| Kubernetes apply | Disabled unless explicitly guarded by request/config |
| Argo CD sync | Disabled unless explicitly guarded by request/config |
| Remote host deploy | Disabled unless explicitly guarded by request/config |
| Kubernetes apply | Disabled unless `apply=true` and confirmation/target policy allow it |
| Argo CD sync | Disabled unless `sync=true`, `allowSync=true`, confirmation, and policy allow it |
| Host remote deploy | Disabled unless apply, confirm, CredentialRef, and `allowRemoteHostDeploy` are explicit |
| Git push | Disabled unless explicitly configured and confirmed |
| Destructive rollback/prune | Disabled by default |
| Secret values | Not returned by normal APIs and not logged |
| Runner protocol | Mutation endpoints require runner token or RBAC admin action |

## Dependency Review

The current direct Go dependency set is intentionally small:

- `github.com/go-chi/chi/v5`
- `github.com/jackc/pgx/v5`
- `github.com/spf13/cobra`
- `gopkg.in/yaml.v3`

Review dependency changes before RC/GA for:

- new network clients
- cloud, Kubernetes, Argo CD, registry, scanner, signer, notification, or plugin dependencies
- transitive dependency growth
- CVE advisories
- license compatibility

## Open Risks

- Runner workload sandboxing is not a production isolation guarantee.
- Some persistence and recovery areas remain foundation-level.
- Full enterprise SSO, Vault/KMS hardening, signature verification, SBOM enforcement, and external security scanner integrations remain future hardening.
- Audit storage is not yet tamper-evident by itself.
- Production multi-tenant isolation requires continued negative testing and operational review.
