package aliyun

import (
	"context"
	"fmt"

	"github.com/sevoniva/nivora/internal/adapters/cloud/fake"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
)

type Provider struct {
	fake fake.Provider
}

func New() Provider {
	return Provider{fake: fake.New(domaincloud.ProviderAliyun)}
}

func (p Provider) Info(ctx context.Context) (domaincloud.CloudProviderInfo, error) {
	info, err := p.fake.Info(ctx)
	if err != nil {
		return domaincloud.CloudProviderInfo{}, err
	}
	info.Name = domaincloud.ProviderAliyun
	info.DisplayName = "Aliyun provider foundation"
	info.Status = "foundation"
	info.SDK = "none"
	info.Warnings = []string{"Aliyun SDK integration is future work; baseline inventory remains deterministic"}
	return info, nil
}

func (p Provider) ValidateConfig(ctx context.Context, config domaincloud.CloudProviderConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if config.Provider != "" && config.Provider != domaincloud.ProviderAliyun {
		return fmt.Errorf("aliyun provider received config for %q", config.Provider)
	}
	return nil
}

func (p Provider) ValidateCredential(ctx context.Context, account domaincloud.CloudAccount) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if account.CredentialRef == "" && account.Config.CredentialRef == "" {
		return fmt.Errorf("aliyun credentialRef is required for real credential validation; no secret values are accepted")
	}
	return nil
}

func (p Provider) ListRegions(ctx context.Context, account domaincloud.CloudAccount) ([]domaincloud.CloudRegion, error) {
	return p.fake.ListRegions(ctx, account)
}

func (p Provider) ListClusters(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudCluster, error) {
	return p.fake.ListClusters(ctx, account, region)
}

func (p Provider) ListHosts(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudHost, error) {
	return p.fake.ListHosts(ctx, account, region)
}

func (p Provider) ListRegistries(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudRegistry, error) {
	return p.fake.ListRegistries(ctx, account, region)
}

func (p Provider) GetInventorySnapshot(ctx context.Context, account domaincloud.CloudAccount) (domaincloud.CloudInventorySnapshot, error) {
	snapshot, err := p.fake.GetInventorySnapshot(ctx, account)
	if err != nil {
		return domaincloud.CloudInventorySnapshot{}, err
	}
	snapshot.GeneratedBy = "aliyun-provider-foundation"
	snapshot.Warnings = append(snapshot.Warnings, "Aliyun inventory is foundation-level and does not call Aliyun APIs in baseline mode")
	return snapshot, nil
}
