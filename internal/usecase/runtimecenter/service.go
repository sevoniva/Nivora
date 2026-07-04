package runtimecenter

import (
	"context"
	"time"

	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
)

const (
	StatusHealthy  = "healthy"
	StatusWarning  = "warning"
	StatusDegraded = "degraded"
)

type Options struct {
	WorkerID    string
	StaleAfter  time.Duration
	Limit       int
	Pipeline    pipelineusecase.RuntimeRecoveryOptions
	PlanOnly    bool
	Reconcile   bool
	GeneratedAt time.Time
}

type Action struct {
	SubjectType    string    `json:"subjectType"`
	SubjectID      string    `json:"subjectId"`
	Status         string    `json:"status,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	SafeNextAction string    `json:"safeNextAction"`
	Automatic      bool      `json:"automatic"`
	ObservedAt     time.Time `json:"observedAt"`
}

type Summary struct {
	Status     string                                   `json:"status"`
	CheckedAt  time.Time                                `json:"checkedAt"`
	PlanOnly   bool                                     `json:"planOnly"`
	Reconciled bool                                     `json:"reconciled"`
	Pipeline   pipelineusecase.RuntimeRecoverySummary   `json:"pipeline"`
	Deployment deploymentusecase.RuntimeRecoverySummary `json:"deployment"`
	Release    releaseusecase.RuntimeRecoverySummary    `json:"release"`

	QueuedPipelineRuns           int      `json:"queuedPipelineRuns"`
	ProcessedPipelineRuns        int      `json:"processedPipelineRuns"`
	RecoveredPipelineRuns        int      `json:"recoveredPipelineRuns"`
	StaleRunningPipelineRuns     int      `json:"staleRunningPipelineRuns"`
	ExpiredJobClaims             int      `json:"expiredJobClaims"`
	CancelRequestedPipelineRuns  int      `json:"cancelRequestedPipelineRuns"`
	TimedOutPipelineRuns         int      `json:"timedOutPipelineRuns"`
	OfflineRunners               int      `json:"offlineRunners"`
	PendingOutboxEvents          int      `json:"pendingOutboxEvents"`
	FailedOutboxEvents           int      `json:"failedOutboxEvents"`
	PublishedOutboxEvents        int      `json:"publishedOutboxEvents"`
	NonTerminalDeploymentRuns    int      `json:"nonTerminalDeploymentRuns"`
	StaleDeploymentRuns          int      `json:"staleDeploymentRuns"`
	NonTerminalReleaseExecutions int      `json:"nonTerminalReleaseExecutions"`
	StaleReleaseExecutions       int      `json:"staleReleaseExecutions"`
	SafeNextActions              []Action `json:"safeNextActions,omitempty"`
	Warnings                     []string `json:"warnings,omitempty"`
}

type PipelineService interface {
	RuntimeStatus(ctx context.Context) (pipelineusecase.RuntimeRecoverySummary, error)
	ReconcileRuntime(ctx context.Context, options pipelineusecase.RuntimeRecoveryOptions) (pipelineusecase.RuntimeRecoverySummary, error)
}

type DeploymentService interface {
	RuntimeStatus(ctx context.Context, staleAfter time.Duration, limit int) (deploymentusecase.RuntimeRecoverySummary, error)
}

type ReleaseService interface {
	RuntimeStatus(ctx context.Context, staleAfter time.Duration, limit int) (releaseusecase.RuntimeRecoverySummary, error)
}

type Service struct {
	pipelines   PipelineService
	deployments DeploymentService
	releases    ReleaseService
	now         func() time.Time
}

func NewService(pipelines PipelineService, deployments DeploymentService, releases ReleaseService) *Service {
	return &Service{pipelines: pipelines, deployments: deployments, releases: releases, now: time.Now}
}

func (s *Service) Status(ctx context.Context, options Options) (Summary, error) {
	options = s.defaults(options)
	pipelineSummary, err := s.pipelines.RuntimeStatus(ctx)
	if err != nil {
		return Summary{}, err
	}
	return s.build(ctx, options, pipelineSummary, false)
}

func (s *Service) Reconcile(ctx context.Context, options Options) (Summary, error) {
	options = s.defaults(options)
	pipelineOptions := options.Pipeline
	if pipelineOptions.WorkerID == "" {
		pipelineOptions.WorkerID = options.WorkerID
	}
	if pipelineOptions.StaleAfter <= 0 {
		pipelineOptions.StaleAfter = options.StaleAfter
	}
	if pipelineOptions.ProcessLimit <= 0 {
		pipelineOptions.ProcessLimit = options.Limit
	}
	pipelineSummary, err := s.pipelines.ReconcileRuntime(ctx, pipelineOptions)
	if err != nil {
		return Summary{}, err
	}
	return s.build(ctx, options, pipelineSummary, true)
}

func (s *Service) build(ctx context.Context, options Options, pipelineSummary pipelineusecase.RuntimeRecoverySummary, reconciled bool) (Summary, error) {
	deploymentSummary, err := s.deployments.RuntimeStatus(ctx, options.StaleAfter, options.Limit)
	if err != nil {
		return Summary{}, err
	}
	releaseSummary, err := s.releases.RuntimeStatus(ctx, options.StaleAfter, options.Limit)
	if err != nil {
		return Summary{}, err
	}
	summary := Summary{
		Status:                       StatusHealthy,
		CheckedAt:                    options.GeneratedAt,
		PlanOnly:                     !reconciled,
		Reconciled:                   reconciled,
		Pipeline:                     pipelineSummary,
		Deployment:                   deploymentSummary,
		Release:                      releaseSummary,
		QueuedPipelineRuns:           pipelineSummary.QueuedPipelineRuns,
		ProcessedPipelineRuns:        pipelineSummary.ProcessedPipelineRuns,
		RecoveredPipelineRuns:        pipelineSummary.RecoveredPipelineRuns,
		StaleRunningPipelineRuns:     pipelineSummary.StaleRunningPipelineRuns,
		ExpiredJobClaims:             pipelineSummary.ExpiredJobClaims,
		CancelRequestedPipelineRuns:  pipelineSummary.CancelRequestedPipelineRuns,
		TimedOutPipelineRuns:         pipelineSummary.TimedOutPipelineRuns,
		OfflineRunners:               pipelineSummary.OfflineRunners,
		PendingOutboxEvents:          pipelineSummary.PendingOutboxEvents,
		FailedOutboxEvents:           pipelineSummary.FailedOutboxEvents,
		PublishedOutboxEvents:        pipelineSummary.PublishedOutboxEvents,
		NonTerminalDeploymentRuns:    deploymentSummary.NonTerminalDeploymentRuns,
		StaleDeploymentRuns:          deploymentSummary.StaleDeploymentRuns,
		NonTerminalReleaseExecutions: releaseSummary.NonTerminalReleaseExecutions,
		StaleReleaseExecutions:       releaseSummary.StaleReleaseExecutions,
		Warnings:                     append([]string(nil), pipelineSummary.Warnings...),
	}
	for _, action := range deploymentSummary.Actions {
		summary.SafeNextActions = append(summary.SafeNextActions, Action(action))
	}
	for _, action := range releaseSummary.Actions {
		summary.SafeNextActions = append(summary.SafeNextActions, Action(action))
	}
	summary.Warnings = append(summary.Warnings, deploymentSummary.Warnings...)
	summary.Warnings = append(summary.Warnings, releaseSummary.Warnings...)
	summary.Status = classify(summary)
	return summary, nil
}

func (s *Service) defaults(options Options) Options {
	if options.StaleAfter <= 0 {
		options.StaleAfter = 5 * time.Minute
	}
	if options.Limit <= 0 {
		options.Limit = 100
	}
	if options.WorkerID == "" {
		options.WorkerID = "runtime-center"
	}
	if options.GeneratedAt.IsZero() {
		if s.now != nil {
			options.GeneratedAt = s.now()
		} else {
			options.GeneratedAt = time.Now()
		}
	}
	return options
}

func classify(summary Summary) string {
	if summary.TimedOutPipelineRuns > 0 || summary.FailedOutboxEvents > 0 {
		return StatusDegraded
	}
	if summary.StaleRunningPipelineRuns > 0 ||
		summary.ExpiredJobClaims > 0 ||
		summary.CancelRequestedPipelineRuns > 0 ||
		summary.OfflineRunners > 0 ||
		summary.StaleDeploymentRuns > 0 ||
		summary.StaleReleaseExecutions > 0 ||
		summary.PendingOutboxEvents > 0 {
		return StatusWarning
	}
	return StatusHealthy
}
