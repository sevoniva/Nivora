# External Integration Boundary Matrix

Nivora uses ports and adapters to describe external systems. This matrix is a status document for the current built-in integration catalog; it is not a production-readiness claim.

All entries below are safe-by-default metadata entries in the built-in registry. They must not require network access in baseline tests, must not return secret values, and must not mutate external systems by default.

| Integration | Type | Maturity | Adapter Kind | Boundary | Credential Mode | Network Access | Default Mutation | Current Scope | Not Implemented |
|---|---|---|---|---|---|---|---:|---|---|
| `artifact-oci` | artifact | partial | foundation | read-only | credential_ref_only | optional | false | OCI reference parsing, digest resolution when configured, saved registry metadata, explicit insecure flag, fake/HTTP tests | Harbor/Nexus/JFrog administration, registry crawling, mandatory network resolution |
| `executor-argocd` | gitops | experimental | noop | guarded-action | credential_ref_only | optional | false | GitOps plan, deterministic Argo CD status, guarded sync model | Production Argo automation, app CRUD, default sync, credentialed remote sync in CI |
| `executor-yaml-apply` | executor | experimental | foundation | guarded-action | credential_ref_only | optional | false | Kubernetes YAML plan, safety validation, guarded noop apply/test flow | Kubernetes operator behavior, default apply, default prune/delete |
| `scm-generic` | scm | foundation | skeleton | metadata-only | credential_ref_only | none | false | Repository metadata and validation shape | Real GitHub/GitLab/Gitea clone, push, webhook integration |
| `cloud-aws` | cloud | foundation | skeleton | metadata-only | credential_ref_only | none | false | Cloud inventory model and deterministic local/provider skeleton behavior | Real AWS inventory/deployment automation |
| `cloud-aliyun` | cloud | foundation | skeleton | metadata-only | credential_ref_only | none | false | Cloud inventory model and deterministic local/provider skeleton behavior | Real Aliyun inventory/deployment automation |
| `cloud-tencent` | cloud | foundation | skeleton | metadata-only | credential_ref_only | none | false | Cloud inventory model and deterministic local/provider skeleton behavior | Real Tencent Cloud inventory/deployment automation |
| `executor-shell` | executor | partial | foundation | development-only | none | none | false | Local shell execution for foundation PipelineRun workflows with runner guardrails | OS-level sandboxing, trusted isolation for untrusted workloads |
| `secret-builtin` | secret | partial | foundation | development-only | secret_ref_only | none | false | Local development SecretRef provider and redaction foundation | Production Vault/KMS lifecycle validation |
| `notification-noop` | notification | foundation | noop | noop | none | none | false | Metadata-only notification records for tests/foundation flows | Real Slack/Feishu/DingTalk/email delivery by default |
| `policy-builtin` | policy | foundation | foundation | read-only | none | none | false | Built-in policy evaluation rules | OPA/Kyverno/Gatekeeper integration |
| `scanner-noop` | scanner | foundation | noop | noop | none | none | false | Deterministic noop security scan results and built-in manifest checks | Trivy/Cosign/SBOM production integrations |

## Mechanical Checks

The boundary metadata is generated from the built-in plugin registry and exposed through:

- `GET /api/v1/integrations`
- `nivora integrations list --local`
- `nivora integrations list --server ... --token-env ...`
- the web console MCP Safety page

Tests enforce that built-in entries:

- declare `maturity`, `adapterKind`, `boundary`, `credentialMode`, and `networkAccess`
- are `safeByDefault`
- do not advertise `defaultMutation`
- do not expose `mutatesExternalSystems`
- do not require network access in the baseline catalog
- do not include secret-looking values in integration metadata

## Operating Rules

- `credential_ref_only` and `secret_ref_only` mean references only. Secret values must stay inside the configured SecretProvider and must not appear in API, CLI, MCP, logs, events, audit, or web output.
- `optional` network access means the adapter may call a configured external endpoint only when the relevant feature is explicitly configured and tests use fake/noop paths by default.
- `guarded-action` does not mean production automation. It means the model supports a guarded request shape while default behavior remains disabled.
- `development-only` entries must not be treated as production isolation.
- Cloud providers remain inventory skeletons. Nivora does not perform cloud deployments through these entries.
