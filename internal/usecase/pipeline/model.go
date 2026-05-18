package pipeline

import (
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
)

type RunRecord struct {
	Pipeline   domainpipeline.Pipeline    `json:"pipeline"`
	Run        domainpipeline.PipelineRun `json:"run"`
	Definition Definition                 `json:"definition,omitempty"`
	Stages     []StageRecord              `json:"stages"`
	Logs       []event.LogChunk           `json:"logs,omitempty"`
	Events     []event.Event              `json:"events,omitempty"`
	Audits     []audit.AuditLog           `json:"audits,omitempty"`
}

type StageRecord struct {
	Stage domainpipeline.StageRun `json:"stage"`
	Jobs  []JobRecord             `json:"jobs"`
}

type JobRecord struct {
	Job   domainpipeline.JobRun    `json:"job"`
	Steps []domainpipeline.StepRun `json:"steps"`
}

type TimelineEntry struct {
	Type    string            `json:"type"`
	Time    time.Time         `json:"time"`
	Subject string            `json:"subject"`
	Status  string            `json:"status,omitempty"`
	Message string            `json:"message,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

type RunnerRecord struct {
	Runner domainrunner.Runner `json:"runner"`
}

type CreateRunInput struct {
	Definition Definition
	ActorID    string
}

type CreateRunResult struct {
	Record RunRecord
}
