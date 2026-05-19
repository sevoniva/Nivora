# Phase 3: Multi-Cloud and DevSecOps

## Objective

Expand delivery targets and security controls. Phase 3.0 starts with DevSecOps foundations rather than production-grade integrations.

## Scope

- SecurityScan and SecurityFinding models.
- SecurityScanner port.
- Noop and fake scanner adapters.
- Built-in policy gate decisions.
- SignatureCheck and SBOMRef foundations.
- Optional future Trivy integration design.
- Optional future Cosign integration design.
- SecretRef, Credential, SecretUsage, and SecretProvider foundation.
- Builtin development secret provider.
- Secret redaction and audit rules.
- External secret provider foundations for Vault, Kubernetes Secrets, and AWS/Aliyun/Tencent KMS placeholders.
- Secret rotation model, provider validation, and simple usage policy metadata.
- Local AuthN/AuthZ and RBAC foundation.
- Token auth mode with token values sourced from environment variables.
- OIDC provider interface and configuration foundation.
- Service accounts and hashed API token lifecycle foundation.
- Multi-tenancy foundations for scoped subjects, tenant quota read models, usage summaries, and runner/credential scope checks.
- Compliance audit and evidence foundations for audit search, evidence bundles, and retention policy metadata.
- ApprovalRequest and ApprovalDecision foundations.
- Simple ChangeWindow evaluation.
- NotificationProvider port with noop/log-style local behavior.
- Release and deployment gates can enter WaitingApproval.
- CloudAccount, CloudProviderConfig, and CloudInventorySnapshot foundations.
- AWS, Aliyun, Tencent, and generic cloud inventory adapter skeletons.
- HostTarget, HostGroup, HostDeploymentPlan, and HostDeploymentRunDetail foundations.
- HostExecutor port with noop execution and a disabled SSH skeleton.
- Versioned release directory, symlink switch, health check, and rollback plan modeling for host targets.
- Runner protocol foundation with job claim leases, log append, status update, cancellation request, and event outbox records.
- Minimal database migration shape for future durable PipelineRun, JobRun, runner, log, and event outbox repositories.
- AWS provider Adapter.
- Aliyun provider Adapter.
- Tencent Cloud provider Adapter.
- OIDC.
- Advanced secret handling.

## Non-Goals

- Provider-specific architecture in the domain.
- Unreviewed privileged execution.
- Opaque security automation without audit.
- Requiring Trivy, Cosign, external registries, or cloud access in CI.
- Production-grade security platform claims.

## Expected Deliverables

Phase 3.0 delivers auditable security scan and policy gate foundations through Ports and Adapters. Phase 3.1 adds the minimal Secret and Credential model needed by future adapters. Phase 3.2 adds local AuthN/AuthZ and RBAC foundations. Phase 7.1 adds external secret provider foundations and rotation metadata without making Vault, Kubernetes, or cloud KMS required for CI. Production Vault/KMS integrations, OIDC/Keycloak production integration, and full security integrations remain future work.

Phase 3.3 adds backend-only human governance foundations: approvals, change windows, notification records, and audit/event trails. Phase 6.3 hardens those foundations with scoped approval policy metadata, pending/approved/rejected/expired/canceled lifecycle behavior, timezone-aware change-window evaluation, and guarded notification adapters. Phase 7.2 adds the first multi-tenancy and quota foundation for scoped org/project/environment operations. Phase 7.3 adds compliance-oriented audit search, evidence bundles, and retention metadata. It does not add frontend workflows, ITSM integration, or real external notification delivery by default.

Phase 3.4 adds multi-cloud inventory foundations for cloud accounts, regions, clusters, hosts, registries, and snapshots. Phase 8.0 hardens provider capability metadata, config validation, credential-ref handling, inventory snapshots, and target binding metadata. It does not add cloud deployment or require real provider SDK integration in CI.

Phase 3.5 adds a safe host deployment foundation for VM and bare-metal targets. It supports host planning and noop/local execution only by default; remote SSH execution is guarded and not production-ready.

Phase 3.6 hardens the runtime boundary between server, worker, and runner. It adds a compact HTTP runner protocol, in-memory claim/lease behavior, outbox persistence shape, and worker outbox publishing without adding a heavy workflow engine.

## Acceptance Criteria

- Cloud SDKs stay inside Adapters.
- Security findings can be linked to Artifacts, Releases, or DeploymentRuns.
- Noop/fake scanners allow deterministic tests without external tools.
- Policy gate decisions can allow, warn, deny, or require approval.
- Secret and credential handling follows the security model.
- Approval decisions and change-window evaluations are auditable.
- Notification delivery remains adapter-driven and external sends are not required in tests.
- Cloud inventory can be queried through fake provider adapters without credentials.
- Host deployment plans can be generated without remote execution.
- Remote host deployment remains disabled unless explicit confirmation, credential references, and allow flags are present.
- Runners can register, heartbeat, claim a job, append logs, update job status, and observe cancel-request state in the local runtime.
- Event outbox records can be listed and marked published by the worker.

## Contribution Opportunities

- Cloud provider RFCs.
- Scanner Adapter design.
- Policy engine design.
- Secret provider tests.
- Approval and change-window policy tests.
- Cloud provider adapter RFCs and inventory model tests.
- Host executor adapter RFCs.
- Host deployment dry-run examples and tests.
