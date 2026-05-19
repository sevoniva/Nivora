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
	if len(snapshot.Bindings) == 0 {
		t.Fatalf("expected target binding metadata in snapshot: %#v", snapshot)
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

func TestCloudProviderInfoAndConfigValidation(t *testing.T) {
	service := newTestService()
	providers, err := service.Providers(context.Background())
	if err != nil {
		t.Fatalf("providers: %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("expected providers")
	}
	for _, provider := range providers {
		if !provider.Capabilities.InventorySnapshot || provider.Capabilities.RealCloudAPI {
			t.Fatalf("unexpected provider capabilities: %#v", provider)
		}
	}
	if _, err := service.CreateAccount(context.Background(), CreateAccountInput{Name: "wrong", Provider: domaincloud.ProviderAWS, Config: domaincloud.CloudProviderConfig{Provider: domaincloud.ProviderTencent}}); err == nil {
		t.Fatal("expected mismatched provider config to fail")
	}
}

func TestCloudCredentialErrorRedaction(t *testing.T) {
	service := NewService(NewMemoryStore(), map[string]cloud.CloudProvider{
		domaincloud.ProviderAWS: redactionProvider{},
	}, nil)
	account, err := service.CreateAccount(context.Background(), CreateAccountInput{Name: "redaction", Provider: domaincloud.ProviderAWS, CredentialRef: "cred-ref"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	_, err = service.ValidateAccount(context.Background(), account.ID)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got := err.Error(); got == "token secret access_key password" {
		t.Fatalf("error leaked secret markers: %q", got)
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

func (p testProvider) Info(ctx context.Context) (domaincloud.CloudProviderInfo, error) {
	return domaincloud.CloudProviderInfo{
		Name:        p.provider,
		DisplayName: p.provider,
		Status:      "test",
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
	}, nil
}

func (p testProvider) ValidateConfig(ctx context.Context, config domaincloud.CloudProviderConfig) error {
	if config.Provider != "" && config.Provider != p.provider {
		return assertErr("provider mismatch")
	}
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
	return domaincloud.CloudInventorySnapshot{
		ID:         "snapshot-1",
		AccountID:  account.ID,
		Provider:   p.provider,
		Regions:    regions,
		Clusters:   clusters,
		Hosts:      hosts,
		Registries: registries,
		Bindings: []domaincloud.CloudTargetBinding{{
			ID:        "binding-1",
			AccountID: account.ID,
			Provider:  p.provider,
			Region:    "region-1",
			ClusterID: "cluster-1",
		}},
	}, nil
}

type redactionProvider struct {
	testProvider
}

func (redactionProvider) Info(ctx context.Context) (domaincloud.CloudProviderInfo, error) {
	return testProvider{provider: domaincloud.ProviderAWS}.Info(ctx)
}

func (redactionProvider) ValidateConfig(ctx context.Context, config domaincloud.CloudProviderConfig) error {
	return nil
}

func (redactionProvider) ValidateCredential(ctx context.Context, account domaincloud.CloudAccount) error {
	return assertErr("token secret access_key password")
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
