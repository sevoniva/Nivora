package credential

import (
	"os"

	domaincredential "github.com/sevoniva/nivora/internal/domain/credential"
	"gopkg.in/yaml.v3"
)

type Definition struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Metadata   struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec CredentialSpec `json:"spec" yaml:"spec"`
}

type CredentialSpec struct {
	Type      string                     `json:"type" yaml:"type"`
	ScopeType string                     `json:"scopeType" yaml:"scopeType"`
	ScopeID   string                     `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	SecretRef domaincredential.SecretRef `json:"secretRef" yaml:"secretRef"`
	Metadata  map[string]string          `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

func LoadDefinitionFile(path string) (Definition, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, err
	}
	var definition Definition
	if err := yaml.Unmarshal(body, &definition); err != nil {
		return Definition{}, err
	}
	return definition, nil
}

func (d Definition) CreateInput() CredentialCreateInput {
	return CredentialCreateInput{
		Name:      d.Metadata.Name,
		Type:      d.Spec.Type,
		ScopeType: d.Spec.ScopeType,
		ScopeID:   d.Spec.ScopeID,
		SecretRef: d.Spec.SecretRef,
		Metadata:  d.Spec.Metadata,
	}
}
