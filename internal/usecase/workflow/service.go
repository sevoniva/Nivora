package workflow

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("workflow record not found")

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{store: store, now: time.Now}
}

func (s *Service) Plan(ctx context.Context, input PlanInput) (PlanRecord, error) {
	if err := ctx.Err(); err != nil {
		return PlanRecord{}, err
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return PlanRecord{}, fmt.Errorf("%w: workflow content is required", ErrInvalid)
	}
	def, err := ParseDefinition([]byte(content))
	if err != nil {
		return PlanRecord{}, err
	}
	plan, err := PlanDefinition(def, input.Options)
	if err != nil {
		return PlanRecord{}, err
	}
	now := s.now().UTC()
	hash := contentHash(content)
	record := PlanRecord{
		ID:           defaultID("wplan"),
		WorkflowID:   plan.WorkflowID,
		RepositoryID: strings.TrimSpace(input.RepositoryID),
		Path:         strings.TrimSpace(input.Path),
		Ref:          strings.TrimSpace(input.Ref),
		Name:         plan.Name,
		ContentHash:  hash,
		Plan:         plan,
		CreatedAt:    now,
	}
	record.Plan.PlanID = record.ID
	record.Plan.RepositoryID = record.RepositoryID
	record.Plan.SourcePath = record.Path
	record.Plan.Ref = record.Ref
	record.Plan.ContentHash = record.ContentHash
	record.Plan.CreatedAt = now
	if err := s.store.SavePlan(ctx, record); err != nil {
		return PlanRecord{}, err
	}
	return record, nil
}

func (s *Service) GetPlan(ctx context.Context, id string) (PlanRecord, error) {
	return s.store.GetPlan(ctx, strings.TrimSpace(id))
}

func (s *Service) GetLatestPlan(ctx context.Context, workflowID string) (PlanRecord, error) {
	return s.store.GetLatestPlan(ctx, strings.TrimSpace(workflowID))
}

func (s *Service) ListPlans(ctx context.Context, filter PlanListFilter) ([]PlanRecord, error) {
	filter.RepositoryID = strings.TrimSpace(filter.RepositoryID)
	filter.WorkflowID = strings.TrimSpace(filter.WorkflowID)
	return s.store.ListPlans(ctx, filter)
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func defaultID(prefix string) string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(raw[:])
}
