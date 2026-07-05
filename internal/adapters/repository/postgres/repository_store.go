package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	repositoryusecase "github.com/sevoniva/nivora/internal/usecase/repository"
)

type RepositoryStore struct {
	pool *pgxpool.Pool
}

var _ repositoryusecase.Store = (*RepositoryStore)(nil)

func NewRepositoryStore(pool *pgxpool.Pool) *RepositoryStore {
	return &RepositoryStore{pool: pool}
}

func (s *RepositoryStore) SaveRepository(ctx context.Context, repository repositoryusecase.Repository) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO repository_records (
		id, name, provider, url, web_url, default_branch, credential_ref, project_id, environment_id,
		labels, metadata, status, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	ON CONFLICT (id) DO UPDATE SET
		name=EXCLUDED.name,
		provider=EXCLUDED.provider,
		url=EXCLUDED.url,
		web_url=EXCLUDED.web_url,
		default_branch=EXCLUDED.default_branch,
		credential_ref=EXCLUDED.credential_ref,
		project_id=EXCLUDED.project_id,
		environment_id=EXCLUDED.environment_id,
		labels=EXCLUDED.labels,
		metadata=EXCLUDED.metadata,
		status=EXCLUDED.status,
		updated_at=EXCLUDED.updated_at`,
		repository.ID, repository.Name, string(repository.Provider), repository.URL, repository.WebURL, repository.DefaultBranch, repository.CredentialRef,
		repository.ProjectID, repository.EnvironmentID, mapJSON(repository.Labels), mapJSON(repository.Metadata), repository.Status, repository.CreatedAt, repository.UpdatedAt)
	return err
}

func (s *RepositoryStore) GetRepository(ctx context.Context, id string) (repositoryusecase.Repository, error) {
	repository, err := scanRepositoryRecord(s.pool.QueryRow(ctx, `SELECT id, name, provider, url, web_url, default_branch, credential_ref, project_id, environment_id, labels, metadata, status, created_at, updated_at FROM repository_records WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return repositoryusecase.Repository{}, fmt.Errorf("%w: repository %q", repositoryusecase.ErrNotFound, id)
	}
	return repository, err
}

func (s *RepositoryStore) ListRepositories(ctx context.Context, projectID string) ([]repositoryusecase.Repository, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, provider, url, web_url, default_branch, credential_ref, project_id, environment_id, labels, metadata, status, created_at, updated_at FROM repository_records WHERE ($1='' OR project_id=$1) ORDER BY name, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []repositoryusecase.Repository
	for rows.Next() {
		repository, err := scanRepositoryRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, repository)
	}
	return nonNil(out), rows.Err()
}

func (s *RepositoryStore) SaveSnapshot(ctx context.Context, snapshot repositoryusecase.RepositorySnapshot) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO repository_snapshots (
		id, repository_id, ref, commit_sha, branch, tag, tree_hash, files,
		detected_languages, detected_frameworks, detected_build_tools, detected_package_managers,
		detected_deployment_files, detected_workflow_files, detected_security_files, warnings, metadata, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	ON CONFLICT (id) DO UPDATE SET
		repository_id=EXCLUDED.repository_id,
		ref=EXCLUDED.ref,
		commit_sha=EXCLUDED.commit_sha,
		branch=EXCLUDED.branch,
		tag=EXCLUDED.tag,
		tree_hash=EXCLUDED.tree_hash,
		files=EXCLUDED.files,
		detected_languages=EXCLUDED.detected_languages,
		detected_frameworks=EXCLUDED.detected_frameworks,
		detected_build_tools=EXCLUDED.detected_build_tools,
		detected_package_managers=EXCLUDED.detected_package_managers,
		detected_deployment_files=EXCLUDED.detected_deployment_files,
		detected_workflow_files=EXCLUDED.detected_workflow_files,
		detected_security_files=EXCLUDED.detected_security_files,
		warnings=EXCLUDED.warnings,
		metadata=EXCLUDED.metadata`,
		snapshot.ID, snapshot.RepositoryID, snapshot.Ref, snapshot.CommitSHA, snapshot.Branch, snapshot.Tag, snapshot.TreeHash,
		jsonBytes(snapshot.Files), jsonBytes(snapshot.DetectedLanguages), jsonBytes(snapshot.DetectedFrameworks), jsonBytes(snapshot.DetectedBuildTools),
		jsonBytes(snapshot.DetectedPackageManagers), jsonBytes(snapshot.DetectedDeploymentFiles), jsonBytes(snapshot.DetectedWorkflowFiles),
		jsonBytes(snapshot.DetectedSecurityFiles), jsonBytes(snapshot.Warnings), mapJSON(snapshot.Metadata), snapshot.CreatedAt)
	return err
}

func (s *RepositoryStore) GetSnapshot(ctx context.Context, id string) (repositoryusecase.RepositorySnapshot, error) {
	snapshot, err := scanRepositorySnapshot(s.pool.QueryRow(ctx, repositorySnapshotSelect()+` WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return repositoryusecase.RepositorySnapshot{}, fmt.Errorf("%w: snapshot %q", repositoryusecase.ErrNotFound, id)
	}
	return snapshot, err
}

func (s *RepositoryStore) GetLatestSnapshot(ctx context.Context, repositoryID string) (repositoryusecase.RepositorySnapshot, error) {
	snapshot, err := scanRepositorySnapshot(s.pool.QueryRow(ctx, repositorySnapshotSelect()+` WHERE repository_id=$1 ORDER BY created_at DESC, id DESC LIMIT 1`, repositoryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return repositoryusecase.RepositorySnapshot{}, fmt.Errorf("%w: latest snapshot for repository %q", repositoryusecase.ErrNotFound, repositoryID)
	}
	return snapshot, err
}

func (s *RepositoryStore) ListSnapshots(ctx context.Context, repositoryID string) ([]repositoryusecase.RepositorySnapshot, error) {
	rows, err := s.pool.Query(ctx, repositorySnapshotSelect()+` WHERE ($1='' OR repository_id=$1) ORDER BY created_at, id`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []repositoryusecase.RepositorySnapshot
	for rows.Next() {
		snapshot, err := scanRepositorySnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, snapshot)
	}
	return nonNil(out), rows.Err()
}

func (s *RepositoryStore) SaveIntelligence(ctx context.Context, intelligence repositoryusecase.RepositoryIntelligence) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO repository_intelligence (
		repository_id, snapshot_id, language_summary, framework_summary, build_command_candidates,
		test_command_candidates, package_command_candidates, deployment_target_candidates,
		security_scan_candidates, recommended_nivora_workflow_draft, warnings, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	ON CONFLICT (repository_id, snapshot_id) DO UPDATE SET
		language_summary=EXCLUDED.language_summary,
		framework_summary=EXCLUDED.framework_summary,
		build_command_candidates=EXCLUDED.build_command_candidates,
		test_command_candidates=EXCLUDED.test_command_candidates,
		package_command_candidates=EXCLUDED.package_command_candidates,
		deployment_target_candidates=EXCLUDED.deployment_target_candidates,
		security_scan_candidates=EXCLUDED.security_scan_candidates,
		recommended_nivora_workflow_draft=EXCLUDED.recommended_nivora_workflow_draft,
		warnings=EXCLUDED.warnings,
		created_at=EXCLUDED.created_at`,
		intelligence.RepositoryID, intelligence.SnapshotID, jsonBytes(intelligence.LanguageSummary), jsonBytes(intelligence.FrameworkSummary),
		jsonBytes(intelligence.BuildCommandCandidates), jsonBytes(intelligence.TestCommandCandidates), jsonBytes(intelligence.PackageCommandCandidates),
		jsonBytes(intelligence.DeploymentTargetCandidates), jsonBytes(intelligence.SecurityScanCandidates), intelligence.RecommendedNivoraWorkflowDraft,
		jsonBytes(intelligence.Warnings), intelligence.CreatedAt)
	return err
}

func (s *RepositoryStore) GetIntelligence(ctx context.Context, repositoryID string, snapshotID string) (repositoryusecase.RepositoryIntelligence, error) {
	intelligence, err := scanRepositoryIntelligence(s.pool.QueryRow(ctx, `SELECT repository_id, snapshot_id, language_summary, framework_summary, build_command_candidates, test_command_candidates, package_command_candidates, deployment_target_candidates, security_scan_candidates, recommended_nivora_workflow_draft, warnings, created_at FROM repository_intelligence WHERE repository_id=$1 AND snapshot_id=$2`, repositoryID, snapshotID))
	if errors.Is(err, pgx.ErrNoRows) {
		return repositoryusecase.RepositoryIntelligence{}, fmt.Errorf("%w: intelligence for repository %q snapshot %q", repositoryusecase.ErrNotFound, repositoryID, snapshotID)
	}
	return intelligence, err
}

func (s *RepositoryStore) SaveDevOpsPlan(ctx context.Context, record repositoryusecase.DevOpsPlanRecord) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO repository_devops_plan_records (
		id, repository_id, snapshot_id, project_id, content_hash, plan, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7)
	ON CONFLICT (id) DO UPDATE SET
		repository_id=EXCLUDED.repository_id,
		snapshot_id=EXCLUDED.snapshot_id,
		project_id=EXCLUDED.project_id,
		content_hash=EXCLUDED.content_hash,
		plan=EXCLUDED.plan,
		created_at=EXCLUDED.created_at`,
		record.ID, record.RepositoryID, record.SnapshotID, record.ProjectID, record.ContentHash, jsonBytes(record.Plan), record.CreatedAt)
	return err
}

func (s *RepositoryStore) GetDevOpsPlan(ctx context.Context, id string) (repositoryusecase.DevOpsPlanRecord, error) {
	record, err := scanDevOpsPlanRecord(s.pool.QueryRow(ctx, repositoryDevOpsPlanRecordSelect()+` WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return repositoryusecase.DevOpsPlanRecord{}, fmt.Errorf("%w: DevOps plan %q", repositoryusecase.ErrNotFound, id)
	}
	return record, err
}

func (s *RepositoryStore) GetLatestDevOpsPlan(ctx context.Context, repositoryID string) (repositoryusecase.DevOpsPlanRecord, error) {
	record, err := scanDevOpsPlanRecord(s.pool.QueryRow(ctx, repositoryDevOpsPlanRecordSelect()+` WHERE repository_id=$1 ORDER BY created_at DESC, id DESC LIMIT 1`, repositoryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return repositoryusecase.DevOpsPlanRecord{}, fmt.Errorf("%w: latest DevOps plan for repository %q", repositoryusecase.ErrNotFound, repositoryID)
	}
	return record, err
}

func (s *RepositoryStore) ListDevOpsPlans(ctx context.Context, repositoryID string) ([]repositoryusecase.DevOpsPlanRecord, error) {
	rows, err := s.pool.Query(ctx, repositoryDevOpsPlanRecordSelect()+` WHERE ($1='' OR repository_id=$1) ORDER BY created_at, id`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []repositoryusecase.DevOpsPlanRecord
	for rows.Next() {
		record, err := scanDevOpsPlanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return nonNil(out), rows.Err()
}

func (s *RepositoryStore) AppendEvent(ctx context.Context, subject string, evt event.Event) error {
	if evt.Time.IsZero() {
		evt.Time = time.Now().UTC()
	}
	if evt.Subject == "" {
		evt.Subject = subject
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO governance_event_logs (id, source, event_type, subject, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (id) DO NOTHING`,
		evt.ID, "repository", evt.Type, subject, payload, evt.Time)
	return err
}

func (s *RepositoryStore) EventsBySubject(ctx context.Context, subject string) ([]event.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_event_logs WHERE source='repository' AND subject=$1 ORDER BY created_at, id`, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []event.Event
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var evt event.Event
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil, err
		}
		out = append(out, evt)
	}
	return nonNil(out), rows.Err()
}

func (s *RepositoryStore) AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error {
	if entry.Subject == "" {
		entry.Subject = subject
	}
	return AppendHashChainedAudit(ctx, s.pool, "repository", entry)
}

func (s *RepositoryStore) AuditsBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_audit_logs WHERE source='repository' AND subject=$1 ORDER BY created_at, id`, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []audit.AuditLog
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var entry audit.AuditLog
		if err := json.Unmarshal(payload, &entry); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return nonNil(out), rows.Err()
}

func repositorySnapshotSelect() string {
	return `SELECT id, repository_id, ref, commit_sha, branch, tag, tree_hash, files, detected_languages, detected_frameworks, detected_build_tools, detected_package_managers, detected_deployment_files, detected_workflow_files, detected_security_files, warnings, metadata, created_at FROM repository_snapshots`
}

func repositoryDevOpsPlanRecordSelect() string {
	return `SELECT id, repository_id, snapshot_id, project_id, content_hash, plan, created_at FROM repository_devops_plan_records`
}

func scanRepositoryRecord(row scanner) (repositoryusecase.Repository, error) {
	var repository repositoryusecase.Repository
	var labels, metadata []byte
	var provider string
	err := row.Scan(&repository.ID, &repository.Name, &provider, &repository.URL, &repository.WebURL, &repository.DefaultBranch, &repository.CredentialRef, &repository.ProjectID, &repository.EnvironmentID, &labels, &metadata, &repository.Status, &repository.CreatedAt, &repository.UpdatedAt)
	if err != nil {
		return repositoryusecase.Repository{}, err
	}
	repository.Provider = repositoryusecase.Provider(provider)
	repository.Labels = readMap(labels)
	repository.Metadata = readMap(metadata)
	return repository, nil
}

func scanRepositorySnapshot(row scanner) (repositoryusecase.RepositorySnapshot, error) {
	var snapshot repositoryusecase.RepositorySnapshot
	var files, languages, frameworks, buildTools, packageManagers, deploymentFiles, workflowFiles, securityFiles, warnings, metadata []byte
	err := row.Scan(&snapshot.ID, &snapshot.RepositoryID, &snapshot.Ref, &snapshot.CommitSHA, &snapshot.Branch, &snapshot.Tag, &snapshot.TreeHash, &files, &languages, &frameworks, &buildTools, &packageManagers, &deploymentFiles, &workflowFiles, &securityFiles, &warnings, &metadata, &snapshot.CreatedAt)
	if err != nil {
		return repositoryusecase.RepositorySnapshot{}, err
	}
	readJSON(files, &snapshot.Files)
	readJSON(languages, &snapshot.DetectedLanguages)
	readJSON(frameworks, &snapshot.DetectedFrameworks)
	readJSON(buildTools, &snapshot.DetectedBuildTools)
	readJSON(packageManagers, &snapshot.DetectedPackageManagers)
	readJSON(deploymentFiles, &snapshot.DetectedDeploymentFiles)
	readJSON(workflowFiles, &snapshot.DetectedWorkflowFiles)
	readJSON(securityFiles, &snapshot.DetectedSecurityFiles)
	readJSON(warnings, &snapshot.Warnings)
	snapshot.Metadata = readMap(metadata)
	return snapshot, nil
}

func scanRepositoryIntelligence(row scanner) (repositoryusecase.RepositoryIntelligence, error) {
	var intelligence repositoryusecase.RepositoryIntelligence
	var languages, frameworks, buildCandidates, testCandidates, packageCandidates, deploymentTargets, securityScans, warnings []byte
	err := row.Scan(&intelligence.RepositoryID, &intelligence.SnapshotID, &languages, &frameworks, &buildCandidates, &testCandidates, &packageCandidates, &deploymentTargets, &securityScans, &intelligence.RecommendedNivoraWorkflowDraft, &warnings, &intelligence.CreatedAt)
	if err != nil {
		return repositoryusecase.RepositoryIntelligence{}, err
	}
	readJSON(languages, &intelligence.LanguageSummary)
	readJSON(frameworks, &intelligence.FrameworkSummary)
	readJSON(buildCandidates, &intelligence.BuildCommandCandidates)
	readJSON(testCandidates, &intelligence.TestCommandCandidates)
	readJSON(packageCandidates, &intelligence.PackageCommandCandidates)
	readJSON(deploymentTargets, &intelligence.DeploymentTargetCandidates)
	readJSON(securityScans, &intelligence.SecurityScanCandidates)
	readJSON(warnings, &intelligence.Warnings)
	return intelligence, nil
}

func scanDevOpsPlanRecord(row scanner) (repositoryusecase.DevOpsPlanRecord, error) {
	var record repositoryusecase.DevOpsPlanRecord
	var plan []byte
	err := row.Scan(&record.ID, &record.RepositoryID, &record.SnapshotID, &record.ProjectID, &record.ContentHash, &plan, &record.CreatedAt)
	if err != nil {
		return repositoryusecase.DevOpsPlanRecord{}, err
	}
	readJSON(plan, &record.Plan)
	return record, nil
}

func jsonBytes(value any) []byte {
	body, _ := json.Marshal(value)
	if len(body) == 0 || string(body) == "null" {
		return []byte("[]")
	}
	return body
}

func readJSON(raw []byte, out any) {
	if len(raw) == 0 {
		return
	}
	_ = json.Unmarshal(raw, out)
}
