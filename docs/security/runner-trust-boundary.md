# Runner Trust Boundary

Nivora runners execute delivery work outside the server process. Treat every runner as a separate trust boundary. This project is a hardened beta-candidate and provides config-level runner isolation profiles; it does not provide OS-level sandboxing by default. The shell executor is not a sandbox.

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

## Runner Isolation Profiles

Nivora provides config-level runner isolation profiles (`runtime.runner_isolation_profile`) that gate what execution modes are allowed per environment:

| Profile | Allowed in Production | Description |
|---|---|---|
| `local-dev` | No | Local development only. Shell executor, inherits environment. |
| `shell-hardened` | With explicit flag | Software-level isolation: workspace, env blocklist, process group cleanup. NOT an OS sandbox. |
| `container-isolated` | Yes | Runner runs inside a container. Operators must configure Docker/podman isolation. |
| `kubernetes-job` | Yes | Runner runs as a Kubernetes Job. Operators must configure pod security. |
| `external-runner` | Yes | Runner runs on an external isolated host. Operators manage isolation. |

Production validation rejects:
- `local-dev` profile
- `allow_docker_socket_mount: true`
- `allow_host_path_mount: true`
- `allow_privileged_executor: true`

## Operator Responsibilities

Nivora provides policy/config gates for runner safety. It does not provide runtime containment. Operators must:

- Deploy runners in containers, VMs, or dedicated hosts for production workloads
- Restrict filesystem, network, and capability access per deployment profile
- Never mount Docker socket or host paths in production runner containers
- Rotate runner tokens regularly

## Current Limitations

- No OS-level container sandbox (seccomp/AppArmor/gVisor) is enforced by Nivora.
- No production autoscaling policy is implemented.
- Runner group/project/environment restrictions are foundations and require continued hardening.
- Resource limits and network policy are operator responsibilities.
