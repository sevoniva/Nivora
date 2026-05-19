package postgres

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func TestClaimRecordJobUpdatesLeaseAndReturnsPlan(t *testing.T) {
	now := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	leaseUntil := now.Add(30 * time.Second)
	record := pipelineusecase.RunRecord{
		Definition: pipelineusecase.Definition{
			Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
				Name: "build",
				Jobs: []pipelineusecase.Job{{
					Name:     "echo",
					Executor: "shell",
					Steps:    []pipelineusecase.Step{{Name: "say", Run: "printf alpha"}},
				}},
			}}},
		},
		Run: domainpipeline.PipelineRun{ID: "prun-1", Status: domainpipeline.PipelineRunQueued, CreatedAt: now, UpdatedAt: now},
		Stages: []pipelineusecase.StageRecord{{
			Stage: domainpipeline.StageRun{ID: "stage-1", PipelineRunID: "prun-1", Status: domainpipeline.JobRunPending},
			Jobs: []pipelineusecase.JobRecord{{
				Job:   domainpipeline.JobRun{ID: "job-1", StageRunID: "stage-1", Status: domainpipeline.JobRunPending, Attempt: 1},
				Steps: []domainpipeline.StepRun{{ID: "step-1", JobRunID: "job-1", Status: domainpipeline.JobRunPending}},
			}},
		}},
	}

	claim, ok := claimRecordJob(&record, domainrunner.Runner{ID: "runner-1", Status: "online", Executors: []string{"shell"}}, leaseUntil, now)
	if !ok {
		t.Fatal("expected claimable job")
	}
	if record.Run.Status != domainpipeline.PipelineRunRunning {
		t.Fatalf("run status = %s, want Running", record.Run.Status)
	}
	if record.Stages[0].Jobs[0].Job.Status != domainpipeline.JobRunAssigned {
		t.Fatalf("job status = %s, want Assigned", record.Stages[0].Jobs[0].Job.Status)
	}
	if claim.PipelineRunID != "prun-1" || claim.JobRunID != "job-1" || claim.RunnerID != "runner-1" {
		t.Fatalf("unexpected claim: %#v", claim)
	}
	if len(claim.Commands) != 1 || claim.Commands[0] != "printf alpha" {
		t.Fatalf("commands = %#v", claim.Commands)
	}
	if len(claim.StepRunIDs) != 1 || claim.StepRunIDs[0] != "step-1" {
		t.Fatalf("step IDs = %#v", claim.StepRunIDs)
	}
}

func TestUpdateRecordJobStatusMarksFailure(t *testing.T) {
	now := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	record := pipelineusecase.RunRecord{
		Run: domainpipeline.PipelineRun{ID: "prun-1", Status: domainpipeline.PipelineRunRunning, UpdatedAt: now},
		Stages: []pipelineusecase.StageRecord{{
			Stage: domainpipeline.StageRun{ID: "stage-1"},
			Jobs: []pipelineusecase.JobRecord{{
				Job: domainpipeline.JobRun{ID: "job-1", StageRunID: "stage-1", Status: domainpipeline.JobRunRunning},
			}},
		}},
	}

	ok := updateRecordJobStatus(&record, "job-1", domainpipeline.JobRunFailed, "exit 1", now.Add(time.Minute))
	if !ok {
		t.Fatal("expected status update")
	}
	if record.Run.Status != domainpipeline.PipelineRunFailed {
		t.Fatalf("run status = %s, want Failed", record.Run.Status)
	}
	if record.Stages[0].Jobs[0].Job.FailureReason != "exit 1" {
		t.Fatalf("failure reason = %q", record.Stages[0].Jobs[0].Job.FailureReason)
	}
	if record.Stages[0].Jobs[0].Job.LeaseExpiresAt != nil {
		t.Fatal("terminal job should clear lease")
	}
}

func TestPersistenceMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000003_persistence_foundation.up.sql")
	down := readMigration(t, "000003_persistence_foundation.down.sql")

	requiredTables := []string{
		"runtime_pipeline_runs",
		"runtime_job_runs",
		"runtime_log_chunks",
		"runtime_events",
		"runtime_audit_logs",
		"runtime_runners",
		"runtime_event_outbox",
		"idempotency_keys",
	}
	for _, table := range requiredTables {
		if !strings.Contains(up, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("up migration missing table %s", table)
		}
		if !strings.Contains(down, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("down migration missing table %s", table)
		}
	}
	for _, index := range []string{
		"idx_runtime_pipeline_runs_status_created_at",
		"idx_runtime_job_runs_lease",
		"idx_runtime_log_chunks_run_sequence",
		"idx_runtime_outbox_status_created_at",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("up migration missing index %s", index)
		}
		if !strings.Contains(down, index) {
			t.Fatalf("down migration missing index %s", index)
		}
	}
}

func readMigration(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "infra", "migration", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	return string(body)
}
