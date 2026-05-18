# Phase 3: Multi-Cloud and DevSecOps

## Objective

Expand delivery targets and security controls.

## Scope

- AWS provider Adapter.
- Aliyun provider Adapter.
- Tencent Cloud provider Adapter.
- Trivy integration.
- Cosign integration.
- SBOM support.
- Policy gates.
- OIDC.
- Advanced secret handling.

## Non-Goals

- Provider-specific architecture in the domain.
- Unreviewed privileged execution.
- Opaque security automation without audit.

## Expected Deliverables

Multi-cloud inventory and security checks integrated through Ports and Adapters.

## Acceptance Criteria

- Cloud SDKs stay inside Adapters.
- Security findings can be linked to Artifacts, Releases, or DeploymentRuns.
- Secret and credential handling follows the security model.

## Contribution Opportunities

- Cloud provider RFCs.
- Scanner Adapter design.
- Policy engine design.
- Secret provider tests.

