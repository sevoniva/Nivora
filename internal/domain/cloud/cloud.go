package cloud

import "time"

const (
	ProviderAWS     = "aws"
	ProviderAliyun  = "aliyun"
	ProviderTencent = "tencent"
	ProviderGeneric = "generic"
)

type CloudAccount struct {
	ID            string              `json:"id" yaml:"id"`
	Name          string              `json:"name" yaml:"name"`
	Provider      string              `json:"provider" yaml:"provider"`
	Config        CloudProviderConfig `json:"config" yaml:"config"`
	CredentialRef string              `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Metadata      map[string]string   `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt     time.Time           `json:"createdAt" yaml:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt" yaml:"updatedAt"`
}

type CloudProviderConfig struct {
	Provider      string            `json:"provider" yaml:"provider"`
	AccountID     string            `json:"accountId,omitempty" yaml:"accountId,omitempty"`
	DefaultRegion string            `json:"defaultRegion,omitempty" yaml:"defaultRegion,omitempty"`
	Endpoint      string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Regions       []string          `json:"regions,omitempty" yaml:"regions,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type CloudProviderCapabilities struct {
	CredentialValidation bool `json:"credentialValidation" yaml:"credentialValidation"`
	Regions              bool `json:"regions" yaml:"regions"`
	Clusters             bool `json:"clusters" yaml:"clusters"`
	Hosts                bool `json:"hosts" yaml:"hosts"`
	Registries           bool `json:"registries" yaml:"registries"`
	InventorySnapshot    bool `json:"inventorySnapshot" yaml:"inventorySnapshot"`
	TargetBinding        bool `json:"targetBinding" yaml:"targetBinding"`
	RealCloudAPI         bool `json:"realCloudApi" yaml:"realCloudApi"`
}

type CloudProviderInfo struct {
	Name         string                    `json:"name" yaml:"name"`
	DisplayName  string                    `json:"displayName" yaml:"displayName"`
	Status       string                    `json:"status" yaml:"status"`
	Capabilities CloudProviderCapabilities `json:"capabilities" yaml:"capabilities"`
	SDK          string                    `json:"sdk,omitempty" yaml:"sdk,omitempty"`
	Warnings     []string                  `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

type CloudRegion struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Provider string `json:"provider" yaml:"provider"`
}

type CloudCluster struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Provider string `json:"provider" yaml:"provider"`
	Region   string `json:"region" yaml:"region"`
	Type     string `json:"type,omitempty" yaml:"type,omitempty"`
	Status   string `json:"status,omitempty" yaml:"status,omitempty"`
}

type CloudHost struct {
	ID       string            `json:"id" yaml:"id"`
	Name     string            `json:"name" yaml:"name"`
	Provider string            `json:"provider" yaml:"provider"`
	Region   string            `json:"region" yaml:"region"`
	Type     string            `json:"type,omitempty" yaml:"type,omitempty"`
	Status   string            `json:"status,omitempty" yaml:"status,omitempty"`
	Labels   map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type CloudRegistry struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Provider string `json:"provider" yaml:"provider"`
	Region   string `json:"region" yaml:"region"`
	Type     string `json:"type,omitempty" yaml:"type,omitempty"`
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

type CloudTargetBinding struct {
	ID            string            `json:"id" yaml:"id"`
	AccountID     string            `json:"accountId" yaml:"accountId"`
	Provider      string            `json:"provider" yaml:"provider"`
	EnvironmentID string            `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	TargetID      string            `json:"targetId,omitempty" yaml:"targetId,omitempty"`
	TargetType    string            `json:"targetType,omitempty" yaml:"targetType,omitempty"`
	Region        string            `json:"region,omitempty" yaml:"region,omitempty"`
	ClusterID     string            `json:"clusterId,omitempty" yaml:"clusterId,omitempty"`
	HostID        string            `json:"hostId,omitempty" yaml:"hostId,omitempty"`
	RegistryID    string            `json:"registryId,omitempty" yaml:"registryId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"createdAt" yaml:"createdAt"`
}

type CloudInventorySnapshot struct {
	ID          string               `json:"id" yaml:"id"`
	AccountID   string               `json:"accountId" yaml:"accountId"`
	Provider    string               `json:"provider" yaml:"provider"`
	Regions     []CloudRegion        `json:"regions,omitempty" yaml:"regions,omitempty"`
	Clusters    []CloudCluster       `json:"clusters,omitempty" yaml:"clusters,omitempty"`
	Hosts       []CloudHost          `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	Registries  []CloudRegistry      `json:"registries,omitempty" yaml:"registries,omitempty"`
	ScannedAt   time.Time            `json:"scannedAt" yaml:"scannedAt"`
	Warnings    []string             `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	GeneratedBy string               `json:"generatedBy,omitempty" yaml:"generatedBy,omitempty"`
	Bindings    []CloudTargetBinding `json:"bindings,omitempty" yaml:"bindings,omitempty"`
}
