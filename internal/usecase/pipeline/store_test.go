package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
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
	if err := store.RegisterRunner(context.Background(), domainrunner.Runner{ID: "runner-a", Name: "runner-a", Status: "online", Executors: []string{"shell"}, MaxConcurrency: 1}); err != nil {
		t.Fatalf("register runner-a: %v", err)
	}
	claim, err := store.ClaimJob(context.Background(), "runner-a", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim expired lease: %v", err)
	}
	if claim.JobRunID != "job-1" || claim.RunnerID != "runner-a" {
		t.Fatalf("claim = %#v", claim)
	}
	if err := store.RegisterRunner(context.Background(), domainrunner.Runner{ID: "runner-b", Name: "runner-b", Status: "online", Executors: []string{"shell"}, MaxConcurrency: 1}); err != nil {
		t.Fatalf("register runner-b: %v", err)
	}
	_, err = store.ClaimJob(context.Background(), "runner-b", now.Add(time.Minute))
	if !errors.Is(err, ErrNoClaimableJob) {
		t.Fatalf("expected no claimable job, got %v", err)
	}
}

func TestMemoryStorePipelineRunLeaseAndRecoveryQueries(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	record := RunRecord{Run: domainpipeline.PipelineRun{
		ID:        "run-lease",
		Status:    domainpipeline.PipelineRunRunning,
		CreatedAt: now,
		UpdatedAt: now.Add(-10 * time.Minute),
	}}
	if err := store.Save(context.Background(), record); err != nil {
		t.Fatalf("save: %v", err)
	}
	leased, err := store.AcquirePipelineRunLease(context.Background(), "run-lease", "worker-a", now.Add(time.Minute), now)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	if leased.Run.OwnerID != "worker-a" || leased.Run.LeaseExpiresAt == nil || leased.Run.HeartbeatAt == nil || leased.Run.Attempt != 1 {
		t.Fatalf("leased run = %#v", leased.Run)
	}
	stale, err := store.ListStaleRunningPipelineRuns(context.Background(), now.Add(2*time.Minute), 10)
	if err != nil {
		t.Fatalf("list stale: %v", err)
	}
	if len(stale) != 1 {
		t.Fatalf("stale = %#v", stale)
	}
	if _, err := store.HeartbeatPipelineRunLease(context.Background(), "run-lease", "worker-b", now.Add(time.Minute), now); err == nil {
		t.Fatal("expected heartbeat owned by another worker to fail")
	}
}
