package cloud

import (
	domainaudit "github.com/sevoniva/nivora/internal/domain/audit"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
	"github.com/sevoniva/nivora/internal/domain/event"
)

const (
	EventCloudAccountCreated      = "devops.cloud.account.created"
	EventCloudCredentialValidated = "devops.cloud.credential.validated"
	EventCloudInventoryScanned    = "devops.cloud.inventory.scanned"
	EventCloudInventoryFailed     = "devops.cloud.inventory.failed"
)

type CreateAccountInput struct {
	Name          string                          `json:"name" yaml:"name"`
	Provider      string                          `json:"provider" yaml:"provider"`
	Config        domaincloud.CloudProviderConfig `json:"config" yaml:"config"`
	CredentialRef string                          `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Metadata      map[string]string               `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ValidationResult struct {
	AccountID string `json:"accountId"`
	Provider  string `json:"provider"`
	Valid     bool   `json:"valid"`
	Message   string `json:"message,omitempty"`
}

type Record struct {
	Account    domaincloud.CloudAccount           `json:"account,omitempty"`
	Regions    []domaincloud.CloudRegion          `json:"regions,omitempty"`
	Clusters   []domaincloud.CloudCluster         `json:"clusters,omitempty"`
	Hosts      []domaincloud.CloudHost            `json:"hosts,omitempty"`
	Registries []domaincloud.CloudRegistry        `json:"registries,omitempty"`
	Snapshot   domaincloud.CloudInventorySnapshot `json:"snapshot,omitempty"`
	Events     []event.Event                      `json:"events,omitempty"`
	Audits     []domainaudit.AuditLog             `json:"audits,omitempty"`
}
