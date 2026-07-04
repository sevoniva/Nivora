package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	domainenv "github.com/sevoniva/nivora/internal/domain/environment"
	domainorg "github.com/sevoniva/nivora/internal/domain/org"
	domainproject "github.com/sevoniva/nivora/internal/domain/project"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
)

type CatalogStore struct {
	pool *pgxpool.Pool
}

var _ catalogusecase.Store = (*CatalogStore)(nil)

func NewCatalogStore(pool *pgxpool.Pool) *CatalogStore {
	return &CatalogStore{pool: pool}
}

func (s *CatalogStore) CreateOrg(ctx context.Context, org domainorg.Org) (domainorg.Org, error) {
	labels, metadata := mapJSON(org.Labels), mapJSON(org.Metadata)
	_, err := s.pool.Exec(ctx, `INSERT INTO catalog_orgs (id, name, slug, description, labels, metadata, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		org.ID, org.Name, org.Slug, org.Description, labels, metadata, org.Enabled, org.CreatedAt, org.UpdatedAt)
	if duplicateKey(err) {
		return domainorg.Org{}, fmt.Errorf("%w: org id %q", catalogusecase.ErrAlreadyExists, org.ID)
	}
	return org, err
}

func (s *CatalogStore) GetOrg(ctx context.Context, id string) (domainorg.Org, error) {
	org, err := scanOrg(s.pool.QueryRow(ctx, `SELECT id, name, slug, description, labels, metadata, enabled, created_at, updated_at FROM catalog_orgs WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainorg.Org{}, fmt.Errorf("%w: org %q", catalogusecase.ErrNotFound, id)
	}
	return org, err
}

func (s *CatalogStore) ListOrgs(ctx context.Context) ([]domainorg.Org, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, slug, description, labels, metadata, enabled, created_at, updated_at FROM catalog_orgs ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainorg.Org
	for rows.Next() {
		org, err := scanOrg(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, org)
	}
	return nonNil(out), rows.Err()
}

func (s *CatalogStore) UpdateOrg(ctx context.Context, org domainorg.Org) (domainorg.Org, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE catalog_orgs SET name=$2, slug=$3, description=$4, labels=$5, metadata=$6, enabled=$7, updated_at=$8 WHERE id=$1`,
		org.ID, org.Name, org.Slug, org.Description, mapJSON(org.Labels), mapJSON(org.Metadata), org.Enabled, org.UpdatedAt)
	if err != nil {
		if duplicateKey(err) {
			return domainorg.Org{}, fmt.Errorf("%w: org %q", catalogusecase.ErrAlreadyExists, org.ID)
		}
		return domainorg.Org{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainorg.Org{}, fmt.Errorf("%w: org %q", catalogusecase.ErrNotFound, org.ID)
	}
	return org, nil
}

func (s *CatalogStore) CreateProject(ctx context.Context, project domainproject.Project) (domainproject.Project, error) {
	_, err := s.pool.Exec(ctx, `INSERT INTO catalog_projects (id, org_id, name, slug, description, labels, metadata, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		project.ID, project.OrgID, project.Name, project.Slug, project.Description, mapJSON(project.Labels), mapJSON(project.Metadata), project.Enabled, project.CreatedAt, project.UpdatedAt)
	if duplicateKey(err) {
		return domainproject.Project{}, fmt.Errorf("%w: project id %q", catalogusecase.ErrAlreadyExists, project.ID)
	}
	return project, err
}

func (s *CatalogStore) GetProject(ctx context.Context, id string) (domainproject.Project, error) {
	project, err := scanProject(s.pool.QueryRow(ctx, `SELECT id, org_id, name, slug, description, labels, metadata, enabled, created_at, updated_at FROM catalog_projects WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainproject.Project{}, fmt.Errorf("%w: project %q", catalogusecase.ErrNotFound, id)
	}
	return project, err
}

func (s *CatalogStore) ListProjects(ctx context.Context, orgID string) ([]domainproject.Project, error) {
	query := `SELECT id, org_id, name, slug, description, labels, metadata, enabled, created_at, updated_at FROM catalog_projects WHERE ($1='' OR org_id=$1) ORDER BY name`
	rows, err := s.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainproject.Project
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, project)
	}
	return nonNil(out), rows.Err()
}

func (s *CatalogStore) UpdateProject(ctx context.Context, project domainproject.Project) (domainproject.Project, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE catalog_projects SET org_id=$2, name=$3, slug=$4, description=$5, labels=$6, metadata=$7, enabled=$8, updated_at=$9 WHERE id=$1`,
		project.ID, project.OrgID, project.Name, project.Slug, project.Description, mapJSON(project.Labels), mapJSON(project.Metadata), project.Enabled, project.UpdatedAt)
	if err != nil {
		if duplicateKey(err) {
			return domainproject.Project{}, fmt.Errorf("%w: project %q", catalogusecase.ErrAlreadyExists, project.ID)
		}
		return domainproject.Project{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainproject.Project{}, fmt.Errorf("%w: project %q", catalogusecase.ErrNotFound, project.ID)
	}
	return project, nil
}

func (s *CatalogStore) CreateApplication(ctx context.Context, app domainapp.Application) (domainapp.Application, error) {
	_, err := s.pool.Exec(ctx, `INSERT INTO catalog_applications (id, project_id, name, slug, description, labels, metadata, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		app.ID, app.ProjectID, app.Name, app.Slug, app.Description, mapJSON(app.Labels), mapJSON(app.Metadata), app.Enabled, app.CreatedAt, app.UpdatedAt)
	if duplicateKey(err) {
		return domainapp.Application{}, fmt.Errorf("%w: application id %q", catalogusecase.ErrAlreadyExists, app.ID)
	}
	return app, err
}

func (s *CatalogStore) GetApplication(ctx context.Context, id string) (domainapp.Application, error) {
	app, err := scanApplication(s.pool.QueryRow(ctx, `SELECT id, project_id, name, slug, description, labels, metadata, enabled, created_at, updated_at FROM catalog_applications WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainapp.Application{}, fmt.Errorf("%w: application %q", catalogusecase.ErrNotFound, id)
	}
	return app, err
}

func (s *CatalogStore) ListApplications(ctx context.Context, projectID string) ([]domainapp.Application, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, name, slug, description, labels, metadata, enabled, created_at, updated_at FROM catalog_applications WHERE ($1='' OR project_id=$1) ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainapp.Application
	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return nonNil(out), rows.Err()
}

func (s *CatalogStore) UpdateApplication(ctx context.Context, app domainapp.Application) (domainapp.Application, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE catalog_applications SET project_id=$2, name=$3, slug=$4, description=$5, labels=$6, metadata=$7, enabled=$8, updated_at=$9 WHERE id=$1`,
		app.ID, app.ProjectID, app.Name, app.Slug, app.Description, mapJSON(app.Labels), mapJSON(app.Metadata), app.Enabled, app.UpdatedAt)
	if err != nil {
		if duplicateKey(err) {
			return domainapp.Application{}, fmt.Errorf("%w: application %q", catalogusecase.ErrAlreadyExists, app.ID)
		}
		return domainapp.Application{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainapp.Application{}, fmt.Errorf("%w: application %q", catalogusecase.ErrNotFound, app.ID)
	}
	return app, nil
}

func (s *CatalogStore) CreateEnvironment(ctx context.Context, environment domainenv.Environment) (domainenv.Environment, error) {
	_, err := s.pool.Exec(ctx, `INSERT INTO catalog_environments (id, project_id, name, slug, description, labels, metadata, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		environment.ID, environment.ProjectID, environment.Name, environment.Slug, environment.Description, mapJSON(environment.Labels), mapJSON(environment.Metadata), environment.Enabled, environment.CreatedAt, environment.UpdatedAt)
	if duplicateKey(err) {
		return domainenv.Environment{}, fmt.Errorf("%w: environment id %q", catalogusecase.ErrAlreadyExists, environment.ID)
	}
	return environment, err
}

func (s *CatalogStore) GetEnvironment(ctx context.Context, id string) (domainenv.Environment, error) {
	environment, err := scanEnvironment(s.pool.QueryRow(ctx, `SELECT id, project_id, name, slug, description, labels, metadata, enabled, created_at, updated_at FROM catalog_environments WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainenv.Environment{}, fmt.Errorf("%w: environment %q", catalogusecase.ErrNotFound, id)
	}
	return environment, err
}

func (s *CatalogStore) ListEnvironments(ctx context.Context, projectID string) ([]domainenv.Environment, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, name, slug, description, labels, metadata, enabled, created_at, updated_at FROM catalog_environments WHERE ($1='' OR project_id=$1) ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainenv.Environment
	for rows.Next() {
		environment, err := scanEnvironment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, environment)
	}
	return nonNil(out), rows.Err()
}

func (s *CatalogStore) UpdateEnvironment(ctx context.Context, environment domainenv.Environment) (domainenv.Environment, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE catalog_environments SET project_id=$2, name=$3, slug=$4, description=$5, labels=$6, metadata=$7, enabled=$8, updated_at=$9 WHERE id=$1`,
		environment.ID, environment.ProjectID, environment.Name, environment.Slug, environment.Description, mapJSON(environment.Labels), mapJSON(environment.Metadata), environment.Enabled, environment.UpdatedAt)
	if err != nil {
		if duplicateKey(err) {
			return domainenv.Environment{}, fmt.Errorf("%w: environment %q", catalogusecase.ErrAlreadyExists, environment.ID)
		}
		return domainenv.Environment{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainenv.Environment{}, fmt.Errorf("%w: environment %q", catalogusecase.ErrNotFound, environment.ID)
	}
	return environment, nil
}

func (s *CatalogStore) CreateRepository(ctx context.Context, repository domainapp.Repository) (domainapp.Repository, error) {
	_, err := s.pool.Exec(ctx, `INSERT INTO catalog_repositories (id, project_id, name, url, provider, default_branch, credential_ref, labels, metadata, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		repository.ID, repository.ProjectID, repository.Name, repository.URL, repository.Provider, repository.DefaultBranch, repository.CredentialRef, mapJSON(repository.Labels), mapJSON(repository.Metadata), repository.Enabled, repository.CreatedAt, repository.UpdatedAt)
	if duplicateKey(err) {
		return domainapp.Repository{}, fmt.Errorf("%w: repository id %q", catalogusecase.ErrAlreadyExists, repository.ID)
	}
	return repository, err
}

func (s *CatalogStore) GetRepository(ctx context.Context, id string) (domainapp.Repository, error) {
	repository, err := scanRepository(s.pool.QueryRow(ctx, `SELECT id, project_id, name, url, provider, default_branch, credential_ref, labels, metadata, enabled, created_at, updated_at FROM catalog_repositories WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainapp.Repository{}, fmt.Errorf("%w: repository %q", catalogusecase.ErrNotFound, id)
	}
	return repository, err
}

func (s *CatalogStore) ListRepositories(ctx context.Context, projectID string) ([]domainapp.Repository, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, name, url, provider, default_branch, credential_ref, labels, metadata, enabled, created_at, updated_at FROM catalog_repositories WHERE ($1='' OR project_id=$1) ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainapp.Repository
	for rows.Next() {
		repository, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, repository)
	}
	return nonNil(out), rows.Err()
}

func (s *CatalogStore) UpdateRepository(ctx context.Context, repository domainapp.Repository) (domainapp.Repository, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE catalog_repositories SET project_id=$2, name=$3, url=$4, provider=$5, default_branch=$6, credential_ref=$7, labels=$8, metadata=$9, enabled=$10, updated_at=$11 WHERE id=$1`,
		repository.ID, repository.ProjectID, repository.Name, repository.URL, repository.Provider, repository.DefaultBranch, repository.CredentialRef, mapJSON(repository.Labels), mapJSON(repository.Metadata), repository.Enabled, repository.UpdatedAt)
	if err != nil {
		return domainapp.Repository{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainapp.Repository{}, fmt.Errorf("%w: repository %q", catalogusecase.ErrNotFound, repository.ID)
	}
	return repository, nil
}

func (s *CatalogStore) CreateReleaseTarget(ctx context.Context, target domainenv.ReleaseTarget) (domainenv.ReleaseTarget, error) {
	_, err := s.pool.Exec(ctx, `INSERT INTO catalog_release_targets (id, project_id, environment_id, name, target_type, context, namespace, config_ref, credential_ref, labels, metadata, allow_apply, allow_sync, allow_remote_host_deploy, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		target.ID, target.ProjectID, target.EnvironmentID, target.Name, target.TargetType, target.Context, target.Namespace, target.ConfigRef, target.CredentialRef, mapJSON(target.Labels), mapJSON(target.Metadata), target.AllowApply, target.AllowSync, target.AllowRemoteHostDeploy, target.Enabled, target.CreatedAt, target.UpdatedAt)
	if duplicateKey(err) {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target id %q", catalogusecase.ErrAlreadyExists, target.ID)
	}
	return target, err
}

func (s *CatalogStore) GetReleaseTarget(ctx context.Context, id string) (domainenv.ReleaseTarget, error) {
	target, err := scanReleaseTarget(s.pool.QueryRow(ctx, `SELECT id, project_id, environment_id, name, target_type, context, namespace, config_ref, credential_ref, labels, metadata, allow_apply, allow_sync, allow_remote_host_deploy, enabled, created_at, updated_at FROM catalog_release_targets WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target %q", catalogusecase.ErrNotFound, id)
	}
	return target, err
}

func (s *CatalogStore) ListReleaseTargets(ctx context.Context, projectID string, environmentID string) ([]domainenv.ReleaseTarget, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, environment_id, name, target_type, context, namespace, config_ref, credential_ref, labels, metadata, allow_apply, allow_sync, allow_remote_host_deploy, enabled, created_at, updated_at
		FROM catalog_release_targets WHERE ($1='' OR project_id=$1) AND ($2='' OR environment_id=$2) ORDER BY name`, projectID, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainenv.ReleaseTarget
	for rows.Next() {
		target, err := scanReleaseTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, target)
	}
	return nonNil(out), rows.Err()
}

func (s *CatalogStore) UpdateReleaseTarget(ctx context.Context, target domainenv.ReleaseTarget) (domainenv.ReleaseTarget, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE catalog_release_targets SET project_id=$2, environment_id=$3, name=$4, target_type=$5, context=$6, namespace=$7, config_ref=$8, credential_ref=$9, labels=$10, metadata=$11, allow_apply=$12, allow_sync=$13, allow_remote_host_deploy=$14, enabled=$15, updated_at=$16 WHERE id=$1`,
		target.ID, target.ProjectID, target.EnvironmentID, target.Name, target.TargetType, target.Context, target.Namespace, target.ConfigRef, target.CredentialRef, mapJSON(target.Labels), mapJSON(target.Metadata), target.AllowApply, target.AllowSync, target.AllowRemoteHostDeploy, target.Enabled, target.UpdatedAt)
	if err != nil {
		return domainenv.ReleaseTarget{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target %q", catalogusecase.ErrNotFound, target.ID)
	}
	return target, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanOrg(row scanner) (domainorg.Org, error) {
	var org domainorg.Org
	var labels, metadata []byte
	if err := row.Scan(&org.ID, &org.Name, &org.Slug, &org.Description, &labels, &metadata, &org.Enabled, &org.CreatedAt, &org.UpdatedAt); err != nil {
		return domainorg.Org{}, err
	}
	org.Labels = readMap(labels)
	org.Metadata = readMap(metadata)
	return org, nil
}

func scanProject(row scanner) (domainproject.Project, error) {
	var project domainproject.Project
	var labels, metadata []byte
	if err := row.Scan(&project.ID, &project.OrgID, &project.Name, &project.Slug, &project.Description, &labels, &metadata, &project.Enabled, &project.CreatedAt, &project.UpdatedAt); err != nil {
		return domainproject.Project{}, err
	}
	project.Labels = readMap(labels)
	project.Metadata = readMap(metadata)
	return project, nil
}

func scanApplication(row scanner) (domainapp.Application, error) {
	var app domainapp.Application
	var labels, metadata []byte
	if err := row.Scan(&app.ID, &app.ProjectID, &app.Name, &app.Slug, &app.Description, &labels, &metadata, &app.Enabled, &app.CreatedAt, &app.UpdatedAt); err != nil {
		return domainapp.Application{}, err
	}
	app.Labels = readMap(labels)
	app.Metadata = readMap(metadata)
	return app, nil
}

func scanEnvironment(row scanner) (domainenv.Environment, error) {
	var environment domainenv.Environment
	var labels, metadata []byte
	if err := row.Scan(&environment.ID, &environment.ProjectID, &environment.Name, &environment.Slug, &environment.Description, &labels, &metadata, &environment.Enabled, &environment.CreatedAt, &environment.UpdatedAt); err != nil {
		return domainenv.Environment{}, err
	}
	environment.Labels = readMap(labels)
	environment.Metadata = readMap(metadata)
	return environment, nil
}

func scanRepository(row scanner) (domainapp.Repository, error) {
	var repository domainapp.Repository
	var labels, metadata []byte
	if err := row.Scan(&repository.ID, &repository.ProjectID, &repository.Name, &repository.URL, &repository.Provider, &repository.DefaultBranch, &repository.CredentialRef, &labels, &metadata, &repository.Enabled, &repository.CreatedAt, &repository.UpdatedAt); err != nil {
		return domainapp.Repository{}, err
	}
	repository.Labels = readMap(labels)
	repository.Metadata = readMap(metadata)
	return repository, nil
}

func scanReleaseTarget(row scanner) (domainenv.ReleaseTarget, error) {
	var target domainenv.ReleaseTarget
	var labels, metadata []byte
	if err := row.Scan(&target.ID, &target.ProjectID, &target.EnvironmentID, &target.Name, &target.TargetType, &target.Context, &target.Namespace, &target.ConfigRef, &target.CredentialRef, &labels, &metadata, &target.AllowApply, &target.AllowSync, &target.AllowRemoteHostDeploy, &target.Enabled, &target.CreatedAt, &target.UpdatedAt); err != nil {
		return domainenv.ReleaseTarget{}, err
	}
	target.Labels = readMap(labels)
	target.Metadata = readMap(metadata)
	return target, nil
}

func mapJSON(values map[string]string) []byte {
	if values == nil {
		values = map[string]string{}
	}
	body, _ := json.Marshal(values)
	return body
}

func readMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	var out map[string]string
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = map[string]string{}
	}
	return out
}

func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

func duplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
