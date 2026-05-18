package deployment

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Definition struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       Spec     `json:"spec" yaml:"spec"`
}

type Metadata struct {
	Name string `json:"name" yaml:"name"`
}

type Spec struct {
	Application string     `json:"application" yaml:"application"`
	Environment string     `json:"environment" yaml:"environment"`
	Target      Target     `json:"target" yaml:"target"`
	Artifacts   []Artifact `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Manifests   []string   `json:"manifests" yaml:"manifests"`
	Options     Options    `json:"options,omitempty" yaml:"options,omitempty"`
}

type Target struct {
	Type      string `json:"type" yaml:"type"`
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

type Artifact struct {
	Name      string `json:"name" yaml:"name"`
	Type      string `json:"type" yaml:"type"`
	Reference string `json:"reference" yaml:"reference"`
}

type Options struct {
	DryRun bool `json:"dryRun" yaml:"dryRun"`
	Apply  bool `json:"apply" yaml:"apply"`
}

func LoadDefinitionFile(path string) (Definition, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("read deployment definition: %w", err)
	}
	return ParseDefinition(body)
}

func ParseDefinition(body []byte) (Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(body, &def); err != nil {
		return Definition{}, fmt.Errorf("decode deployment definition: %w", err)
	}
	if err := def.Validate(); err != nil {
		return Definition{}, err
	}
	return def, nil
}

func (d Definition) Validate() error {
	if d.Kind != "Deployment" {
		return errors.New("deployment kind must be Deployment")
	}
	if d.Metadata.Name == "" {
		return errors.New("deployment metadata.name is required")
	}
	if d.Spec.Application == "" {
		return errors.New("deployment spec.application is required")
	}
	if d.Spec.Environment == "" {
		return errors.New("deployment spec.environment is required")
	}
	if d.Spec.Target.Type == "" {
		return errors.New("deployment target.type is required")
	}
	if d.Spec.Target.Type != "kubernetes-yaml" {
		return fmt.Errorf("deployment target.type %q is not supported in Phase 2.0", d.Spec.Target.Type)
	}
	if d.Spec.Target.Name == "" {
		return errors.New("deployment target.name is required")
	}
	if d.Spec.Target.Namespace == "" {
		return errors.New("deployment target.namespace is required for kubernetes-yaml targets")
	}
	if len(d.Spec.Manifests) == 0 {
		return errors.New("deployment must reference at least one manifest")
	}
	for i, path := range d.Spec.Manifests {
		if path == "" {
			return fmt.Errorf("deployment manifest %d path is required", i)
		}
	}
	for i, artifact := range d.Spec.Artifacts {
		if artifact.Name == "" {
			return fmt.Errorf("deployment artifact %d name is required", i)
		}
		if artifact.Type == "" {
			return fmt.Errorf("deployment artifact %q type is required", artifact.Name)
		}
		if artifact.Reference == "" {
			return fmt.Errorf("deployment artifact %q reference is required", artifact.Name)
		}
	}
	return nil
}
