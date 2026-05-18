package pipeline

import (
	"testing"
	"time"

	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
)

func TestPipelineRunTransitions(t *testing.T) {
	now := time.Now()
	run := domainpipeline.PipelineRun{Status: domainpipeline.PipelineRunPending}
	if err := transitionPipelineRun(&run, domainpipeline.PipelineRunQueued, now, ""); err != nil {
		t.Fatalf("queue transition: %v", err)
	}
	if err := transitionPipelineRun(&run, domainpipeline.PipelineRunSucceeded, now, ""); err == nil {
		t.Fatal("expected invalid transition from queued to succeeded")
	}
	if err := transitionPipelineRun(&run, domainpipeline.PipelineRunRunning, now, ""); err != nil {
		t.Fatalf("running transition: %v", err)
	}
	if err := transitionPipelineRun(&run, domainpipeline.PipelineRunSucceeded, now, ""); err != nil {
		t.Fatalf("succeeded transition: %v", err)
	}
	if run.StartedAt == nil || run.FinishedAt == nil {
		t.Fatal("expected timestamps")
	}
	if err := transitionPipelineRun(&run, domainpipeline.PipelineRunRunning, now, ""); err == nil {
		t.Fatal("expected terminal PipelineRun to reject transition back to running")
	}
}

func TestJobRunTransitions(t *testing.T) {
	now := time.Now()
	job := domainpipeline.JobRun{Status: domainpipeline.JobRunPending}
	if err := transitionJobRun(&job, domainpipeline.JobRunAssigned, now, ""); err != nil {
		t.Fatalf("assigned transition: %v", err)
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunRunning, now, ""); err != nil {
		t.Fatalf("running transition: %v", err)
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunPending, now, ""); err == nil {
		t.Fatal("expected invalid transition back to pending")
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunRetrying, now, ""); err != nil {
		t.Fatalf("retrying transition: %v", err)
	}
}

func TestJobRunTerminalTransitions(t *testing.T) {
	now := time.Now()
	job := domainpipeline.JobRun{Status: domainpipeline.JobRunPending}
	if err := transitionJobRun(&job, domainpipeline.JobRunAssigned, now, ""); err != nil {
		t.Fatalf("assigned transition: %v", err)
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunRunning, now, ""); err != nil {
		t.Fatalf("running transition: %v", err)
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunSucceeded, now, ""); err != nil {
		t.Fatalf("succeeded transition: %v", err)
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunRunning, now, ""); err == nil {
		t.Fatal("expected terminal JobRun to reject transition back to running")
	}
}

func TestFailedJobRunCanRetry(t *testing.T) {
	now := time.Now()
	job := domainpipeline.JobRun{Status: domainpipeline.JobRunPending}
	if err := transitionJobRun(&job, domainpipeline.JobRunAssigned, now, ""); err != nil {
		t.Fatalf("assigned transition: %v", err)
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunRunning, now, ""); err != nil {
		t.Fatalf("running transition: %v", err)
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunFailed, now, "failed"); err != nil {
		t.Fatalf("failed transition: %v", err)
	}
	if err := transitionJobRun(&job, domainpipeline.JobRunRetrying, now, "retrying"); err != nil {
		t.Fatalf("retrying transition: %v", err)
	}
}

func TestStageAndStepTransitions(t *testing.T) {
	now := time.Now()
	stage := domainpipeline.StageRun{Status: domainpipeline.JobRunPending}
	if err := transitionStageRun(&stage, domainpipeline.JobRunRunning, now, ""); err != nil {
		t.Fatalf("stage running: %v", err)
	}
	if err := transitionStageRun(&stage, domainpipeline.JobRunSucceeded, now, ""); err != nil {
		t.Fatalf("stage succeeded: %v", err)
	}

	step := domainpipeline.StepRun{Status: domainpipeline.JobRunPending}
	if err := transitionStepRun(&step, domainpipeline.JobRunRunning, now, ""); err != nil {
		t.Fatalf("step running: %v", err)
	}
	if err := transitionStepRun(&step, domainpipeline.JobRunSucceeded, now, ""); err != nil {
		t.Fatalf("step succeeded: %v", err)
	}
}

func TestStepFailureTransition(t *testing.T) {
	now := time.Now()
	step := domainpipeline.StepRun{Status: domainpipeline.JobRunPending}
	if err := transitionStepRun(&step, domainpipeline.JobRunRunning, now, ""); err != nil {
		t.Fatalf("step running: %v", err)
	}
	if err := transitionStepRun(&step, domainpipeline.JobRunFailed, now, "failed"); err != nil {
		t.Fatalf("step failed: %v", err)
	}
	if step.FailureReason != "failed" {
		t.Fatalf("failure reason = %q", step.FailureReason)
	}
}
