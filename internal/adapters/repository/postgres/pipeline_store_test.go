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

	claim, ok := claimRecordJob(&record, domainrunner.Runner{ID: "runner-1", Status: "online", Executors: []string{"shell"}}, domainrunner.RunnerGroup{}, leaseUntil, now)
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

func TestClaimRecordJobRespectsRunnerProjectScope(t *testing.T) {
	now := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	leaseUntil := now.Add(30 * time.Second)
	record := pipelineusecase.RunRecord{
		Pipeline: domainpipeline.Pipeline{ID: "pipe-1", ProjectID: "project-b", Name: "scoped"},
		Definition: pipelineusecase.Definition{
			Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
				Name: "build",
				Jobs: []pipelineusecase.Job{{
					Name:     "echo",
					Executor: "shell",
					Steps:    []pipelineusecase.Step{{Name: "say", Run: "printf scoped"}},
				}},
			}}},
		},
		Run: domainpipeline.PipelineRun{ID: "prun-scoped", Status: domainpipeline.PipelineRunQueued, CreatedAt: now, UpdatedAt: now},
		Stages: []pipelineusecase.StageRecord{{
			Stage: domainpipeline.StageRun{ID: "stage-1", PipelineRunID: "prun-scoped", Status: domainpipeline.JobRunPending},
			Jobs: []pipelineusecase.JobRecord{{
				Job:   domainpipeline.JobRun{ID: "job-1", StageRunID: "stage-1", Status: domainpipeline.JobRunPending, Attempt: 1},
				Steps: []domainpipeline.StepRun{{ID: "step-1", JobRunID: "job-1", Status: domainpipeline.JobRunPending}},
			}},
		}},
	}
	runner := domainrunner.Runner{
		ID:        "runner-project-a",
		Status:    "online",
		Labels:    map[string]string{"projectId": "project-a"},
		Executors: []string{"shell"},
	}

	claim, ok := claimRecordJob(&record, runner, domainrunner.RunnerGroup{}, leaseUntil, now)
	if ok || claim.JobRunID != "" {
		t.Fatalf("project-a runner should not claim project-b record: ok=%v claim=%#v", ok, claim)
	}
	if record.Run.Status != domainpipeline.PipelineRunQueued || record.Stages[0].Jobs[0].Job.RunnerID != "" {
		t.Fatalf("scope mismatch mutated record: run=%#v job=%#v", record.Run, record.Stages[0].Jobs[0].Job)
	}

	record.Pipeline.ProjectID = "project-a"
	claim, ok = claimRecordJob(&record, runner, domainrunner.RunnerGroup{}, leaseUntil, now)
	if !ok {
		t.Fatal("project-a runner should claim project-a record")
	}
	if claim.PipelineRunID != "prun-scoped" || claim.JobRunID != "job-1" || claim.RunnerID != "runner-project-a" {
		t.Fatalf("unexpected claim: %#v", claim)
	}
}

func TestClaimRecordJobRespectsRunnerEnvironmentScope(t *testing.T) {
	now := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	leaseUntil := now.Add(30 * time.Second)
	record := pipelineusecase.RunRecord{
		Pipeline: domainpipeline.Pipeline{
			ID:        "pipe-1",
			ProjectID: "project-a",
			Name:      "scoped",
			Labels:    map[string]string{"environmentId": "env-dev"},
			Metadata:  map[string]string{"environmentId": "env-dev"},
		},
		Definition: pipelineusecase.Definition{
			Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
				Name: "build",
				Jobs: []pipelineusecase.Job{{
					Name:     "echo",
					Executor: "shell",
					Steps:    []pipelineusecase.Step{{Name: "say", Run: "printf scoped"}},
				}},
			}}},
		},
		Run: domainpipeline.PipelineRun{ID: "prun-env", Status: domainpipeline.PipelineRunQueued, CreatedAt: now, UpdatedAt: now},
		Stages: []pipelineusecase.StageRecord{{
			Stage: domainpipeline.StageRun{ID: "stage-1", PipelineRunID: "prun-env", Status: domainpipeline.JobRunPending},
			Jobs: []pipelineusecase.JobRecord{{
				Job:   domainpipeline.JobRun{ID: "job-1", StageRunID: "stage-1", Status: domainpipeline.JobRunPending, Attempt: 1},
				Steps: []domainpipeline.StepRun{{ID: "step-1", JobRunID: "job-1", Status: domainpipeline.JobRunPending}},
			}},
		}},
	}
	runner := domainrunner.Runner{
		ID:        "runner-env-prod",
		Status:    "online",
		Labels:    map[string]string{"environmentId": "env-prod"},
		Executors: []string{"shell"},
	}

	claim, ok := claimRecordJob(&record, runner, domainrunner.RunnerGroup{}, leaseUntil, now)
	if ok || claim.JobRunID != "" {
		t.Fatalf("env-prod runner should not claim env-dev record: ok=%v claim=%#v", ok, claim)
	}
	if record.Run.Status != domainpipeline.PipelineRunQueued || record.Stages[0].Jobs[0].Job.RunnerID != "" {
		t.Fatalf("scope mismatch mutated record: run=%#v job=%#v", record.Run, record.Stages[0].Jobs[0].Job)
	}

	record.Pipeline.Labels["environmentId"] = "env-prod"
	record.Pipeline.Metadata["environmentId"] = "env-prod"
	claim, ok = claimRecordJob(&record, runner, domainrunner.RunnerGroup{}, leaseUntil, now)
	if !ok {
		t.Fatal("env-prod runner should claim env-prod record")
	}
	if claim.PipelineRunID != "prun-env" || claim.JobRunID != "job-1" || claim.RunnerID != "runner-env-prod" {
		t.Fatalf("unexpected claim: %#v", claim)
	}
}

func TestClaimRecordJobRespectsRunnerGroupScopeAndExecutors(t *testing.T) {
	now := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	leaseUntil := now.Add(30 * time.Second)
	record := pipelineusecase.RunRecord{
		Pipeline: domainpipeline.Pipeline{
			ID:        "pipe-1",
			ProjectID: "project-a",
			Name:      "group-scoped",
			Labels:    map[string]string{"environmentId": "env-dev"},
			Metadata:  map[string]string{"environmentId": "env-dev"},
		},
		Definition: pipelineusecase.Definition{
			Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
				Name: "build",
				Jobs: []pipelineusecase.Job{{
					Name:     "echo",
					Executor: "shell",
					Steps:    []pipelineusecase.Step{{Name: "say", Run: "printf group"}},
				}},
			}}},
		},
		Run: domainpipeline.PipelineRun{ID: "prun-group", Status: domainpipeline.PipelineRunQueued, CreatedAt: now, UpdatedAt: now},
		Stages: []pipelineusecase.StageRecord{{
			Stage: domainpipeline.StageRun{ID: "stage-1", PipelineRunID: "prun-group", Status: domainpipeline.JobRunPending},
			Jobs: []pipelineusecase.JobRecord{{
				Job:   domainpipeline.JobRun{ID: "job-1", StageRunID: "stage-1", Status: domainpipeline.JobRunPending, Attempt: 1},
				Steps: []domainpipeline.StepRun{{ID: "step-1", JobRunID: "job-1", Status: domainpipeline.JobRunPending}},
			}},
		}},
	}
	runner := domainrunner.Runner{ID: "runner-group", GroupID: "rgrp-prod", Status: "online", Executors: []string{"shell"}}
	group := domainrunner.RunnerGroup{ID: "rgrp-prod", ProjectID: "project-a", EnvironmentIDs: []string{"env-prod"}, Executors: []string{"shell"}}

	claim, ok := claimRecordJob(&record, runner, group, leaseUntil, now)
	if ok || claim.JobRunID != "" {
		t.Fatalf("group should not claim disallowed env: ok=%v claim=%#v", ok, claim)
	}
	record.Pipeline.Labels["environmentId"] = "env-prod"
	record.Pipeline.Metadata["environmentId"] = "env-prod"
	claim, ok = claimRecordJob(&record, runner, domainrunner.RunnerGroup{ID: "rgrp-prod", ProjectID: "project-a", EnvironmentIDs: []string{"env-prod"}, Executors: []string{"container"}}, leaseUntil, now)
	if ok || claim.JobRunID != "" {
		t.Fatalf("group should not claim disallowed executor: ok=%v claim=%#v", ok, claim)
	}
	claim, ok = claimRecordJob(&record, runner, group, leaseUntil, now)
	if !ok || claim.PipelineRunID != "prun-group" {
		t.Fatalf("group should claim allowed run: ok=%v claim=%#v", ok, claim)
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
	runnerGroupUp := readMigration(t, "000015_runtime_runner_groups.up.sql")
	runnerGroupDown := readMigration(t, "000015_runtime_runner_groups.down.sql")

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
	if !strings.Contains(runnerGroupUp, "CREATE TABLE IF NOT EXISTS runtime_runner_groups") {
		t.Fatal("runner group migration missing runtime_runner_groups table")
	}
	if !strings.Contains(runnerGroupUp, "idx_runtime_runner_groups_project_id") {
		t.Fatal("runner group migration missing project index")
	}
	if !strings.Contains(runnerGroupDown, "DROP TABLE IF EXISTS runtime_runner_groups") {
		t.Fatal("runner group down migration missing table drop")
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
