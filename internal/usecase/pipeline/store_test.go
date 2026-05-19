package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
)

func TestMemoryStoreLogOrdering(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	record := RunRecord{Run: domainpipeline.PipelineRun{
		ID:        "run-logs",
		Status:    domainpipeline.PipelineRunQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}}
	if err := store.Save(context.Background(), record); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.AppendLog(context.Background(), "run-logs", event.LogChunk{ID: "log-b", Content: "second"}); err != nil {
		t.Fatalf("append second: %v", err)
	}
	if err := store.AppendLog(context.Background(), "run-logs", event.LogChunk{ID: "log-a", Content: "first"}); err != nil {
		t.Fatalf("append first: %v", err)
	}
	logs, err := store.LogsByPipelineRun(context.Background(), "run-logs")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("logs = %#v", logs)
	}
	if logs[0].Sequence != 1 || logs[1].Sequence != 2 {
		t.Fatalf("sequences = %d, %d", logs[0].Sequence, logs[1].Sequence)
	}
}

func TestMemoryStoreSelectRunnerNoAvailableRunner(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.SelectRunner(context.Background(), "shell", nil)
	if !errors.Is(err, ErrRunnerNotFound) {
		t.Fatalf("error = %v", err)
	}
}

func TestMemoryStoreClaimJobAndLeaseExpiration(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	past := now.Add(-time.Minute)
	record := RunRecord{
		Run: domainpipeline.PipelineRun{ID: "run-claim", Status: domainpipeline.PipelineRunQueued, CreatedAt: now, UpdatedAt: now},
		Stages: []StageRecord{{
			Stage: domainpipeline.StageRun{ID: "stage-1", PipelineRunID: "run-claim", Status: domainpipeline.JobRunPending},
			Jobs: []JobRecord{{
				Job:   domainpipeline.JobRun{ID: "job-1", StageRunID: "stage-1", Status: domainpipeline.JobRunAssigned, LeaseExpiresAt: &past, Attempt: 1},
				Steps: []domainpipeline.StepRun{{ID: "step-1", JobRunID: "job-1", Status: domainpipeline.JobRunPending}},
			}},
		}},
	}
	if err := store.Save(context.Background(), record); err != nil {
		t.Fatalf("save: %v", err)
	}
	claim, err := store.ClaimJob(context.Background(), "runner-a", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim expired lease: %v", err)
	}
	if claim.JobRunID != "job-1" || claim.RunnerID != "runner-a" {
		t.Fatalf("claim = %#v", claim)
	}
	_, err = store.ClaimJob(context.Background(), "runner-b", now.Add(time.Minute))
	if !errors.Is(err, ErrNoClaimableJob) {
		t.Fatalf("expected no claimable job, got %v", err)
	}
}
