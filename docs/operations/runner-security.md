# Runner Security Operations

Nivora runners are execution-plane components. They should be operated as less-trusted infrastructure than the control-plane server.

## Recommended Baseline

- Use a dedicated runner identity and runner token per runner or runner group.
- Store runner tokens outside Git, preferably in a secret manager or runner-local protected environment file.
- Run the runner as a non-root user where practical.
- Keep runner workspaces isolated from server config, kubeconfigs, cloud credentials, and database credentials.
- Keep local shell runners away from shared production hosts unless the workload is trusted and isolated.
- Prefer short-lived, disposable runner instances for untrusted pipeline jobs.
- Restrict filesystem and network access using host/container controls.
- Rotate runner tokens after compromise, operator turnover, or unexpected job behavior.

## Token Handling

Raw runner tokens are one-time values returned during registration or rotation. They are not returned by list/get APIs and token hashes must never appear in API responses, logs, audit records, or events.

## Endpoint Boundaries

Runner tokens are valid only for:

- `POST /api/v1/runners/{id}/heartbeat`
- `POST /api/v1/runners/{id}/jobs/claim`
- `POST /api/v1/runners/{id}/jobs/{job_id}/logs`
- `POST /api/v1/runners/{id}/jobs/{job_id}/status`

Administrative runner APIs require control-plane authentication and `runner.manage` permission.

Runner groups are first-class runtime metadata for project/environment ownership, executor allow-lists, and aggregate concurrency limits. Project-scoped and environment-scoped runner group creation and runner registration are forced to the caller's scope. Job claim checks compare both the runner's scoped labels and its runner group's project/environment constraints against queued PipelineRun ownership in memory and PostgreSQL stores. PostgreSQL integration tests now cover project/environment mismatch, executor mismatch, capability-based claim, runner concurrency, and RunnerGroup concurrency. This prevents a valid runner token from claiming another project's or environment's queued job through the runner protocol. This is still a metadata guardrail, not an OS sandbox or a complete fleet-scale scheduler.

## Incident Response

If a runner is suspected compromised:

1. Revoke or rotate the runner token.
2. Mark the runner offline if it is still registered.
3. Inspect recently claimed jobs, logs, and status updates.
4. Rotate any secrets that may have been exposed to jobs on that runner.
5. Rebuild or replace the runner host before returning it to service.

## Production Status

Runner security is improving, but Nivora is not production-ready. Strong sandboxing, workload isolation, multi-tenant runner policies, and tamper-resistant logs require additional hardening.
