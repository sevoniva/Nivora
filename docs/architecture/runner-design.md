# Runner Design

The runner is separate from the control plane. It will register with the control plane, send heartbeats, receive assigned work, execute through executor adapters, and stream logs.

Phase 0 includes only placeholder registration and heartbeat logs plus a basic shell executor. Privileged execution, container isolation, Kubernetes jobs, and remote host execution are not implemented yet.

