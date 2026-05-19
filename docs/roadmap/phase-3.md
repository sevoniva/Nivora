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
- Local AuthN/AuthZ and RBAC foundation.
- Token auth mode with token values sourced from environment variables.
- OIDC and Keycloak placeholders only.
- ApprovalRequest and ApprovalDecision foundations.
- Simple ChangeWindow evaluation.
- NotificationProvider port with noop/log-style local behavior.
- Release and deployment gates can enter WaitingApproval.
- CloudAccount, CloudProviderConfig, and CloudInventorySnapshot foundations.
- AWS, Aliyun, Tencent, and generic cloud inventory adapter skeletons.
- HostTarget, HostGroup, HostDeploymentPlan, and HostDeploymentRunDetail foundations.
- HostExecutor port with noop execution and a disabled SSH skeleton.
- Versioned release directory, symlink switch, health check, and rollback plan modeling for host targets.
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

Phase 3.0 delivers auditable security scan and policy gate foundations through Ports and Adapters. Phase 3.1 adds the minimal Secret and Credential model needed by future adapters. Phase 3.2 adds local AuthN/AuthZ and RBAC foundations. Multi-cloud inventory, Vault/KMS integrations, OIDC/Keycloak production integration, and full security integrations remain future Phase 3 work.

Phase 3.3 adds backend-only human governance foundations: approvals, change windows, notification records, and audit/event trails. It does not add frontend workflows, ITSM integration, or real external notification delivery.

Phase 3.4 adds multi-cloud inventory foundations for cloud accounts, regions, clusters, hosts, registries, and snapshots. It does not add cloud deployment or real provider SDK integration.

Phase 3.5 adds a safe host deployment foundation for VM and bare-metal targets. It supports host planning and noop/local execution only by default; remote SSH execution is guarded and not production-ready.

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

## Contribution Opportunities

- Cloud provider RFCs.
- Scanner Adapter design.
- Policy engine design.
- Secret provider tests.
- Approval and change-window policy tests.
- Cloud provider adapter RFCs and inventory model tests.
- Host executor adapter RFCs.
- Host deployment dry-run examples and tests.
