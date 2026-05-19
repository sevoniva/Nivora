package tencent

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
	return Provider{fake: fake.New(domaincloud.ProviderTencent)}
}

func (p Provider) Info(ctx context.Context) (domaincloud.CloudProviderInfo, error) {
	info, err := p.fake.Info(ctx)
	if err != nil {
		return domaincloud.CloudProviderInfo{}, err
	}
	info.Name = domaincloud.ProviderTencent
	info.DisplayName = "Tencent Cloud provider foundation"
	info.Status = "foundation"
	info.SDK = "none"
	info.Warnings = []string{"Tencent Cloud SDK integration is future work; baseline inventory remains deterministic"}
	return info, nil
}

func (p Provider) ValidateConfig(ctx context.Context, config domaincloud.CloudProviderConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if config.Provider != "" && config.Provider != domaincloud.ProviderTencent {
		return fmt.Errorf("tencent provider received config for %q", config.Provider)
	}
	return nil
}

func (p Provider) ValidateCredential(ctx context.Context, account domaincloud.CloudAccount) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if account.CredentialRef == "" && account.Config.CredentialRef == "" {
		return fmt.Errorf("tencent credentialRef is required for real credential validation; no secret values are accepted")
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
	snapshot.GeneratedBy = "tencent-provider-foundation"
	snapshot.Warnings = append(snapshot.Warnings, "Tencent Cloud inventory is foundation-level and does not call Tencent APIs in baseline mode")
	return snapshot, nil
}
