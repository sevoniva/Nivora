package cloud

import (
	"context"
	"testing"

	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
	"github.com/sevoniva/nivora/internal/ports/cloud"
)

func TestFakeCloudInventory(t *testing.T) {
	service := newTestService()
	account, err := service.CreateAccount(context.Background(), CreateAccountInput{Name: "dev-aws", Provider: domaincloud.ProviderAWS, CredentialRef: "cred-placeholder"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	result, err := service.ValidateAccount(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("validate account: %v", err)
	}
	if !result.Valid {
		t.Fatalf("validation = %#v", result)
	}
	snapshot, err := service.Inventory(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if len(snapshot.Regions) == 0 || len(snapshot.Clusters) == 0 || len(snapshot.Hosts) == 0 || len(snapshot.Registries) == 0 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestListCloudResources(t *testing.T) {
	service := newTestService()
	account, err := service.CreateAccount(context.Background(), CreateAccountInput{Name: "dev-aliyun", Provider: domaincloud.ProviderAliyun})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	clusters, err := service.Clusters(context.Background(), account.ID, "")
	if err != nil {
		t.Fatalf("clusters: %v", err)
	}
	hosts, err := service.Hosts(context.Background(), account.ID, "")
	if err != nil {
		t.Fatalf("hosts: %v", err)
	}
	registries, err := service.Registries(context.Background(), account.ID, "")
	if err != nil {
		t.Fatalf("registries: %v", err)
	}
	if len(clusters) != 1 || len(hosts) != 1 || len(registries) != 1 {
		t.Fatalf("clusters=%d hosts=%d registries=%d", len(clusters), len(hosts), len(registries))
	}
}

func newTestService() *Service {
	providers := map[string]cloud.CloudProvider{
		domaincloud.ProviderAWS:     testProvider{provider: domaincloud.ProviderAWS},
		domaincloud.ProviderAliyun:  testProvider{provider: domaincloud.ProviderAliyun},
		domaincloud.ProviderTencent: testProvider{provider: domaincloud.ProviderTencent},
		domaincloud.ProviderGeneric: testProvider{provider: domaincloud.ProviderGeneric},
	}
	return NewService(NewMemoryStore(), providers, nil)
}

type testProvider struct {
	provider string
}

func (p testProvider) ValidateCredential(ctx context.Context, account domaincloud.CloudAccount) error {
	return nil
}

func (p testProvider) ListRegions(ctx context.Context, account domaincloud.CloudAccount) ([]domaincloud.CloudRegion, error) {
	return []domaincloud.CloudRegion{{ID: "region-1", Name: "region-1", Provider: p.provider}}, nil
}

func (p testProvider) ListClusters(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudCluster, error) {
	return []domaincloud.CloudCluster{{ID: "cluster-1", Name: "cluster-1", Provider: p.provider, Region: "region-1"}}, nil
}

func (p testProvider) ListHosts(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudHost, error) {
	return []domaincloud.CloudHost{{ID: "host-1", Name: "host-1", Provider: p.provider, Region: "region-1"}}, nil
}

func (p testProvider) ListRegistries(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudRegistry, error) {
	return []domaincloud.CloudRegistry{{ID: "registry-1", Name: "registry-1", Provider: p.provider, Region: "region-1"}}, nil
}

func (p testProvider) GetInventorySnapshot(ctx context.Context, account domaincloud.CloudAccount) (domaincloud.CloudInventorySnapshot, error) {
	regions, _ := p.ListRegions(ctx, account)
	clusters, _ := p.ListClusters(ctx, account, "")
	hosts, _ := p.ListHosts(ctx, account, "")
	registries, _ := p.ListRegistries(ctx, account, "")
	return domaincloud.CloudInventorySnapshot{ID: "snapshot-1", AccountID: account.ID, Provider: p.provider, Regions: regions, Clusters: clusters, Hosts: hosts, Registries: registries}, nil
}
