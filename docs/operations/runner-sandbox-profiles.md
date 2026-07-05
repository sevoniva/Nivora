# Runner Sandbox Profiles

Nivora provides config-level runner isolation profiles. The shell executor is NOT an OS-level sandbox. Container isolation depends on host runtime configuration.

## Profiles

| Profile | Config Value | Production | Description |
|---|---|---|---|
| Local Dev | `local-dev` | ❌ Rejected | Shell executor, inherits host environment. Dev only. |
| Shell Hardened | `shell-hardened` | ❌ Rejected | Software isolation for controlled non-production use: workspace, env blocklist, process group. |
| Container Isolated | `container-isolated` | ✅ Preferred | Runner in container. Operator must configure security. |
| Kubernetes Job | `kubernetes-job` | ✅ | Runner as K8s Job. Operator must configure PodSecurity. |
| External Required | `external-required` | ✅ | Preferred wording for a runner on a dedicated isolated host or fleet. Operator manages isolation. |
| External Runner | `external-runner` | ✅ | Backward-compatible alias for `external-required`. |

## Container Isolated Profile

The container executor (`internal/adapters/executor/container/`) provides a safe skeleton for container-based runner isolation. The current implementation enforces the same safety rules a real Docker/podman adapter would, without requiring Docker in tests.

### Safety Gates (enforced at config and runtime)

| Gate | Default | Production Requirement |
|---|---|---|
| `allow_privileged_executor` | false | Must be false |
| `allow_docker_socket_mount` | false | Must be false |
| `allow_host_path_mount` | false | Must be false |
| `allow_host_network` | N/A (rejected at executor level) | Must be false |
| Default image | `alpine:3.20` | Operator must pin to specific digest |
| Max output | 10MB | Configurable |
| Max timeout | 3600s | Configurable |

### Production Config

```yaml
runtime:
  runner_isolation_profile: container-isolated
  allow_local_shell_executor: false
  allow_privileged_executor: false
  allow_docker_socket_mount: false
  allow_host_path_mount: false
```

### Operator Requirements for Real Container Isolation

The current container executor skeleton runs commands locally. For real container isolation, operators must:

1. Deploy runners inside containers with restricted security context
2. Use read-only root filesystem (`--read-only`)
3. Drop all capabilities (`--cap-drop=ALL`)
4. Add only required capabilities (never `--privileged`)
5. Never mount Docker socket (`-v /var/run/docker.sock`)
6. Use seccomp/AppArmor profiles to restrict syscalls
7. Set memory/CPU limits
8. Use non-root user
9. Pin container images by digest
10. Mount workspace as tmpfs or restricted volume

Example safe Docker run (operator responsibility):

```bash
docker run --rm \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges \
  --memory=512m \
  --cpus=1 \
  --user=65532:65532 \
  --tmpfs=/workspace:rw,noexec,nosuid,size=1g \
  --network=none \
  sevoniva/nivora-runner:0.9.0-rc.1 \
  runner --config /etc/nivora/runner.yaml
```

## What Nivora Does vs What Operators Do

| Concern | Nivora Provides | Operator Provides |
|---|---|---|
| Profile selection gate | Config validation rejects unsafe profiles in production | Chooses profile per environment |
| Privileged execution | Rejected at config and executor level | Must not enable in container runtime |
| Docker socket | Config gate rejects in production | Must not mount in container runtime |
| Host paths | Config gate rejects in production | Must not mount in container runtime |
| Container image | Default `alpine:3.20` placeholder | Must pin to specific digest |
| Resource limits | Max output, max timeout | CPU/memory/disk limits |
| Syscall filtering | Not implemented | seccomp/AppArmor profiles |
| Network isolation | Not implemented | Container network policy |

## Current Limitations

- Container executor uses local execution skeleton; real Docker/podman adapter is future work.
- No seccomp/AppArmor/syscall filtering.
- No network egress policy.
- No image pull policy or digest enforcement.
- No Kubernetes PodSecurityPolicy or SecurityContext integration.
- The shell executor remains NOT an OS-level sandbox regardless of profile.

## See Also

- `docs/security/runner-trust-boundary.md` — Runner security boundaries
- `docs/architecture/runner-design.md` — Runner architecture
- `internal/adapters/executor/container/` — Container executor implementation
