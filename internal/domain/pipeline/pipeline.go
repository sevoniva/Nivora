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
	ID        string
	ProjectID string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PipelineVersion struct {
	ID         string
	PipelineID string
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type PipelineRun struct {
	ID                string
	PipelineID        string
	PipelineVersionID string
	Status            PipelineRunStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type StageRun struct {
	ID            string
	PipelineRunID string
	Name          string
	Status        JobRunStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type JobRun struct {
	ID         string
	StageRunID string
	Name       string
	Status     JobRunStatus
	RunnerID   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type StepRun struct {
	ID        string
	JobRunID  string
	Name      string
	Status    JobRunStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
