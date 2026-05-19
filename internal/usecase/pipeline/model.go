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

type JobClaim struct {
	PipelineRunID   string                      `json:"pipelineRunId"`
	StageRunID      string                      `json:"stageRunId"`
	JobRunID        string                      `json:"jobRunId"`
	StepRunIDs      []string                    `json:"stepRunIds,omitempty"`
	RunnerID        string                      `json:"runnerId"`
	Executor        string                      `json:"executor"`
	Commands        []string                    `json:"commands,omitempty"`
	Attempt         int                         `json:"attempt"`
	LeaseExpiresAt  time.Time                   `json:"leaseExpiresAt"`
	CancelRequested bool                        `json:"cancelRequested,omitempty"`
	Status          domainpipeline.JobRunStatus `json:"status"`
}

type AppendJobLogInput struct {
	PipelineRunID string `json:"pipelineRunId"`
	StageRunID    string `json:"stageRunId,omitempty"`
	StepRunID     string `json:"stepRunId,omitempty"`
	Stream        string `json:"stream"`
	Content       string `json:"content"`
}

type UpdateJobStatusInput struct {
	Status domainpipeline.JobRunStatus `json:"status"`
	Reason string                      `json:"reason,omitempty"`
}

type EventOutboxRecord struct {
	ID          string      `json:"id"`
	EventType   string      `json:"eventType"`
	Subject     string      `json:"subject"`
	Payload     event.Event `json:"payload"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"createdAt"`
	PublishedAt *time.Time  `json:"publishedAt,omitempty"`
}

type CreateRunInput struct {
	Definition    Definition
	ActorID       string
	CorrelationID string
}

type CreateRunResult struct {
	Record RunRecord
}
