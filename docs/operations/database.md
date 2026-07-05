# Database Operations

Nivora uses PostgreSQL as the target source of truth. Local demos still run with in-memory stores by default, but the runtime now has PostgreSQL-backed repository foundations for PipelineRun, DeploymentRun, release artifact binding, ReleasePlan, ReleaseExecution, catalog metadata, Pipeline definition state, repository snapshot/intelligence records, WorkflowPlan/WorkflowRun records, artifact registry catalogs, and policy catalogs.

Nivora is not production-ready.

## Migrations

Migrations live under `internal/infra/migration/`.

Current migration groups:

- `000001_init`: initial relational skeleton for projects, applications, pipelines, releases, deployments, events, audit, logs, and related concepts.
- `000002_runtime_protocol`: event outbox and runner protocol fields on the initial skeleton.
- `000003_persistence_foundation`: beta-candidate runtime tables for PipelineRun snapshots, JobRun claim state, ordered logs, events, audit, runners, outbox, and idempotency keys.
- `000004_runtime_recovery`: PipelineRun and DeploymentRun lease fields plus outbox retry metadata and recovery indexes.
- `000005_runner_fleet`: runner token metadata, capabilities, max concurrency, last seen time, and runner fleet indexes.
- `000006_performance_indexes`: query-shape indexes for list, log, event, audit, runner, lease, and outbox paths.
- `000007_deployment_release_runtime`: DeploymentRun, DeploymentPlan/resource/log/event/audit, Release, ReleaseArtifact, ReleasePlan, and ReleaseExecution runtime tables.
- `000008_compliance_audit_evidence`: compliance audit records, evidence bundles, policy results, approval decisions, and retention policy tables.
- `000009_governance_persistence`: auth, credential, security, approval, cloud, tenancy, and governance audit tables.
- `000010_catalog_persistence`: org, project, application, environment, repository, release target, and Pipeline definition catalog tables.
- `000011_policy_artifact_registry_catalog`: artifact registry, policy, and policy attachment catalog tables.
- `000012_pipeline_definition_versions`: saved Pipeline definition version bodies and current-version metadata.
- `000013_runtime_artifact_catalog`: standalone runtime artifact records and release-bound artifact indexes.
- `000014_security_scan_scope`: project and environment scope metadata for security scans and findings.
- `000015_runtime_runner_groups`: runner group metadata and group-level claim constraints.
- `000016_security_policy_results`: persisted policy result records for security and deployment/release gates.
- `000017_repository_workflow_persistence`: repository record, snapshot, and intelligence persistence for repository workflow foundations.
- `000018_workflow_plan_persistence`: stored Nivora Workflow plan records with content hashes and redacted plan JSON.
- `000019_workflow_run_persistence`: guarded WorkflowRun metadata records linked to queued PipelineRuns.

Run migrations with:

```sh
make migrate-up
```

Roll back the latest migration with:

```sh
make migrate-down
```

Set `DATABASE_URL` explicitly. Do not commit credentials.

Enable the PostgreSQL runtime store with:

```yaml
database:
  runtime_store: postgres
  url: "<set per environment>"
```

Local demos keep `runtime_store: memory` by default. Production/prod configs are rejected if they use memory mode.

## Runtime Tables

The Phase 5.1 runtime tables are prefixed with `runtime_` and use text IDs to match existing beta-candidate runtime identifiers:

- `runtime_pipeline_runs`
- `runtime_job_runs`
- `runtime_log_chunks`
- `runtime_events`
- `runtime_audit_logs`
- `runtime_runners`
- `runtime_event_outbox`
- `idempotency_keys`
- `runtime_deployment_runs`
- `runtime_deployment_logs`
- `runtime_deployment_events`
- `runtime_deployment_audit_logs`
- `runtime_deployment_resources`
- `runtime_manifest_snapshots`
- `runtime_rollback_plans`
- `runtime_releases`
- `runtime_release_artifacts`
- `runtime_release_plans`
- `runtime_release_executions`
- `runtime_release_execution_targets`
- `runtime_release_execution_events`
- `runtime_release_execution_audit_logs`
- `catalog_orgs`
- `catalog_projects`
- `catalog_applications`
- `catalog_environments`
- `catalog_repositories`
- `catalog_release_targets`
- `repository_records`
- `repository_snapshots`
- `repository_intelligence`
- `workflow_plan_records`
- `workflow_run_records`
- `pipeline_definitions`
- `catalog_artifact_registries`
- `catalog_policies`
- `catalog_policy_attachments`

## Operational Notes

- `runtime_event_outbox` is a persistence foundation. External broker publication remains future work.
- `runtime_pipeline_runs.owner_id`, `lease_expires_at`, `attempt`, and `heartbeat_at` support worker restart recovery.
- `runtime_event_outbox.retry_count`, `next_attempt_at`, and `last_error` support retriable event publication.
- `runtime_job_runs.lease_expires_at` supports recovery of assigned jobs whose runners stop heartbeating.
- `runtime_runners.token_hash` stores runner token hashes only; raw tokens are returned once by registration or rotation.
- `runtime_runners.max_concurrency` and `capabilities` support safer job claim decisions.
- `runtime_pipeline_runs.version`, `runtime_job_runs.version`, and `runtime_runners.version` provide optimistic-locking-friendly fields for later hardening.
- Log chunks are ordered by `(pipeline_run_id, sequence)`.
- Catalog records are persisted only when the server is configured with `database.runtime_store: postgres`; memory mode remains available for local development and unit tests.
- Secret values must never be stored in runtime logs, events, audit records, or idempotency request hashes.

## Not Yet Complete

- The default local server still uses in-memory stores. Set `database.runtime_store: postgres` to use the runtime PostgreSQL stores.
- DeploymentRun and ReleaseExecution persistence is a foundation: it stores durable aggregate records and query tables, but worker recovery policy and idempotency at every API boundary still need further hardening.
- Catalog persistence covers org/project/application/environment/repository/release-target metadata, Pipeline definitions, artifact registry metadata, policy definitions, and policy attachments.
- Notification state and more complete production restore drills remain future work.
- Phase 8.2 documents HA, backup, and restore procedures, but Nivora still does not automate them or claim production readiness.
