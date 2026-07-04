package deployment

import (
	"context"
	"time"
)

type RuntimeRecoveryAction struct {
	SubjectType    string    `json:"subjectType"`
	SubjectID      string    `json:"subjectId"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason,omitempty"`
	SafeNextAction string    `json:"safeNextAction"`
	Automatic      bool      `json:"automatic"`
	ObservedAt     time.Time `json:"observedAt"`
}

type RuntimeRecoverySummary struct {
	NonTerminalDeploymentRuns int                     `json:"nonTerminalDeploymentRuns"`
	StaleDeploymentRuns       int                     `json:"staleDeploymentRuns"`
	Actions                   []RuntimeRecoveryAction `json:"actions,omitempty"`
	Warnings                  []string                `json:"warnings,omitempty"`
}

func (s *Service) RuntimeStatus(ctx context.Context, staleAfter time.Duration, limit int) (RuntimeRecoverySummary, error) {
	if staleAfter <= 0 {
		staleAfter = 5 * time.Minute
	}
	if limit <= 0 {
		limit = 100
	}
	now := s.now()
	nonTerminal, err := s.store.ListNonTerminalDeploymentRuns(ctx, limit)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	stale, err := s.store.ListStaleDeploymentRuns(ctx, now.Add(-staleAfter), limit)
	if err != nil {
		return RuntimeRecoverySummary{}, err
	}
	summary := RuntimeRecoverySummary{
		NonTerminalDeploymentRuns: len(nonTerminal),
		StaleDeploymentRuns:       len(stale),
	}
	for _, record := range stale {
		summary.Actions = append(summary.Actions, RuntimeRecoveryAction{
			SubjectType:    "deploymentRun",
			SubjectID:      record.Run.ID,
			Status:         string(record.Run.Status),
			Reason:         record.Run.Reason,
			SafeNextAction: "Inspect deployment timeline, health, logs, and rollback plan before manually resuming or canceling.",
			Automatic:      false,
			ObservedAt:     now,
		})
	}
	return summary, nil
}
