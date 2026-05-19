package cloud

import (
	"context"

	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
)

type CredentialRef struct {
	ID        string
	SecretKey string
}

type Cluster struct {
	ID     string
	Name   string
	Region string
}

type Host struct {
	ID     string
	Name   string
	Region string
}

type Registry struct {
	ID     string
	Name   string
	Region string
}

type CloudProvider interface {
	ValidateCredential(ctx context.Context, account domaincloud.CloudAccount) error
	ListRegions(ctx context.Context, account domaincloud.CloudAccount) ([]domaincloud.CloudRegion, error)
	ListClusters(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudCluster, error)
	ListHosts(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudHost, error)
	ListRegistries(ctx context.Context, account domaincloud.CloudAccount, region string) ([]domaincloud.CloudRegistry, error)
	GetInventorySnapshot(ctx context.Context, account domaincloud.CloudAccount) (domaincloud.CloudInventorySnapshot, error)
}
