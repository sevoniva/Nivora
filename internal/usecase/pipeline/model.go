package pipeline

import (
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
)

type RunRecord struct {
	Pipeline    domainpipeline.Pipeline    `json:"pipeline"`
	Run         domainpipeline.PipelineRun `json:"run"`
	Definition  Definition                 `json:"definition,omitempty"`
	Stages      []StageRecord              `json:"stages"`
	Logs        []event.LogChunk           `json:"logs,omitempty"`
	Events      []event.Event              `json:"events,omitempty"`
	Audits      []audit.AuditLog           `json:"audits,omitempty"`
	Artifacts   []PipelineArtifact         `json:"artifacts,omitempty"`
	Caches      []PipelineCacheEntry       `json:"caches,omitempty"`
	Annotations []StepAnnotation           `json:"annotations,omitempty"`
	Summaries   []StepSummary              `json:"summaries,omitempty"`
}

type StageRecord struct {
	Stage domainpipeline.StageRun `json:"stage"`
	Jobs  []JobRecord             `json:"jobs"`
}

type JobRecord struct {
	Job   domainpipeline.JobRun    `json:"job"`
	Steps []domainpipeline.StepRun `json:"steps"`
}

type TimelineEntry struct {
	Type    string            `json:"type"`
	Time    time.Time         `json:"time"`
	Subject string            `json:"subject"`
	Status  string            `json:"status,omitempty"`
	Message string            `json:"message,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

type PipelineArtifact struct {
	ID            string            `json:"id"`
	PipelineRunID string            `json:"pipelineRunId"`
	StageRunID    string            `json:"stageRunId,omitempty"`
	JobRunID      string            `json:"jobRunId,omitempty"`
	StepRunID     string            `json:"stepRunId,omitempty"`
	Name          string            `json:"name"`
	Type          string            `json:"type,omitempty"`
	SizeBytes     int64             `json:"sizeBytes,omitempty"`
	ContentHash   string            `json:"contentHash,omitempty"`
	StorageRef    string            `json:"storageRef,omitempty"`
	RetentionDays int               `json:"retentionDays,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

type PipelineCacheEntry struct {
	ID            string            `json:"id"`
	PipelineRunID string            `json:"pipelineRunId"`
	JobRunID      string            `json:"jobRunId,omitempty"`
	StepRunID     string            `json:"stepRunId,omitempty"`
	Key           string            `json:"key"`
	RestoreKeys   []string          `json:"restoreKeys,omitempty"`
	Scope         string            `json:"scope,omitempty"`
	Hit           bool              `json:"hit"`
	SizeBytes     int64             `json:"sizeBytes,omitempty"`
	StorageRef    string            `json:"storageRef,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	ExpiresAt     *time.Time        `json:"expiresAt,omitempty"`
}

type StepAnnotation struct {
	ID            string            `json:"id"`
	PipelineRunID string            `json:"pipelineRunId"`
	StageRunID    string            `json:"stageRunId,omitempty"`
	JobRunID      string            `json:"jobRunId,omitempty"`
	StepRunID     string            `json:"stepRunId,omitempty"`
	Level         string            `json:"level"`
	File          string            `json:"file,omitempty"`
	Line          int               `json:"line,omitempty"`
	Column        int               `json:"column,omitempty"`
	Title         string            `json:"title,omitempty"`
	Message       string            `json:"message"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

type StepSummary struct {
	ID            string            `json:"id"`
	PipelineRunID string            `json:"pipelineRunId"`
	StageRunID    string            `json:"stageRunId,omitempty"`
	JobRunID      string            `json:"jobRunId,omitempty"`
	StepRunID     string            `json:"stepRunId,omitempty"`
	Title         string            `json:"title,omitempty"`
	Content       string            `json:"content,omitempty"`
	StorageRef    string            `json:"storageRef,omitempty"`
	Format        string            `json:"format,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type PipelineRunSummary struct {
	PipelineRunID   string               `json:"pipelineRunId"`
	Status          string               `json:"status"`
	ArtifactCount   int                  `json:"artifactCount"`
	CacheCount      int                  `json:"cacheCount"`
	AnnotationCount int                  `json:"annotationCount"`
	SummaryCount    int                  `json:"summaryCount"`
	Artifacts       []PipelineArtifact   `json:"artifacts,omitempty"`
	Caches          []PipelineCacheEntry `json:"caches,omitempty"`
	Annotations     []StepAnnotation     `json:"annotations,omitempty"`
	Summaries       []StepSummary        `json:"summaries,omitempty"`
	GeneratedAt     time.Time            `json:"generatedAt"`
}

type RunnerRecord struct {
	Runner domainrunner.Runner `json:"runner"`
}

type RunnerToken struct {
	TokenID  string    `json:"tokenId"`
	Token    string    `json:"token"`
	IssuedAt time.Time `json:"issuedAt"`
}

type RegisterRunnerResult struct {
	Runner domainrunner.Runner `json:"runner"`
	Token  RunnerToken         `json:"token"`
}

type JobClaim struct {
	PipelineRunID   string                      `json:"pipelineRunId"`
	StageRunID      string                      `json:"stageRunId"`
	JobRunID        string                      `json:"jobRunId"`
	StepRunIDs      []string                    `json:"stepRunIds,omitempty"`
	RunnerID        string                      `json:"runnerId"`
	Executor        string                      `json:"executor"`
	Commands        []string                    `json:"commands,omitempty"`
	Attempt         int                         `json:"attempt"`
	LeaseExpiresAt  time.Time                   `json:"leaseExpiresAt"`
	CancelRequested bool                        `json:"cancelRequested,omitempty"`
	Status          domainpipeline.JobRunStatus `json:"status"`
}

type AppendJobLogInput struct {
	PipelineRunID string `json:"pipelineRunId"`
	StageRunID    string `json:"stageRunId,omitempty"`
	StepRunID     string `json:"stepRunId,omitempty"`
	Stream        string `json:"stream"`
	Content       string `json:"content"`
}

type UpdateJobStatusInput struct {
	Status domainpipeline.JobRunStatus `json:"status"`
	Reason string                      `json:"reason,omitempty"`
}

type EventOutboxRecord struct {
	ID            string      `json:"id"`
	EventType     string      `json:"eventType"`
	Subject       string      `json:"subject"`
	Payload       event.Event `json:"payload"`
	Status        string      `json:"status"`
	RetryCount    int         `json:"retryCount,omitempty"`
	NextAttemptAt *time.Time  `json:"nextAttemptAt,omitempty"`
	LastError     string      `json:"lastError,omitempty"`
	CreatedAt     time.Time   `json:"createdAt"`
	PublishedAt   *time.Time  `json:"publishedAt,omitempty"`
}

type RuntimeRecoveryOptions struct {
	WorkerID      string
	LeaseDuration time.Duration
	StaleAfter    time.Duration
	TimeoutAfter  time.Duration
	ProcessLimit  int
	OutboxLimit   int
}

type RuntimeRecoverySummary struct {
	WorkerID                    string    `json:"workerId"`
	QueuedPipelineRuns          int       `json:"queuedPipelineRuns"`
	ProcessedPipelineRuns       int       `json:"processedPipelineRuns"`
	RecoveredPipelineRuns       int       `json:"recoveredPipelineRuns"`
	StaleRunningPipelineRuns    int       `json:"staleRunningPipelineRuns"`
	ExpiredJobClaims            int       `json:"expiredJobClaims"`
	CancelRequestedPipelineRuns int       `json:"cancelRequestedPipelineRuns"`
	TimedOutPipelineRuns        int       `json:"timedOutPipelineRuns"`
	OfflineRunners              int       `json:"offlineRunners"`
	PendingOutboxEvents         int       `json:"pendingOutboxEvents"`
	PublishedOutboxEvents       int       `json:"publishedOutboxEvents"`
	FailedOutboxEvents          int       `json:"failedOutboxEvents"`
	CheckedAt                   time.Time `json:"checkedAt"`
	Warnings                    []string  `json:"warnings,omitempty"`
}

type CreateRunInput struct {
	Definition        Definition
	ProjectID         string
	EnvironmentID     string
	PipelineID        string
	PipelineVersionID string
	ActorID           string
	CorrelationID     string
	Workflow          WorkflowRunMetadata
}

type WorkflowRunMetadata struct {
	WorkflowID           string `json:"workflowId,omitempty"`
	WorkflowPlanID       string `json:"workflowPlanId,omitempty"`
	WorkflowRunID        string `json:"workflowRunId,omitempty"`
	RepositoryID         string `json:"repositoryId,omitempty"`
	RepositorySnapshotID string `json:"repositorySnapshotId,omitempty"`
	SourcePath           string `json:"sourcePath,omitempty"`
	Ref                  string `json:"ref,omitempty"`
}

type CreateRunResult struct {
	Record RunRecord
}
