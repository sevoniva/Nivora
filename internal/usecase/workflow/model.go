package workflow

import (
	"time"

	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

const (
	DefaultMaxJobs       = 100
	DefaultMaxSteps      = 200
	DefaultMaxMatrixSize = 64
	DefaultMaxEnvSize    = 4096
)

type Definition struct {
	APIVersion  string            `json:"apiVersion" yaml:"apiVersion"`
	Kind        string            `json:"kind" yaml:"kind"`
	Metadata    Metadata          `json:"metadata" yaml:"metadata"`
	On          TriggerSet        `json:"on" yaml:"on"`
	Permissions map[string]string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Jobs        map[string]Job    `json:"jobs" yaml:"jobs"`
	Artifacts   []ArtifactSpec    `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Cache       []CacheSpec       `json:"cache,omitempty" yaml:"cache,omitempty"`
	Security    map[string]any    `json:"security,omitempty" yaml:"security,omitempty"`
	Release     map[string]any    `json:"release,omitempty" yaml:"release,omitempty"`
	Deployment  map[string]any    `json:"deployment,omitempty" yaml:"deployment,omitempty"`
}

type Metadata struct {
	Name   string            `json:"name" yaml:"name"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type TriggerSet struct {
	Events []string `json:"events"`
}

type Job struct {
	Name           string            `json:"name,omitempty" yaml:"name,omitempty"`
	Needs          []string          `json:"needs,omitempty" yaml:"needs,omitempty"`
	RunsOn         []string          `json:"runsOn,omitempty" yaml:"runsOn,omitempty"`
	Labels         map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	TimeoutMinutes int               `json:"timeoutMinutes,omitempty" yaml:"timeoutMinutes,omitempty"`
	Strategy       Strategy          `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	Env            map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Steps          []Step            `json:"steps" yaml:"steps"`
}

type Strategy struct {
	Matrix Matrix `json:"matrix,omitempty" yaml:"matrix,omitempty"`
}

type Matrix struct {
	Values  map[string][]string `json:"values,omitempty" yaml:",inline"`
	Include []map[string]string `json:"include,omitempty" yaml:"include,omitempty"`
	Exclude []map[string]string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}

type Step struct {
	Name            string            `json:"name,omitempty" yaml:"name,omitempty"`
	Run             string            `json:"run,omitempty" yaml:"run,omitempty"`
	Uses            string            `json:"uses,omitempty" yaml:"uses,omitempty"`
	Env             map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	TimeoutMinutes  int               `json:"timeoutMinutes,omitempty" yaml:"timeoutMinutes,omitempty"`
	ContinueOnError bool              `json:"continueOnError,omitempty" yaml:"continueOnError,omitempty"`
}

type ArtifactSpec struct {
	Name string `json:"name" yaml:"name"`
	Path string `json:"path" yaml:"path"`
}

type CacheSpec struct {
	Key  string   `json:"key" yaml:"key"`
	Path []string `json:"path" yaml:"path"`
}

type PlanOptions struct {
	MaxJobs       int
	MaxSteps      int
	MaxMatrixSize int
	MaxEnvSize    int
}

type Plan struct {
	WorkflowID          string              `json:"workflowId"`
	Name                string              `json:"name"`
	Triggers            []string            `json:"triggers,omitempty"`
	Jobs                []PlannedJob        `json:"jobs"`
	Steps               []PlannedStep       `json:"steps"`
	Edges               []Edge              `json:"edges,omitempty"`
	MatrixExpansions    []MatrixExpansion   `json:"matrixExpansions,omitempty"`
	RunnerRequirements  []RunnerRequirement `json:"runnerRequirements,omitempty"`
	ArtifactOutputs     []ArtifactSpec      `json:"artifactOutputs,omitempty"`
	CacheHints          []CacheSpec         `json:"cacheHints,omitempty"`
	SecurityWarnings    []string            `json:"securityWarnings,omitempty"`
	UnsupportedFeatures []string            `json:"unsupportedFeatures,omitempty"`
	EstimatedMode       string              `json:"estimatedExecutionMode"`
	ConversionReady     bool                `json:"conversionReady"`
	Warnings            []string            `json:"warnings,omitempty"`
	CreatedAt           time.Time           `json:"createdAt"`
}

type PlannedJob struct {
	ID              string            `json:"id"`
	BaseID          string            `json:"baseId,omitempty"`
	Name            string            `json:"name"`
	Needs           []string          `json:"needs,omitempty"`
	RunsOn          []string          `json:"runsOn,omitempty"`
	Matrix          map[string]string `json:"matrix,omitempty"`
	TimeoutMinutes  int               `json:"timeoutMinutes,omitempty"`
	StepCount       int               `json:"stepCount"`
	ConversionReady bool              `json:"conversionReady"`
}

type PlannedStep struct {
	ID              string            `json:"id"`
	JobID           string            `json:"jobId"`
	Name            string            `json:"name"`
	Run             string            `json:"run,omitempty"`
	Uses            string            `json:"uses,omitempty"`
	TimeoutMinutes  int               `json:"timeoutMinutes,omitempty"`
	ContinueOnError bool              `json:"continueOnError,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	ConversionReady bool              `json:"conversionReady"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type MatrixExpansion struct {
	JobID  string            `json:"jobId"`
	Values map[string]string `json:"values"`
}

type RunnerRequirement struct {
	JobID  string   `json:"jobId"`
	RunsOn []string `json:"runsOn"`
}

type PipelineConversion struct {
	Definition pipelineusecase.Definition `json:"definition"`
	Warnings   []string                   `json:"warnings,omitempty"`
}
