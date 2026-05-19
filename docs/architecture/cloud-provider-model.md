# Cloud Provider Model

Phase 3.4 added a multi-cloud inventory foundation for AWS, Aliyun, Tencent Cloud, and generic cloud targets. Phase 8.0 hardens provider metadata, config validation, inventory snapshots, and target binding metadata without adding cloud deployment.

## Current Scope

- Cloud account metadata.
- Provider configuration with `CredentialRef`.
- Region, cluster, host, and registry inventory records.
- Deterministic fake provider adapters for tests and local development.
- AWS, Aliyun, and Tencent adapter foundations without baseline cloud SDK requirements.
- Provider capability metadata and config validation.
- Cloud target binding metadata in inventory snapshots.

```mermaid
flowchart LR
    API["API / CLI"]
    USECASE["Cloud Inventory Use Case"]
    PORT["CloudProvider Port"]
    AWS["AWS Adapter Skeleton"]
    ALI["Aliyun Adapter Skeleton"]
    TENCENT["Tencent Adapter Skeleton"]
    FAKE["Fake Provider"]
    SNAPSHOT["CloudInventorySnapshot"]

    API --> USECASE --> PORT
    PORT --> AWS
    PORT --> ALI
    PORT --> TENCENT
    PORT --> FAKE
    USECASE --> SNAPSHOT
```

## Boundaries

Cloud providers are adapters. Domain and usecase packages do not import cloud SDKs. Credentials must be referenced through `CredentialRef` or `SecretRef`; secret values must never be stored in cloud account models.

Phase 8.0 does not deploy infrastructure, create clusters, mutate cloud resources, require real credentials in CI, or claim production-ready cloud integration.
