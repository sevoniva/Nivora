package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
)

type ArtifactRegistryStore struct {
	pool *pgxpool.Pool
}

var _ artifactusecase.RegistryStore = (*ArtifactRegistryStore)(nil)

func NewArtifactRegistryStore(pool *pgxpool.Pool) *ArtifactRegistryStore {
	return &ArtifactRegistryStore{pool: pool}
}

func (s *ArtifactRegistryStore) CreateRegistry(ctx context.Context, registry domainartifact.ArtifactRegistry) (domainartifact.ArtifactRegistry, error) {
	if err := s.insertOrUpdate(ctx, registry, false); err != nil {
		if duplicateKey(err) {
			return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: registry %q", artifactusecase.ErrRegistryAlreadyExists, registry.ID)
		}
		return domainartifact.ArtifactRegistry{}, err
	}
	return cloneArtifactRegistry(registry), nil
}

func (s *ArtifactRegistryStore) GetRegistry(ctx context.Context, id string) (domainartifact.ArtifactRegistry, error) {
	registry, err := scanArtifactRegistry(s.pool.QueryRow(ctx, `SELECT id, project_id, name, registry_type, registry_url, endpoint, insecure, credential_ref, capabilities, labels, metadata, enabled, created_at, updated_at FROM catalog_artifact_registries WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: registry %q", artifactusecase.ErrRegistryNotFound, id)
	}
	return registry, err
}

func (s *ArtifactRegistryStore) ListRegistries(ctx context.Context, projectID string) ([]domainartifact.ArtifactRegistry, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, name, registry_type, registry_url, endpoint, insecure, credential_ref, capabilities, labels, metadata, enabled, created_at, updated_at
		FROM catalog_artifact_registries WHERE ($1='' OR project_id=$1) ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainartifact.ArtifactRegistry
	for rows.Next() {
		registry, err := scanArtifactRegistry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, registry)
	}
	return nonNil(out), rows.Err()
}

func (s *ArtifactRegistryStore) UpdateRegistry(ctx context.Context, registry domainartifact.ArtifactRegistry) (domainartifact.ArtifactRegistry, error) {
	if err := s.insertOrUpdate(ctx, registry, true); err != nil {
		if duplicateKey(err) {
			return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: registry %q", artifactusecase.ErrRegistryAlreadyExists, registry.ID)
		}
		return domainartifact.ArtifactRegistry{}, err
	}
	return cloneArtifactRegistry(registry), nil
}

func (s *ArtifactRegistryStore) insertOrUpdate(ctx context.Context, registry domainartifact.ArtifactRegistry, update bool) error {
	capabilities, _ := json.Marshal(registry.Capabilities)
	if !update {
		_, err := s.pool.Exec(ctx, `INSERT INTO catalog_artifact_registries (id, project_id, name, registry_type, registry_url, endpoint, insecure, credential_ref, capabilities, labels, metadata, enabled, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			registry.ID, registry.ProjectID, registry.Name, registry.Type, registry.URL, registry.Endpoint, registry.Insecure, registry.CredentialRef,
			capabilities, mapJSON(registry.Labels), mapJSON(registry.Metadata), registry.Enabled, registry.CreatedAt, registry.UpdatedAt)
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE catalog_artifact_registries SET project_id=$2, name=$3, registry_type=$4, registry_url=$5, endpoint=$6, insecure=$7, credential_ref=$8, capabilities=$9, labels=$10, metadata=$11, enabled=$12, updated_at=$13 WHERE id=$1`,
		registry.ID, registry.ProjectID, registry.Name, registry.Type, registry.URL, registry.Endpoint, registry.Insecure, registry.CredentialRef,
		capabilities, mapJSON(registry.Labels), mapJSON(registry.Metadata), registry.Enabled, registry.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: registry %q", artifactusecase.ErrRegistryNotFound, registry.ID)
	}
	return nil
}

func scanArtifactRegistry(row scanner) (domainartifact.ArtifactRegistry, error) {
	var registry domainartifact.ArtifactRegistry
	var capabilities, labels, metadata []byte
	if err := row.Scan(
		&registry.ID,
		&registry.ProjectID,
		&registry.Name,
		&registry.Type,
		&registry.URL,
		&registry.Endpoint,
		&registry.Insecure,
		&registry.CredentialRef,
		&capabilities,
		&labels,
		&metadata,
		&registry.Enabled,
		&registry.CreatedAt,
		&registry.UpdatedAt,
	); err != nil {
		return domainartifact.ArtifactRegistry{}, err
	}
	_ = json.Unmarshal(capabilities, &registry.Capabilities)
	registry.Labels = readMap(labels)
	registry.Metadata = readMap(metadata)
	return registry, nil
}

func cloneArtifactRegistry(registry domainartifact.ArtifactRegistry) domainartifact.ArtifactRegistry {
	registry.Capabilities = append([]string(nil), registry.Capabilities...)
	registry.Labels = readMap(mapJSON(registry.Labels))
	registry.Metadata = readMap(mapJSON(registry.Metadata))
	return registry
}
