package event

import "time"

type Event struct {
	SpecVersion     string         `json:"specversion"`
	ID              string         `json:"id"`
	Type            string         `json:"type"`
	Source          string         `json:"source"`
	Subject         string         `json:"subject,omitempty"`
	Time            time.Time      `json:"time"`
	DataContentType string         `json:"datacontenttype"`
	Data            map[string]any `json:"data,omitempty"`
}

type LogChunk struct {
	ID              string    `json:"id"`
	PipelineRunID   string    `json:"pipelineRunId,omitempty"`
	DeploymentRunID string    `json:"deploymentRunId,omitempty"`
	StageRunID      string    `json:"stageRunId,omitempty"`
	JobRunID        string    `json:"jobRunId,omitempty"`
	StepRunID       string    `json:"stepRunId,omitempty"`
	Stream          string    `json:"stream"`
	Sequence        int64     `json:"sequence"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}
