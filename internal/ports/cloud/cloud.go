package cloud

import "context"

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
	ValidateCredential(ctx context.Context, credential CredentialRef) error
	ListRegions(ctx context.Context) ([]string, error)
	ListClusters(ctx context.Context, region string) ([]Cluster, error)
	ListHosts(ctx context.Context, region string) ([]Host, error)
	ListRegistries(ctx context.Context, region string) ([]Registry, error)
}
