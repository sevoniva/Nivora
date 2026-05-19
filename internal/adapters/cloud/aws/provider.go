package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/sevoniva/nivora/internal/adapters/cloud/fake"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
)

type Provider struct {
	fake fake.Provider
}

func New() Provider {
	return Provider{fake: fake.New(domaincloud.ProviderAWS)}
}

func (p Provider) Info(ctx context.Context) (domaincloud.CloudProviderInfo, error) {
	info, err := p.fake.Info(ctx)
	if err != nil {
		return domaincloud.CloudProviderInfo{}, err
	}
	info.Name = domaincloud.ProviderAWS
	info.DisplayName = "AWS provider foundation"
	info.Status = "foundation"
	info.SDK = "none"
	info.Warnings = []string{"AWS SDK integration is not enabled in Phase 8.0; inventory uses a deterministic foundation unless a future adapter is configured"}
	return info, nil
}

func (p Provider) ValidateConfig(ctx context.Context, config domaincloud.CloudProviderConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if config.Provider != "" && config.Provider != domaincloud.ProviderAWS {
		return fmt.Errorf("aws provider received config for %q", config.Provider)
	}
	return nil
}

func (p Provider) ValidateCredential(ctx context.Context, account domaincloud.CloudAccount) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if account.CredentialRef == "" && account.Config.CredentialRef == "" {
		return fmt.Errorf("aws credentialRef is required for real credential validation; no secret values are accepted")
	}
	return nil
}

func (p Provider) ListRegions(ctx context.Context, account domaincloud.CloudAccount) ([]domaincloud.CloudRegion, error) {
	if len(account.Config.Regions) > 0 {
		regions := make([]domaincloud.CloudRegion, 0, len(account.Config.Regions))
		for _, region := range account.Config.Regions {
			region = strings.TrimSpace(region)
			if region == "" {
				continue
			}
			regions = append(regions, domaincloud.CloudRegion{ID: region, Name: region, Provider: domaincloud.ProviderAWS})
		}
		return regions, ctx.Err()
	}
	return []domaincloud.CloudRegion{
		{ID: "us-east-1", Name: "us-east-1", Provider: domaincloud.ProviderAWS},
		{ID: "us-west-2", Name: "us-west-2", Provider: domaincloud.ProviderAWS},
	}, ctx.Err()
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
	regions, err := p.ListRegions(ctx, account)
	if err != nil {
		return domaincloud.CloudInventorySnapshot{}, err
	}
	snapshot.Provider = domaincloud.ProviderAWS
	snapshot.Regions = regions
	snapshot.GeneratedBy = "aws-provider-foundation"
	snapshot.Warnings = append(snapshot.Warnings, "AWS inventory is foundation-level and does not call AWS APIs in baseline mode")
	return snapshot, nil
}
