package pipeline

import "time"

type PipelineRunStatus string

const (
	PipelineRunPending   PipelineRunStatus = "Pending"
	PipelineRunQueued    PipelineRunStatus = "Queued"
	PipelineRunRunning   PipelineRunStatus = "Running"
	PipelineRunPaused    PipelineRunStatus = "Paused"
	PipelineRunSucceeded PipelineRunStatus = "Succeeded"
	PipelineRunFailed    PipelineRunStatus = "Failed"
	PipelineRunCanceled  PipelineRunStatus = "Canceled"
	PipelineRunTimeout   PipelineRunStatus = "Timeout"
)

func (s PipelineRunStatus) Valid() bool {
	switch s {
	case PipelineRunPending, PipelineRunQueued, PipelineRunRunning, PipelineRunPaused,
		PipelineRunSucceeded, PipelineRunFailed, PipelineRunCanceled, PipelineRunTimeout:
		return true
	default:
		return false
	}
}

type JobRunStatus string

const (
	JobRunPending   JobRunStatus = "Pending"
	JobRunAssigned  JobRunStatus = "Assigned"
	JobRunRunning   JobRunStatus = "Running"
	JobRunSucceeded JobRunStatus = "Succeeded"
	JobRunFailed    JobRunStatus = "Failed"
	JobRunSkipped   JobRunStatus = "Skipped"
	JobRunRetrying  JobRunStatus = "Retrying"
	JobRunCanceled  JobRunStatus = "Canceled"
)

func (s JobRunStatus) Valid() bool {
	switch s {
	case JobRunPending, JobRunAssigned, JobRunRunning, JobRunSucceeded,
		JobRunFailed, JobRunSkipped, JobRunRetrying, JobRunCanceled:
		return true
	default:
		return false
	}
}

type Pipeline struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PipelineVersion struct {
	ID         string    `json:"id"`
	PipelineID string    `json:"pipelineId"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type PipelineRun struct {
	ID                string            `json:"id"`
	PipelineID        string            `json:"pipelineId"`
	PipelineVersionID string            `json:"pipelineVersionId,omitempty"`
	Status            PipelineRunStatus `json:"status"`
	StartedAt         *time.Time        `json:"startedAt,omitempty"`
	FinishedAt        *time.Time        `json:"finishedAt,omitempty"`
	FailureReason     string            `json:"failureReason,omitempty"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
}

type StageRun struct {
	ID            string       `json:"id"`
	PipelineRunID string       `json:"pipelineRunId"`
	Name          string       `json:"name"`
	Status        JobRunStatus `json:"status"`
	StartedAt     *time.Time   `json:"startedAt,omitempty"`
	FinishedAt    *time.Time   `json:"finishedAt,omitempty"`
	FailureReason string       `json:"failureReason,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

type JobRun struct {
	ID            string       `json:"id"`
	StageRunID    string       `json:"stageRunId"`
	Name          string       `json:"name"`
	Status        JobRunStatus `json:"status"`
	RunnerID      string       `json:"runnerId,omitempty"`
	Attempt       int          `json:"attempt"`
	MaxRetries    int          `json:"maxRetries"`
	StartedAt     *time.Time   `json:"startedAt,omitempty"`
	FinishedAt    *time.Time   `json:"finishedAt,omitempty"`
	FailureReason string       `json:"failureReason,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

type StepRun struct {
	ID            string       `json:"id"`
	JobRunID      string       `json:"jobRunId"`
	Name          string       `json:"name"`
	Status        JobRunStatus `json:"status"`
	Attempt       int          `json:"attempt"`
	StartedAt     *time.Time   `json:"startedAt,omitempty"`
	FinishedAt    *time.Time   `json:"finishedAt,omitempty"`
	FailureReason string       `json:"failureReason,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}
