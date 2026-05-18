package pipeline

import (
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
)

type RunRecord struct {
	Pipeline domainpipeline.Pipeline    `json:"pipeline"`
	Run      domainpipeline.PipelineRun `json:"run"`
	Stages   []StageRecord              `json:"stages"`
	Logs     []LogRecord                `json:"logs,omitempty"`
	Events   []event.Event              `json:"events,omitempty"`
	Audits   []audit.AuditLog           `json:"audits,omitempty"`
}

type StageRecord struct {
	Stage domainpipeline.StageRun `json:"stage"`
	Jobs  []JobRecord             `json:"jobs"`
}

type JobRecord struct {
	Job   domainpipeline.JobRun    `json:"job"`
	Steps []domainpipeline.StepRun `json:"steps"`
}

type LogRecord struct {
	ID            string    `json:"id"`
	PipelineRunID string    `json:"pipelineRunId"`
	JobRunID      string    `json:"jobRunId"`
	StepRunID     string    `json:"stepRunId"`
	Stream        string    `json:"stream"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"createdAt"`
}

type CreateRunInput struct {
	Definition Definition
	ActorID    string
}

type CreateRunResult struct {
	Record RunRecord
}
