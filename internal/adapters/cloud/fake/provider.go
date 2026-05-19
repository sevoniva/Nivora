package fake

import (
	"context"
	"time"

	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
)

type Provider struct {
	Name string
}

func New(provider string) Provider {
	return Provider{Name: provider}
}

func (p Provider) Info(ctx context.Context) (domaincloud.CloudProviderInfo, error) {
	select {
	case <-ctx.Done():
		return domaincloud.CloudProviderInfo{}, ctx.Err()
	default:
	}
	provider := p.provider(domaincloud.CloudAccount{})
	return domaincloud.CloudProviderInfo{
		Name:        provider,
		DisplayName: provider + " fake provider",
		Status:      "available",
		Capabilities: domaincloud.CloudProviderCapabilities{
			CredentialValidation: true,
			Regions:              true,
			Clusters:             true,
			Hosts:                true,
			Registries:           true,
			InventorySnapshot:    true,
			TargetBinding:        true,
			RealCloudAPI:         false,
		},
		SDK:      "none",
		Warnings: []string{"deterministic fake provider; no cloud API calls are made"},
	}, nil
}

func (p Provider) ValidateConfig(ctx context.Context, config domaincloud.CloudProviderConfig) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (p Provider) ValidateCredential(ctx context.Context, account domaincloud.CloudAccount) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (p Provider) ListRegions(ctx context.Context, account domaincloud.CloudAccount) ([]domaincloud.CloudRegion, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	provider := p.provider(account)
	return []domaincloud.CloudRegion{
		{ID: provider + "-region-1", Name: provider + "-region-1", Provider: provider},
		{ID: provider + "-region-2", Name: provider + "-region-2", Provider: provider},
	}, nil
}

func (p Provider) ListClusters(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudCluster, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	provider := p.provider(account)
	region = p.region(region, provider)
	return []domaincloud.CloudCluster{{ID: provider + "-cluster-1", Name: provider + "-dev-cluster", Provider: provider, Region: region, Type: "kubernetes", Status: "unknown"}}, nil
}

func (p Provider) ListHosts(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudHost, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	provider := p.provider(account)
	region = p.region(region, provider)
	return []domaincloud.CloudHost{{ID: provider + "-host-1", Name: provider + "-example-host", Provider: provider, Region: region, Type: "vm", Status: "unknown", Labels: map[string]string{"source": "fake"}}}, nil
}

func (p Provider) ListRegistries(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudRegistry, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	provider := p.provider(account)
	region = p.region(region, provider)
	return []domaincloud.CloudRegistry{{ID: provider + "-registry-1", Name: provider + "-example-registry", Provider: provider, Region: region, Type: "oci"}}, nil
}

func (p Provider) GetInventorySnapshot(ctx context.Context, account domaincloud.CloudAccount) (domaincloud.CloudInventorySnapshot, error) {
	regions, err := p.ListRegions(ctx, account)
	if err != nil {
		return domaincloud.CloudInventorySnapshot{}, err
	}
	var clusters []domaincloud.CloudCluster
	var hosts []domaincloud.CloudHost
	var registries []domaincloud.CloudRegistry
	for _, region := range regions {
		items, err := p.ListClusters(ctx, account, region.ID)
		if err != nil {
			return domaincloud.CloudInventorySnapshot{}, err
		}
		clusters = append(clusters, items...)
		hostItems, err := p.ListHosts(ctx, account, region.ID)
		if err != nil {
			return domaincloud.CloudInventorySnapshot{}, err
		}
		hosts = append(hosts, hostItems...)
		registryItems, err := p.ListRegistries(ctx, account, region.ID)
		if err != nil {
			return domaincloud.CloudInventorySnapshot{}, err
		}
		registries = append(registries, registryItems...)
	}
	return domaincloud.CloudInventorySnapshot{ID: "snapshot-" + account.ID, AccountID: account.ID, Provider: p.provider(account), Regions: regions, Clusters: clusters, Hosts: hosts, Registries: registries, ScannedAt: time.Now(), GeneratedBy: "fake-cloud-provider", Bindings: bindings(account, clusters, hosts, registries)}, nil
}

func (p Provider) provider(account domaincloud.CloudAccount) string {
	if account.Provider != "" {
		return account.Provider
	}
	if p.Name != "" {
		return p.Name
	}
	return domaincloud.ProviderGeneric
}

func bindings(account domaincloud.CloudAccount, clusters []domaincloud.CloudCluster, hosts []domaincloud.CloudHost, registries []domaincloud.CloudRegistry) []domaincloud.CloudTargetBinding {
	if len(clusters) == 0 && len(hosts) == 0 && len(registries) == 0 {
		return nil
	}
	binding := domaincloud.CloudTargetBinding{
		ID:        "binding-" + account.ID,
		AccountID: account.ID,
		Provider:  account.Provider,
		CreatedAt: time.Now(),
	}
	if len(clusters) > 0 {
		binding.Region = clusters[0].Region
		binding.ClusterID = clusters[0].ID
		binding.TargetType = "kubernetes"
	}
	if len(hosts) > 0 {
		binding.HostID = hosts[0].ID
	}
	if len(registries) > 0 {
		binding.RegistryID = registries[0].ID
	}
	return []domaincloud.CloudTargetBinding{binding}
}

func (p Provider) region(region string, provider string) string {
	if region != "" {
		return region
	}
	return provider + "-region-1"
}
