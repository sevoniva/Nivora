package pipeline

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
	Stages         []Stage `json:"stages" yaml:"stages"`
	TimeoutSeconds int     `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
}

type Stage struct {
	Name string `json:"name" yaml:"name"`
	Jobs []Job  `json:"jobs" yaml:"jobs"`
}

type Job struct {
	Name           string `json:"name" yaml:"name"`
	Executor       string `json:"executor" yaml:"executor"`
	Retries        int    `json:"retries,omitempty" yaml:"retries,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
	Steps          []Step `json:"steps" yaml:"steps"`
}

type Step struct {
	Name           string `json:"name" yaml:"name"`
	Run            string `json:"run" yaml:"run"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
}

func LoadDefinitionFile(path string) (Definition, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("read pipeline definition: %w", err)
	}
	return ParseDefinition(body)
}

func ParseDefinition(body []byte) (Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(body, &def); err != nil {
		return Definition{}, fmt.Errorf("decode pipeline definition: %w", err)
	}
	if err := def.Validate(); err != nil {
		return Definition{}, err
	}
	return def, nil
}

func (d Definition) Validate() error {
	if d.Kind != "Pipeline" {
		return errors.New("pipeline kind must be Pipeline")
	}
	if d.Metadata.Name == "" {
		return errors.New("pipeline metadata.name is required")
	}
	if len(d.Spec.Stages) == 0 {
		return errors.New("pipeline must define at least one stage")
	}
	for i, stage := range d.Spec.Stages {
		if stage.Name == "" {
			return fmt.Errorf("stage %d name is required", i)
		}
		if len(stage.Jobs) == 0 {
			return fmt.Errorf("stage %q must define at least one job", stage.Name)
		}
		for j, job := range stage.Jobs {
			if job.Name == "" {
				return fmt.Errorf("stage %q job %d name is required", stage.Name, j)
			}
			if job.Executor != "shell" {
				return fmt.Errorf("job %q uses unsupported executor %q", job.Name, job.Executor)
			}
			if len(job.Steps) == 0 {
				return fmt.Errorf("job %q must define at least one step", job.Name)
			}
			for k, step := range job.Steps {
				if step.Run == "" {
					return fmt.Errorf("job %q step %d run command is required", job.Name, k)
				}
			}
		}
	}
	return nil
}
