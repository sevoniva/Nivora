# Runner Trust Boundary

Nivora runners execute delivery work outside the server process. Treat every runner as a separate trust boundary. This project is still a beta-candidate foundation and does not provide a production container sandbox by default.

## What A Runner Can Do

- Poll for jobs it is allowed to claim.
- Execute the configured executor, including the shell executor.
- Append logs for jobs it owns.
- Update status for jobs it owns.
- Observe cancellation best-effort through the runner protocol.

## What A Runner Must Not Do

- Use its runner token for control-plane administration APIs.
- Append logs or status for jobs it did not claim.
- Receive broad secret material by default.
- Run with control-plane database or secret-provider credentials.

## Shell Executor Warning

The shell executor is not a sandbox. A shell step can run local commands with the privileges, filesystem access, environment variables, and network access available to the runner process.

For production-like environments:

- run runners on dedicated hosts, VMs, containers, or namespaces;
- avoid running runners as root unless the target executor absolutely requires it;
- use disposable runner environments for untrusted workloads;
- restrict mounted filesystems and workspace paths;
- restrict network egress where possible;
- pass only the minimum required environment variables;
- scope runner groups to projects/environments when that policy is configured;
- rotate runner tokens after host compromise or suspicious logs/status;
- treat runner logs and job status as untrusted input.

## Runner Tokens

Runner tokens are sensitive credentials. Nivora stores token hashes and returns raw runner tokens only at registration or rotation time. Operators must store raw tokens in a secret manager or runner-local secure configuration.

Runner protocol endpoints accept `X-Nivora-Runner-Token` only for heartbeat, claim, log append, and status update paths. The token is scoped to the URL runner id and validated again in the usecase before mutation.

## Cancellation

Cancellation is best-effort. The server can mark cancellation requested, and runners should observe it promptly, but Nivora does not currently guarantee termination of every child process or external operation. Use OS/container-level process supervision for stronger isolation.

## Current Limitations

- No full container sandbox is implemented in this phase.
- No production autoscaling policy is implemented.
- Runner group/project/environment restrictions are foundations and require continued hardening.
- Resource limits, seccomp/AppArmor profiles, and network policy are operator responsibilities today.
