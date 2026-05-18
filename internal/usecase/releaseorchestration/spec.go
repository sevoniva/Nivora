package releaseorchestration

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

func LoadDefinitionFile(path string) (Definition, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("read release orchestration definition: %w", err)
	}
	return ParseDefinition(body)
}

func ParseDefinition(body []byte) (Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(body, &def); err != nil {
		return Definition{}, fmt.Errorf("decode release orchestration definition: %w", err)
	}
	if err := def.Validate(); err != nil {
		return Definition{}, err
	}
	return def, nil
}

func (d Definition) Validate() error {
	if d.Kind != "ReleaseOrchestration" {
		return errors.New("release orchestration kind must be ReleaseOrchestration")
	}
	if d.Metadata.Name == "" {
		return errors.New("release orchestration metadata.name is required")
	}
	if d.Spec.Environment == "" {
		return errors.New("release orchestration spec.environment is required")
	}
	if d.Spec.ReleaseID == "" {
		if d.Spec.Release.Kind == "" {
			return errors.New("release orchestration requires spec.release or spec.releaseId")
		}
		if d.Spec.Release.Kind != "Release" {
			return errors.New("release orchestration spec.release.kind must be Release")
		}
		if d.Spec.Release.Metadata.Name == "" {
			return errors.New("release orchestration spec.release.metadata.name is required")
		}
		if d.Spec.Release.Spec.Version == "" {
			return errors.New("release orchestration spec.release.spec.version is required")
		}
		if len(d.Spec.Release.Spec.Artifacts) == 0 {
			return errors.New("release orchestration spec.release must bind at least one artifact")
		}
	}
	if len(d.Spec.Targets) == 0 {
		return errors.New("release orchestration requires at least one target")
	}
	strategy := d.Spec.Strategy
	if strategy == "" {
		strategy = StrategySequential
	}
	switch strategy {
	case StrategyPlanOnly, StrategySequential, StrategyParallel:
	default:
		return fmt.Errorf("release orchestration strategy %q is not supported", strategy)
	}
	if strategy == StrategyParallel && d.Spec.Concurrency < 0 {
		return errors.New("release orchestration concurrency cannot be negative")
	}
	seen := map[string]struct{}{}
	for i, target := range d.Spec.Targets {
		if target.Name == "" {
			return fmt.Errorf("release orchestration target %d name is required", i)
		}
		if _, ok := seen[target.Name]; ok {
			return fmt.Errorf("release orchestration target %q is duplicated", target.Name)
		}
		seen[target.Name] = struct{}{}
		if target.Type == "" {
			return fmt.Errorf("release orchestration target %q type is required", target.Name)
		}
		if target.Type != "kubernetes-yaml" && target.Type != "argocd" && target.Type != "noop" && target.Type != "webhook" {
			return fmt.Errorf("release orchestration target %q type %q is not supported in Phase 2.7", target.Name, target.Type)
		}
		if target.Enabled != nil && !*target.Enabled {
			continue
		}
		if target.Type == "noop" || target.Type == "webhook" {
			continue
		}
		if err := target.Deployment.Validate(); err != nil {
			return fmt.Errorf("release orchestration target %q deployment invalid: %w", target.Name, err)
		}
	}
	return nil
}

func orderedTargets(targets []TargetSpec) []TargetSpec {
	enabled := make([]TargetSpec, 0, len(targets))
	for _, target := range targets {
		if target.Enabled != nil && !*target.Enabled {
			continue
		}
		enabled = append(enabled, target)
	}
	sort.SliceStable(enabled, func(i, j int) bool {
		if enabled[i].Order == enabled[j].Order {
			return enabled[i].Name < enabled[j].Name
		}
		return enabled[i].Order < enabled[j].Order
	})
	return enabled
}
