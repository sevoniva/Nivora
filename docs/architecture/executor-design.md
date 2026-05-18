# Executor Design

Executors implement a small interface: prepare, run, cancel, and logs. The first concrete executor is a controlled shell executor that runs a command with arguments and an optional timeout.

Future executors may include SSH, Kubernetes Job, YAML apply, Helm, Argo CD, and webhooks. Those adapters should keep integration-specific behavior out of domain packages.

