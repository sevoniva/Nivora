package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domainpolicy "github.com/sevoniva/nivora/internal/domain/policy"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
)

type PolicyStore struct {
	pool *pgxpool.Pool
}

var _ policyusecase.Store = (*PolicyStore)(nil)

func NewPolicyStore(pool *pgxpool.Pool) *PolicyStore {
	return &PolicyStore{pool: pool}
}

func (s *PolicyStore) Create(ctx context.Context, policy domainpolicy.Policy) (domainpolicy.Policy, error) {
	if err := s.insertOrUpdatePolicy(ctx, policy, false); err != nil {
		if duplicateKey(err) {
			return domainpolicy.Policy{}, fmt.Errorf("%w: policy %q", policyusecase.ErrAlreadyExists, policy.ID)
		}
		return domainpolicy.Policy{}, err
	}
	return clonePolicy(policy), nil
}

func (s *PolicyStore) Get(ctx context.Context, id string) (domainpolicy.Policy, error) {
	policy, err := scanPolicy(s.pool.QueryRow(ctx, `SELECT id, project_id, environment_id, name, description, policy_type, mode, critical_deny, high_warn, require_digest, approval_on_critical, labels, metadata, enabled, created_at, updated_at FROM catalog_policies WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainpolicy.Policy{}, fmt.Errorf("%w: policy %q", policyusecase.ErrNotFound, id)
	}
	return policy, err
}

func (s *PolicyStore) List(ctx context.Context, projectID string, environmentID string) ([]domainpolicy.Policy, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, environment_id, name, description, policy_type, mode, critical_deny, high_warn, require_digest, approval_on_critical, labels, metadata, enabled, created_at, updated_at
		FROM catalog_policies WHERE ($1='' OR project_id=$1) AND ($2='' OR environment_id=$2) ORDER BY name`, projectID, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainpolicy.Policy
	for rows.Next() {
		policy, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return nonNil(out), rows.Err()
}

func (s *PolicyStore) Update(ctx context.Context, policy domainpolicy.Policy) (domainpolicy.Policy, error) {
	if err := s.insertOrUpdatePolicy(ctx, policy, true); err != nil {
		if duplicateKey(err) {
			return domainpolicy.Policy{}, fmt.Errorf("%w: policy %q", policyusecase.ErrAlreadyExists, policy.ID)
		}
		return domainpolicy.Policy{}, err
	}
	return clonePolicy(policy), nil
}

func (s *PolicyStore) CreateAttachment(ctx context.Context, attachment domainpolicy.PolicyAttachment) (domainpolicy.PolicyAttachment, error) {
	_, err := s.pool.Exec(ctx, `INSERT INTO catalog_policy_attachments (id, policy_id, scope_type, scope_id, metadata, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		attachment.ID, attachment.PolicyID, attachment.ScopeType, attachment.ScopeID, mapJSON(attachment.Metadata), attachment.Enabled, attachment.CreatedAt, attachment.UpdatedAt)
	if duplicateKey(err) {
		return domainpolicy.PolicyAttachment{}, fmt.Errorf("%w: attachment %q", policyusecase.ErrAlreadyExists, attachment.ID)
	}
	if err != nil {
		return domainpolicy.PolicyAttachment{}, err
	}
	return clonePolicyAttachment(attachment), nil
}

func (s *PolicyStore) ListAttachments(ctx context.Context, input policyusecase.AttachmentListInput) ([]domainpolicy.PolicyAttachment, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, policy_id, scope_type, scope_id, metadata, enabled, created_at, updated_at
		FROM catalog_policy_attachments
		WHERE ($1='' OR policy_id=$1)
		  AND ($2='' OR scope_type=$2)
		  AND ($3='' OR scope_id=$3)
		  AND ($4::boolean IS NULL OR enabled=$4)
		ORDER BY policy_id, scope_type, scope_id`,
		input.PolicyID, input.ScopeType, input.ScopeID, input.Enabled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainpolicy.PolicyAttachment
	for rows.Next() {
		attachment, err := scanPolicyAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, attachment)
	}
	return nonNil(out), rows.Err()
}

func (s *PolicyStore) insertOrUpdatePolicy(ctx context.Context, policy domainpolicy.Policy, update bool) error {
	if !update {
		_, err := s.pool.Exec(ctx, `INSERT INTO catalog_policies (id, project_id, environment_id, name, description, policy_type, mode, critical_deny, high_warn, require_digest, approval_on_critical, labels, metadata, enabled, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			policy.ID, policy.ProjectID, policy.EnvironmentID, policy.Name, policy.Description, policy.Type, policy.Mode, policy.CriticalDeny, policy.HighWarn,
			policy.RequireDigest, policy.ApprovalOnCritical, mapJSON(policy.Labels), mapJSON(policy.Metadata), policy.Enabled, policy.CreatedAt, policy.UpdatedAt)
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE catalog_policies SET project_id=$2, environment_id=$3, name=$4, description=$5, policy_type=$6, mode=$7, critical_deny=$8, high_warn=$9, require_digest=$10, approval_on_critical=$11, labels=$12, metadata=$13, enabled=$14, updated_at=$15 WHERE id=$1`,
		policy.ID, policy.ProjectID, policy.EnvironmentID, policy.Name, policy.Description, policy.Type, policy.Mode, policy.CriticalDeny, policy.HighWarn,
		policy.RequireDigest, policy.ApprovalOnCritical, mapJSON(policy.Labels), mapJSON(policy.Metadata), policy.Enabled, policy.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: policy %q", policyusecase.ErrNotFound, policy.ID)
	}
	return nil
}

func scanPolicy(row scanner) (domainpolicy.Policy, error) {
	var policy domainpolicy.Policy
	var labels, metadata []byte
	if err := row.Scan(
		&policy.ID,
		&policy.ProjectID,
		&policy.EnvironmentID,
		&policy.Name,
		&policy.Description,
		&policy.Type,
		&policy.Mode,
		&policy.CriticalDeny,
		&policy.HighWarn,
		&policy.RequireDigest,
		&policy.ApprovalOnCritical,
		&labels,
		&metadata,
		&policy.Enabled,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	); err != nil {
		return domainpolicy.Policy{}, err
	}
	policy.Labels = readMap(labels)
	policy.Metadata = readMap(metadata)
	return policy, nil
}

func scanPolicyAttachment(row scanner) (domainpolicy.PolicyAttachment, error) {
	var attachment domainpolicy.PolicyAttachment
	var metadata []byte
	if err := row.Scan(
		&attachment.ID,
		&attachment.PolicyID,
		&attachment.ScopeType,
		&attachment.ScopeID,
		&metadata,
		&attachment.Enabled,
		&attachment.CreatedAt,
		&attachment.UpdatedAt,
	); err != nil {
		return domainpolicy.PolicyAttachment{}, err
	}
	attachment.Metadata = readMap(metadata)
	return attachment, nil
}

func clonePolicy(policy domainpolicy.Policy) domainpolicy.Policy {
	policy.Labels = readMap(mapJSON(policy.Labels))
	policy.Metadata = readMap(mapJSON(policy.Metadata))
	return policy
}

func clonePolicyAttachment(attachment domainpolicy.PolicyAttachment) domainpolicy.PolicyAttachment {
	attachment.Metadata = readMap(mapJSON(attachment.Metadata))
	return attachment
}
