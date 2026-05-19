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

## Incident Response

If a runner is suspected compromised:

1. Revoke or rotate the runner token.
2. Mark the runner offline if it is still registered.
3. Inspect recently claimed jobs, logs, and status updates.
4. Rotate any secrets that may have been exposed to jobs on that runner.
5. Rebuild or replace the runner host before returning it to service.

## Production Status

Runner security is improving, but Nivora is not production-ready. Strong sandboxing, workload isolation, multi-tenant runner policies, and tamper-resistant logs require additional hardening.
