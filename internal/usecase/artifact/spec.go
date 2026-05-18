package artifact

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadReleaseDefinitionFile(path string) (ReleaseDefinition, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return ReleaseDefinition{}, fmt.Errorf("read release definition: %w", err)
	}
	return ParseReleaseDefinition(body)
}

func ParseReleaseDefinition(body []byte) (ReleaseDefinition, error) {
	var def ReleaseDefinition
	if err := yaml.Unmarshal(body, &def); err != nil {
		return ReleaseDefinition{}, fmt.Errorf("decode release definition: %w", err)
	}
	if def.Kind != "Release" {
		return ReleaseDefinition{}, fmt.Errorf("release kind must be Release")
	}
	if def.Metadata.Name == "" {
		return ReleaseDefinition{}, fmt.Errorf("release metadata.name is required")
	}
	if def.Spec.Version == "" {
		return ReleaseDefinition{}, fmt.Errorf("release spec.version is required")
	}
	if len(def.Spec.Artifacts) == 0 {
		return ReleaseDefinition{}, fmt.Errorf("release must bind at least one artifact")
	}
	for i, artifact := range def.Spec.Artifacts {
		if artifact.Name == "" {
			return ReleaseDefinition{}, fmt.Errorf("release artifact %d name is required", i)
		}
		if artifact.Type == "" {
			return ReleaseDefinition{}, fmt.Errorf("release artifact %q type is required", artifact.Name)
		}
		if artifact.Reference == "" {
			return ReleaseDefinition{}, fmt.Errorf("release artifact %q reference is required", artifact.Name)
		}
	}
	return def, nil
}
