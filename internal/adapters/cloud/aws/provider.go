package aws

import (
	"github.com/sevoniva/nivora/internal/adapters/cloud/fake"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
)

func New() fake.Provider {
	return fake.New(domaincloud.ProviderAWS)
}
