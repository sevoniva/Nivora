# Cloud Provider

A Cloud Provider represents an external cloud account or platform that Nivora can inspect through a provider adapter.

## Concepts

- `CloudAccount`: metadata for one cloud account.
- `CloudProviderConfig`: provider-specific configuration without secret values.
- `CloudRegion`: a cloud region or location.
- `CloudCluster`: a Kubernetes or managed cluster discovered from inventory.
- `CloudHost`: a VM or host-like compute resource.
- `CloudRegistry`: an artifact registry discovered from inventory.
- `CloudInventorySnapshot`: one inventory scan result.

## Current Limitations

Phase 3.4 is inventory-only. It does not perform cloud deployments, create resources, update resources, or require real cloud credentials in tests.
