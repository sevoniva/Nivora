package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	"github.com/sevoniva/nivora/internal/domain/release"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
)

func TestPostgresIntegrationMigrationUpDown(t *testing.T) {
	db := newPostgresIntegration(t, false)
	defer db.cleanup()

	applyUpMigrations(t, db.pool)
	for _, table := range []string{
		"runtime_pipeline_runs",
		"runtime_job_runs",
		"runtime_event_outbox",
		"runtime_deployment_runs",
		"runtime_deployment_logs",
		"runtime_releases",
		"runtime_release_artifacts",
		"runtime_release_plans",
		"runtime_release_executions",
		"runtime_release_execution_targets",
		"compliance_evidence_bundles",
		"compliance_retention_policies",
		"compliance_audit_records",
	} {
		assertRelationExists(t, db.pool, table)
	}
	for _, index := range []string{
		"idx_runtime_pipeline_runs_status_created_at",
		"idx_runtime_job_runs_lease",
		"idx_runtime_deployment_runs_status_created_at",
		"idx_runtime_release_executions_status_created_at",
		"idx_compliance_evidence_subject",
		"idx_compliance_audit_subject",
	} {
		assertRelationExists(t, db.pool, index)
	}

	applyDownMigrations(t, db.pool)
	for _, table := range []string{"runtime_pipeline_runs", "runtime_deployment_runs", "runtime_release_executions", "compliance_evidence_bundles"} {
		assertRelationMissing(t, db.pool, table)
	}
}

func TestPostgresIntegrationPipelineRunRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	now := fixedIntegrationTime()
	store := NewPipelineStore(db.pool)

	running := pipelineRecord("prun-recover-running", domainpipeline.PipelineRunRunning, now)
	lease := now.Add(-time.Minute)
	running.Run.OwnerID = "worker-before-restart"
	running.Run.LeaseExpiresAt = &lease
	running.Run.HeartbeatAt = &lease
	running.Run.CancelRequested = true
	if err := store.Save(ctx, running); err != nil {
		t.Fatalf("save running pipeline: %v", err)
	}
	if err := store.AppendLog(ctx, running.Run.ID, event.LogChunk{ID: "plog-1", PipelineRunID: running.Run.ID, JobRunID: "job-prun-recover-running", Stream: "stdout", Content: "persisted log", CreatedAt: now}); err != nil {
		t.Fatalf("append pipeline log: %v", err)
	}
	if err := store.AppendEvent(ctx, running.Run.ID, testEvent("pevt-1", "devops.pipeline.run.started", running.Run.ID, now)); err != nil {
		t.Fatalf("append pipeline event: %v", err)
	}
	if err := store.AppendAudit(ctx, running.Run.ID, audit.AuditLog{ID: "paudit-1", Action: "pipeline persisted", Subject: running.Run.ID, CreatedAt: now}); err != nil {
		t.Fatalf("append pipeline audit: %v", err)
	}
	queued := pipelineRecord("prun-recover-queued", domainpipeline.PipelineRunQueued, now)
	if err := store.Save(ctx, queued); err != nil {
		t.Fatalf("save queued pipeline: %v", err)
	}

	store = NewPipelineStore(db.restart(t))
	loaded, err := store.Get(ctx, running.Run.ID)
	if err != nil {
		t.Fatalf("reload running pipeline after restart: %v", err)
	}
	if !loaded.Run.CancelRequested || loaded.Run.OwnerID != "worker-before-restart" {
		t.Fatalf("pipeline recovery lost lease/cancel state: %#v", loaded.Run)
	}
	logs, err := store.LogsByPipelineRun(ctx, running.Run.ID)
	if err != nil || len(logs) != 1 || logs[0].Content != "persisted log" {
		t.Fatalf("pipeline logs after restart = %#v err=%v", logs, err)
	}
	events, err := store.EventsByPipelineRun(ctx, running.Run.ID)
	if err != nil || len(events) != 1 || events[0].ID != "pevt-1" {
		t.Fatalf("pipeline events after restart = %#v err=%v", events, err)
	}
	audits, err := store.AuditBySubject(ctx, running.Run.ID)
	if err != nil || len(audits) != 1 || audits[0].ID != "paudit-1" {
		t.Fatalf("pipeline audits after restart = %#v err=%v", audits, err)
	}
	queuedRuns, err := store.ListQueuedPipelineRuns(ctx, 10)
	if err != nil || len(queuedRuns) != 1 || queuedRuns[0].Run.ID != queued.Run.ID {
		t.Fatalf("queued recovery query = %#v err=%v", queuedRuns, err)
	}
	staleRuns, err := store.ListStaleRunningPipelineRuns(ctx, now, 10)
	if err != nil || len(staleRuns) != 1 || staleRuns[0].Run.ID != running.Run.ID {
		t.Fatalf("stale running recovery query = %#v err=%v", staleRuns, err)
	}
}

func TestPostgresIntegrationDeploymentRunRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	now := fixedIntegrationTime()
	store := NewDeploymentStore(db.pool)

	record := deploymentRecord("drun-recover", now)
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("save deployment: %v", err)
	}
	if err := store.AppendLog(ctx, record.Run.ID, event.LogChunk{ID: "dlog-1", Stream: "stdout", Content: "deployment log", CreatedAt: now}); err != nil {
		t.Fatalf("append deployment log: %v", err)
	}
	if err := store.AppendEvent(ctx, record.Run.ID, testEvent("devt-1", "devops.deployment.created", record.Run.ID, now)); err != nil {
		t.Fatalf("append deployment event: %v", err)
	}
	if err := store.AppendAudit(ctx, record.Run.ID, audit.AuditLog{ID: "daudit-1", Action: "deployment persisted", Subject: record.Run.ID, CreatedAt: now}); err != nil {
		t.Fatalf("append deployment audit: %v", err)
	}

	store = NewDeploymentStore(db.restart(t))
	loaded, err := store.Get(ctx, record.Run.ID)
	if err != nil {
		t.Fatalf("reload deployment after restart: %v", err)
	}
	if loaded.Plan.DeploymentRunID != record.Run.ID || loaded.Snapshot.ID != "snapshot-drun-recover" || loaded.RollbackPlan.CurrentSnapshotID != "snapshot-drun-recover" {
		t.Fatalf("deployment recovery lost plan/snapshot/rollback: %#v", loaded)
	}
	if len(loaded.Inventory.Desired) != 1 || loaded.Inventory.Desired[0].Name != "demo" {
		t.Fatalf("deployment inventory after restart = %#v", loaded.Inventory)
	}
	logs, err := store.Logs(ctx, record.Run.ID)
	if err != nil || len(logs) != 1 || logs[0].Content != "deployment log" {
		t.Fatalf("deployment logs after restart = %#v err=%v", logs, err)
	}
	events, err := store.Events(ctx, record.Run.ID)
	if err != nil || len(events) != 1 || events[0].ID != "devt-1" {
		t.Fatalf("deployment events after restart = %#v err=%v", events, err)
	}
	audits, err := store.Audits(ctx, record.Run.ID)
	if err != nil || len(audits) != 1 || audits[0].ID != "daudit-1" {
		t.Fatalf("deployment audits after restart = %#v err=%v", audits, err)
	}
	nonTerminal, err := store.ListNonTerminalDeploymentRuns(ctx, 10)
	if err != nil || len(nonTerminal) != 1 || nonTerminal[0].Run.ID != record.Run.ID {
		t.Fatalf("non-terminal deployments = %#v err=%v", nonTerminal, err)
	}
	stale, err := store.ListStaleDeploymentRuns(ctx, now, 10)
	if err != nil || len(stale) != 1 || stale[0].Run.ID != record.Run.ID {
		t.Fatalf("stale deployments = %#v err=%v", stale, err)
	}
}

func TestPostgresIntegrationReleaseExecutionRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	now := fixedIntegrationTime()
	releaseStore := NewReleaseStore(db.pool)
	orchestrationStore := NewReleaseOrchestrationStore(db.pool)

	releaseRecord := artifactusecase.ReleaseRecord{
		Release:   release.Release{ID: "rel-recover", Name: "recover", Version: "1.0.0", ApplicationID: "app", EnvironmentID: "dev", Status: "Created", CreatedAt: now, UpdatedAt: now},
		Artifacts: []domainartifact.Artifact{{ID: "artifact-1", Type: domainartifact.ArtifactTypeImage, Name: "demo", Reference: "registry.example.com/demo/app:1.0.0", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", CreatedAt: now}},
		Bindings:  []release.ReleaseArtifact{{ID: "binding-1", ReleaseID: "rel-recover", ArtifactID: "artifact-1", Name: "demo", Type: "image", Reference: "registry.example.com/demo/app:1.0.0", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", CreatedAt: now, UpdatedAt: now}},
		Events:    []event.Event{testEvent("revt-1", "devops.release.created", "rel-recover", now)},
		Audits:    []audit.AuditLog{{ID: "raudit-1", Action: "release persisted", Subject: "rel-recover", CreatedAt: now}},
	}
	if err := releaseStore.SaveRelease(ctx, releaseRecord); err != nil {
		t.Fatalf("save release: %v", err)
	}

	planRecord, executionRecord := releaseRecords(now)
	if err := orchestrationStore.SavePlan(ctx, planRecord); err != nil {
		t.Fatalf("save release plan: %v", err)
	}
	if err := orchestrationStore.SaveExecution(ctx, executionRecord); err != nil {
		t.Fatalf("save release execution: %v", err)
	}
	if err := orchestrationStore.AppendEvent(ctx, executionRecord.Execution.ID, testEvent("reevt-1", "devops.release.execution.started", executionRecord.Execution.ID, now)); err != nil {
		t.Fatalf("append release execution event: %v", err)
	}
	if err := orchestrationStore.AppendAudit(ctx, executionRecord.Execution.ID, audit.AuditLog{ID: "reaudit-1", Action: "release execution persisted", Subject: executionRecord.Execution.ID, CreatedAt: now}); err != nil {
		t.Fatalf("append release execution audit: %v", err)
	}

	releaseStore = NewReleaseStore(db.restart(t))
	orchestrationStore = NewReleaseOrchestrationStore(db.pool)
	loadedRelease, err := releaseStore.GetRelease(ctx, releaseRecord.Release.ID)
	if err != nil {
		t.Fatalf("reload release after restart: %v", err)
	}
	if len(loadedRelease.Bindings) != 1 || loadedRelease.Bindings[0].Digest == "" {
		t.Fatalf("release artifacts after restart = %#v", loadedRelease.Bindings)
	}
	loadedPlan, err := orchestrationStore.GetPlan(ctx, planRecord.Plan.ID)
	if err != nil || loadedPlan.Plan.ReleaseID != planRecord.Plan.ReleaseID {
		t.Fatalf("release plan after restart = %#v err=%v", loadedPlan, err)
	}
	loadedExecution, err := orchestrationStore.GetExecution(ctx, executionRecord.Execution.ID)
	if err != nil {
		t.Fatalf("reload release execution after restart: %v", err)
	}
	if loadedExecution.Execution.Status != releaseorchestration.ExecutionRunning || len(loadedExecution.Execution.Targets) != 1 {
		t.Fatalf("release execution state after restart = %#v", loadedExecution.Execution)
	}
	nonTerminal, err := orchestrationStore.ListNonTerminalReleaseExecutions(ctx, 10)
	if err != nil || len(nonTerminal) != 1 || nonTerminal[0].Execution.ID != executionRecord.Execution.ID {
		t.Fatalf("non-terminal release executions = %#v err=%v", nonTerminal, err)
	}
	stale, err := orchestrationStore.ListStaleReleaseExecutions(ctx, now, 10)
	if err != nil || len(stale) != 1 || stale[0].Execution.ID != executionRecord.Execution.ID {
		t.Fatalf("stale release executions = %#v err=%v", stale, err)
	}
	events, err := orchestrationStore.Events(ctx, executionRecord.Execution.ID)
	if err != nil || len(events) != 1 || events[0].ID != "reevt-1" {
		t.Fatalf("release execution events = %#v err=%v", events, err)
	}
	audits, err := orchestrationStore.Audits(ctx, executionRecord.Execution.ID)
	if err != nil || len(audits) != 1 || audits[0].ID != "reaudit-1" {
		t.Fatalf("release execution audits = %#v err=%v", audits, err)
	}
	first, inserted, err := orchestrationStore.RecordIdempotencyKey(ctx, "release-execution", "idem-1", IdempotencyResult{ResourceType: "release_execution", ResourceID: executionRecord.Execution.ID, RequestHash: "hash-1", CreatedAt: now})
	if err != nil || !inserted || first.ResourceID != executionRecord.Execution.ID {
		t.Fatalf("record idempotency first = %#v inserted=%v err=%v", first, inserted, err)
	}
	second, inserted, err := orchestrationStore.RecordIdempotencyKey(ctx, "release-execution", "idem-1", IdempotencyResult{ResourceType: "release_execution", ResourceID: "different", RequestHash: "hash-2", CreatedAt: now})
	if err != nil || inserted || second.ResourceID != executionRecord.Execution.ID {
		t.Fatalf("record idempotency replay = %#v inserted=%v err=%v", second, inserted, err)
	}
}

func TestPostgresIntegrationRunnerClaimRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	now := fixedIntegrationTime()
	store := NewPipelineStore(db.pool)
	store.now = func() time.Time { return now }

	if err := store.RegisterRunner(ctx, domainrunner.Runner{ID: "runner-recover", Name: "runner-recover", Status: "online", Executors: []string{"shell"}, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("register runner: %v", err)
	}
	record := pipelineRecord("prun-runner-claim", domainpipeline.PipelineRunQueued, now)
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("save claimable pipeline: %v", err)
	}
	claim, err := store.ClaimJob(ctx, "runner-recover", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if claim.JobRunID != "job-prun-runner-claim" {
		t.Fatalf("unexpected claim = %#v", claim)
	}

	store = NewPipelineStore(db.restart(t))
	store.now = func() time.Time { return now.Add(30 * time.Second) }
	_, err = store.ClaimJob(ctx, "runner-recover", now.Add(2*time.Minute))
	if !errors.Is(err, pipelineusecase.ErrNoClaimableJob) {
		t.Fatalf("claim before lease expiry err = %v, want no claimable job", err)
	}
	expired, err := store.ListExpiredJobClaims(ctx, now.Add(2*time.Minute), 10)
	if err != nil || len(expired) != 1 || expired[0].JobRunID != claim.JobRunID {
		t.Fatalf("expired claims = %#v err=%v", expired, err)
	}
	store.now = func() time.Time { return now.Add(2 * time.Minute) }
	reclaimed, err := store.ClaimJob(ctx, "runner-recover", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("reclaim expired job: %v", err)
	}
	if reclaimed.JobRunID != claim.JobRunID || !reclaimed.LeaseExpiresAt.Equal(now.Add(3*time.Minute)) {
		t.Fatalf("reclaimed job = %#v", reclaimed)
	}
}

func TestPostgresIntegrationEventOutboxRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	now := fixedIntegrationTime()
	store := NewPipelineStore(db.pool)

	pending := pipelineusecase.EventOutboxRecord{ID: "outbox-pending", EventType: "devops.test", Subject: "subject-1", Payload: testEvent("outbox-event-1", "devops.test", "subject-1", now), Status: "pending", CreatedAt: now}
	failedDue := pipelineusecase.EventOutboxRecord{ID: "outbox-failed", EventType: "devops.test", Subject: "subject-2", Payload: testEvent("outbox-event-2", "devops.test", "subject-2", now), Status: "failed", RetryCount: 1, NextAttemptAt: ptrTime(now.Add(-time.Minute)), LastError: "temporary", CreatedAt: now.Add(time.Second)}
	if err := store.AppendOutbox(ctx, pending); err != nil {
		t.Fatalf("append pending outbox: %v", err)
	}
	if err := store.AppendOutbox(ctx, failedDue); err != nil {
		t.Fatalf("append failed outbox: %v", err)
	}

	store = NewPipelineStore(db.restart(t))
	items, err := store.ListPendingOutbox(ctx, 10)
	if err != nil || len(items) != 2 {
		t.Fatalf("pending outbox after restart = %#v err=%v", items, err)
	}
	if err := store.MarkOutboxPublished(ctx, pending.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	if err := store.MarkOutboxFailed(ctx, failedDue.ID, 2, time.Now().Add(time.Hour), "retry later"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	items, err = store.ListPendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("list pending after updates: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("outbox published/future retry should not be pending: %#v", items)
	}
}

type postgresIntegration struct {
	admin       *pgxpool.Pool
	pool        *pgxpool.Pool
	databaseURL string
	schema      string
}

func newPostgresIntegration(t *testing.T, migrate bool) *postgresIntegration {
	t.Helper()
	if os.Getenv("NIVORA_RUN_POSTGRES_INTEGRATION") != "true" {
		t.Skip("set NIVORA_RUN_POSTGRES_INTEGRATION=true and DATABASE_URL to run PostgreSQL integration tests")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when NIVORA_RUN_POSTGRES_INTEGRATION=true")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect admin postgres: %v", err)
	}
	schema := fmt.Sprintf("nivora_it_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatalf("create schema: %v", err)
	}
	pool, err := pgxpool.New(ctx, postgresURLWithSearchPath(t, databaseURL, schema))
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close()
		t.Fatalf("connect schema postgres: %v", err)
	}
	db := &postgresIntegration{admin: admin, pool: pool, databaseURL: databaseURL, schema: schema}
	if migrate {
		applyUpMigrations(t, pool)
	}
	return db
}

func (db *postgresIntegration) cleanup() {
	if db.pool != nil {
		db.pool.Close()
	}
	if db.admin != nil {
		_, _ = db.admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+db.schema+" CASCADE")
		db.admin.Close()
	}
}

func (db *postgresIntegration) restart(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db.pool.Close()
	pool, err := pgxpool.New(context.Background(), postgresURLWithSearchPath(t, db.databaseURL, db.schema))
	if err != nil {
		t.Fatalf("restart postgres pool: %v", err)
	}
	db.pool = pool
	return pool
}

func postgresURLWithSearchPath(t *testing.T, raw string, schema string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	q := u.Query()
	q.Set("options", "-c search_path="+schema)
	u.RawQuery = q.Encode()
	return u.String()
}

func applyUpMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, path := range migrationFiles(t, "*.up.sql", false) {
		execMigration(t, pool, path)
	}
}

func applyDownMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, path := range migrationFiles(t, "*.down.sql", true) {
		execMigration(t, pool, path)
	}
}

func migrationFiles(t *testing.T, pattern string, reverse bool) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "infra", "migration", pattern))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)
	if reverse {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}
	return files
}

func execMigration(t *testing.T, pool *pgxpool.Pool, path string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	if _, err := pool.Exec(context.Background(), string(body)); err != nil {
		t.Fatalf("execute migration %s: %v", filepath.Base(path), err)
	}
}

func assertRelationExists(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	if !relationExists(t, pool, name) {
		t.Fatalf("expected relation %s to exist", name)
	}
}

func assertRelationMissing(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	if relationExists(t, pool, name) {
		t.Fatalf("expected relation %s to be dropped", name)
	}
}

func relationExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `SELECT to_regclass($1) IS NOT NULL`, name).Scan(&exists); err != nil {
		t.Fatalf("check relation %s: %v", name, err)
	}
	return exists
}

func fixedIntegrationTime() time.Time {
	return time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
}

func ptrTime(t time.Time) *time.Time { return &t }

func testEvent(id, eventType, subject string, at time.Time) event.Event {
	return event.Event{SpecVersion: "1.0", ID: id, Type: eventType, Source: "nivora.integration-test", Subject: subject, Time: at, DataContentType: "application/json", Data: map[string]any{"status": "test"}}
}

func pipelineRecord(id string, status domainpipeline.PipelineRunStatus, now time.Time) pipelineusecase.RunRecord {
	return pipelineusecase.RunRecord{
		Pipeline: pipelineusecasePipeline(id, now),
		Run:      domainpipeline.PipelineRun{ID: id, PipelineID: "pipeline-" + id, Status: status, CorrelationID: "corr-" + id, Attempt: 1, CreatedAt: now, UpdatedAt: now},
		Definition: pipelineusecase.Definition{Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
			Name: "build",
			Jobs: []pipelineusecase.Job{{Name: "echo", Executor: "shell", Steps: []pipelineusecase.Step{{Name: "say", Run: "printf hello"}}}},
		}}}},
		Stages: []pipelineusecase.StageRecord{{
			Stage: domainpipeline.StageRun{ID: "stage-" + id, PipelineRunID: id, Name: "build", Status: domainpipeline.JobRunPending, CreatedAt: now, UpdatedAt: now},
			Jobs: []pipelineusecase.JobRecord{{
				Job:   domainpipeline.JobRun{ID: "job-" + id, StageRunID: "stage-" + id, Name: "echo", Status: domainpipeline.JobRunPending, Attempt: 1, CreatedAt: now, UpdatedAt: now},
				Steps: []domainpipeline.StepRun{{ID: "step-" + id, JobRunID: "job-" + id, Name: "say", Status: domainpipeline.JobRunPending, Attempt: 1, CreatedAt: now, UpdatedAt: now}},
			}},
		}},
	}
}

func pipelineusecasePipeline(id string, now time.Time) domainpipeline.Pipeline {
	return domainpipeline.Pipeline{ID: "pipeline-" + id, Name: "pipeline-" + id, CreatedAt: now, UpdatedAt: now}
}

func deploymentRecord(id string, now time.Time) deploymentusecase.RunRecord {
	lease := now.Add(-time.Minute)
	resource := deploymentusecase.ManifestResourceSummary{APIVersion: "apps/v1", Group: "apps", Version: "v1", Kind: "Deployment", Namespace: "default", Name: "demo", SourceFile: "examples/yaml/deployment.yaml", Index: 0, DesiredHash: "sha256:test", Health: deploymentusecase.ResourceHealthProgressing, CreatedAt: now, UpdatedAt: now}
	return deploymentusecase.RunRecord{
		Release:     release.Release{ID: "rel-" + id, Name: "release-" + id, Version: "1.0.0", CreatedAt: now, UpdatedAt: now},
		Environment: environment.Environment{ID: "dev", Name: "dev", CreatedAt: now, UpdatedAt: now},
		Target:      environment.ReleaseTarget{ID: "target-" + id, EnvironmentID: "dev", Name: "dev-yaml", TargetType: "kubernetes-yaml", CreatedAt: now, UpdatedAt: now},
		Run: domaindeployment.DeploymentRun{
			ID:                  id,
			ReleaseID:           "rel-" + id,
			ApplicationID:       "app-demo",
			EnvironmentID:       "dev",
			ReleaseTargetID:     "target-" + id,
			TargetType:          "kubernetes-yaml",
			Status:              domaindeployment.DeploymentRunDeploying,
			CorrelationID:       "corr-" + id,
			OwnerID:             "worker-before-restart",
			LeaseExpiresAt:      &lease,
			Attempt:             1,
			HeartbeatAt:         &lease,
			ManifestSnapshotRef: "snapshot-" + id,
			CreatedAt:           now,
			UpdatedAt:           lease,
		},
		Plan:         deploymentusecase.DeploymentPlan{DeploymentRunID: id, TargetType: "kubernetes-yaml", Namespace: "default", ManifestCount: 1, Resources: []deploymentusecase.ManifestResourceSummary{resource}, DryRun: true, Apply: false, Actions: []string{"render", "validate"}, DiffSummary: "desired state only"},
		Inventory:    deploymentusecase.ResourceInventory{DeploymentRunID: id, Desired: []deploymentusecase.ManifestResourceSummary{resource}, CreatedAt: now, UpdatedAt: now},
		Snapshot:     deploymentusecase.ManifestSnapshot{ID: "snapshot-" + id, DeploymentRunID: id, ContentHash: "sha256:test", DocumentCount: 1, ResourceCount: 1, StorageRef: "memory://" + id, CreatedAt: now},
		RollbackPlan: deploymentusecase.RollbackPlan{DeploymentRunID: id, CurrentSnapshotID: "snapshot-" + id, TargetType: "kubernetes-yaml", TargetName: "dev-yaml", Resources: []deploymentusecase.ManifestResourceSummary{resource}, Strategy: "manifest-restore", Executable: false, CreatedAt: now},
	}
}

func releaseRecords(now time.Time) (releaseorchestration.PlanRecord, releaseorchestration.ExecutionRecord) {
	rel := release.Release{ID: "rel-recover", Name: "recover", Version: "1.0.0", ApplicationID: "app", EnvironmentID: "dev", CreatedAt: now, UpdatedAt: now}
	plan := releaseorchestration.ReleasePlan{ID: "rplan-recover", ReleaseID: rel.ID, EnvironmentID: "dev", EnvironmentName: "dev", Targets: []environment.ReleaseTarget{{ID: "target-yaml", Name: "yaml", TargetType: "kubernetes-yaml", EnvironmentID: "dev", CreatedAt: now, UpdatedAt: now}}, Strategy: releaseorchestration.StrategySequential, Concurrency: 1, Ordering: []string{"target-yaml"}, CreatedAt: now}
	lease := now.Add(-time.Minute)
	execution := releaseorchestration.ReleaseExecution{ID: "rexec-recover", ReleaseID: rel.ID, EnvironmentID: "dev", EnvironmentName: "dev", Status: releaseorchestration.ExecutionRunning, CorrelationID: "corr-rexec", OwnerID: "worker-before-restart", LeaseExpiresAt: &lease, Attempt: 1, HeartbeatAt: &lease, Targets: []releaseorchestration.TargetExecution{{TargetID: "target-yaml", TargetName: "yaml", TargetType: "kubernetes-yaml", DeploymentRunID: "drun-recover", Status: releaseorchestration.ExecutionRunning, Order: 1}}, CreatedAt: now, UpdatedAt: lease}
	return releaseorchestration.PlanRecord{Release: rel, Plan: plan}, releaseorchestration.ExecutionRecord{Release: rel, Plan: plan, Execution: execution}
}
