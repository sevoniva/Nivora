package pipeline

import (
	"fmt"
	"time"

	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
)

func transitionPipelineRun(run *domainpipeline.PipelineRun, next domainpipeline.PipelineRunStatus, now time.Time, reason string) error {
	if !canTransitionPipelineRun(run.Status, next) {
		return fmt.Errorf("invalid PipelineRun transition from %s to %s", run.Status, next)
	}
	run.Status = next
	run.UpdatedAt = now
	if next == domainpipeline.PipelineRunRunning && run.StartedAt == nil {
		run.StartedAt = &now
	}
	if isTerminalPipelineStatus(next) {
		run.FinishedAt = &now
	}
	if reason != "" {
		run.FailureReason = reason
	}
	return nil
}

func canTransitionPipelineRun(from domainpipeline.PipelineRunStatus, to domainpipeline.PipelineRunStatus) bool {
	switch from {
	case domainpipeline.PipelineRunPending:
		return to == domainpipeline.PipelineRunQueued || to == domainpipeline.PipelineRunCanceled
	case domainpipeline.PipelineRunQueued:
		return to == domainpipeline.PipelineRunRunning || to == domainpipeline.PipelineRunCanceled
	case domainpipeline.PipelineRunRunning:
		return to == domainpipeline.PipelineRunPaused || to == domainpipeline.PipelineRunSucceeded ||
			to == domainpipeline.PipelineRunFailed || to == domainpipeline.PipelineRunCanceled ||
			to == domainpipeline.PipelineRunTimeout
	case domainpipeline.PipelineRunPaused:
		return to == domainpipeline.PipelineRunRunning || to == domainpipeline.PipelineRunCanceled
	default:
		return false
	}
}

func isTerminalPipelineStatus(status domainpipeline.PipelineRunStatus) bool {
	switch status {
	case domainpipeline.PipelineRunSucceeded, domainpipeline.PipelineRunFailed,
		domainpipeline.PipelineRunCanceled, domainpipeline.PipelineRunTimeout:
		return true
	default:
		return false
	}
}

func transitionJobRun(job *domainpipeline.JobRun, next domainpipeline.JobRunStatus, now time.Time, reason string) error {
	if !canTransitionJobRun(job.Status, next) {
		return fmt.Errorf("invalid JobRun transition from %s to %s", job.Status, next)
	}
	job.Status = next
	job.UpdatedAt = now
	if next == domainpipeline.JobRunRunning && job.StartedAt == nil {
		job.StartedAt = &now
	}
	if isTerminalJobStatus(next) {
		job.FinishedAt = &now
	}
	if reason != "" {
		job.FailureReason = reason
	}
	return nil
}

func transitionStageRun(stage *domainpipeline.StageRun, next domainpipeline.JobRunStatus, now time.Time, reason string) error {
	if !canTransitionStageOrStep(stage.Status, next) {
		return fmt.Errorf("invalid StageRun transition from %s to %s", stage.Status, next)
	}
	stage.Status = next
	stage.UpdatedAt = now
	if next == domainpipeline.JobRunRunning && stage.StartedAt == nil {
		stage.StartedAt = &now
	}
	if isTerminalJobStatus(next) {
		stage.FinishedAt = &now
	}
	if reason != "" {
		stage.FailureReason = reason
	}
	return nil
}

func transitionStepRun(step *domainpipeline.StepRun, next domainpipeline.JobRunStatus, now time.Time, reason string) error {
	if !canTransitionStageOrStep(step.Status, next) {
		return fmt.Errorf("invalid StepRun transition from %s to %s", step.Status, next)
	}
	step.Status = next
	step.UpdatedAt = now
	if next == domainpipeline.JobRunRunning && step.StartedAt == nil {
		step.StartedAt = &now
	}
	if isTerminalJobStatus(next) {
		step.FinishedAt = &now
	}
	if reason != "" {
		step.FailureReason = reason
	}
	return nil
}

func canTransitionStageOrStep(from domainpipeline.JobRunStatus, to domainpipeline.JobRunStatus) bool {
	switch from {
	case domainpipeline.JobRunPending:
		return to == domainpipeline.JobRunRunning || to == domainpipeline.JobRunSkipped || to == domainpipeline.JobRunCanceled
	case domainpipeline.JobRunRunning:
		return to == domainpipeline.JobRunSucceeded || to == domainpipeline.JobRunFailed || to == domainpipeline.JobRunCanceled
	default:
		return false
	}
}

func canTransitionJobRun(from domainpipeline.JobRunStatus, to domainpipeline.JobRunStatus) bool {
	switch from {
	case domainpipeline.JobRunPending:
		return to == domainpipeline.JobRunAssigned || to == domainpipeline.JobRunSkipped || to == domainpipeline.JobRunCanceled
	case domainpipeline.JobRunAssigned:
		return to == domainpipeline.JobRunRunning || to == domainpipeline.JobRunCanceled
	case domainpipeline.JobRunRunning:
		return to == domainpipeline.JobRunSucceeded || to == domainpipeline.JobRunFailed ||
			to == domainpipeline.JobRunRetrying || to == domainpipeline.JobRunCanceled
	case domainpipeline.JobRunFailed:
		return to == domainpipeline.JobRunRetrying
	case domainpipeline.JobRunRetrying:
		return to == domainpipeline.JobRunAssigned || to == domainpipeline.JobRunCanceled
	default:
		return false
	}
}

func isTerminalJobStatus(status domainpipeline.JobRunStatus) bool {
	switch status {
	case domainpipeline.JobRunSucceeded, domainpipeline.JobRunFailed,
		domainpipeline.JobRunSkipped, domainpipeline.JobRunCanceled:
		return true
	default:
		return false
	}
}
