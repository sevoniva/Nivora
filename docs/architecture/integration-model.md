# Integration Model

Nivora integrates with external systems through Ports and Adapters.

## Ports

- SCMProvider: Git provider operations.
- ArtifactProvider: Artifact Registry operations.
- CloudProvider: regions, clusters, hosts, and registries.
- Executor: execution mechanism.
- WorkflowRuntime: workflow lifecycle abstraction.
- SecretProvider: secret storage and retrieval.
- NotificationProvider: notifications.
- PolicyEngine: policy evaluation.
- EventBus: event publish and subscribe.
- ObjectStore: object storage.

## Adapters

Adapters implement Ports for specific systems or local mechanisms. Current Phase 0 Adapters are placeholders or local implementations, such as memory event bus, local object store, builtin secret store, shell Executor, generic SCM placeholder, and OCI artifact placeholder.

## Adding Future Integrations

Future real integrations should:

- start with a clear Port contract
- keep SDK code inside Adapter packages
- avoid importing vendor SDKs into domain or usecase packages
- include timeout, retry, error redaction, and audit considerations
- avoid network calls in unit tests
- use RFCs for large or architecture-sensitive integrations

