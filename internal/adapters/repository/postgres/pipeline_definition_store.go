package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

type PipelineDefinitionStore struct {
	pool *pgxpool.Pool
}

var _ pipelineusecase.DefinitionCatalogStore = (*PipelineDefinitionStore)(nil)

func NewPipelineDefinitionStore(pool *pgxpool.Pool) *PipelineDefinitionStore {
	return &PipelineDefinitionStore{pool: pool}
}

func (s *PipelineDefinitionStore) CreateDefinition(ctx context.Context, record pipelineusecase.DefinitionRecord) (pipelineusecase.DefinitionRecord, error) {
	if err := s.insertOrUpdate(ctx, record, false); err != nil {
		if duplicateKey(err) {
			return pipelineusecase.DefinitionRecord{}, fmt.Errorf("%w: pipeline id %q", pipelineusecase.ErrPipelineDefinitionAlreadyExists, record.Pipeline.ID)
		}
		return pipelineusecase.DefinitionRecord{}, err
	}
	return clonePipelineDefinitionRecord(record), nil
}

func (s *PipelineDefinitionStore) GetDefinition(ctx context.Context, id string) (pipelineusecase.DefinitionRecord, error) {
	record, err := scanPipelineDefinition(s.pool.QueryRow(ctx, `SELECT id, project_id, name, description, labels, metadata, enabled, version_id, version, definition_hash, definition, created_at, updated_at, version_created_at, version_updated_at FROM pipeline_definitions WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return pipelineusecase.DefinitionRecord{}, fmt.Errorf("%w: pipeline %q", pipelineusecase.ErrPipelineDefinitionNotFound, id)
	}
	return record, err
}

func (s *PipelineDefinitionStore) ListDefinitions(ctx context.Context, projectID string) ([]pipelineusecase.DefinitionRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, name, description, labels, metadata, enabled, version_id, version, definition_hash, definition, created_at, updated_at, version_created_at, version_updated_at
		FROM pipeline_definitions WHERE ($1='' OR project_id=$1) ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pipelineusecase.DefinitionRecord
	for rows.Next() {
		record, err := scanPipelineDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return nonNil(out), rows.Err()
}

func (s *PipelineDefinitionStore) ListDefinitionVersions(ctx context.Context, id string) ([]domainpipeline.PipelineVersion, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, pipeline_id, version, definition_hash, created_at, updated_at
		FROM pipeline_definition_versions WHERE pipeline_id=$1 ORDER BY version`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []domainpipeline.PipelineVersion
	for rows.Next() {
		var version domainpipeline.PipelineVersion
		if err := rows.Scan(&version.ID, &version.PipelineID, &version.Version, &version.DefinitionHash, &version.CreatedAt, &version.UpdatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(versions) > 0 {
		return versions, nil
	}
	record, err := s.GetDefinition(ctx, id)
	if err != nil {
		return nil, err
	}
	return []domainpipeline.PipelineVersion{record.Version}, nil
}

func (s *PipelineDefinitionStore) GetDefinitionVersion(ctx context.Context, id string, version int) (pipelineusecase.DefinitionVersionRecord, error) {
	var record pipelineusecase.DefinitionVersionRecord
	var definitionJSON []byte
	err := s.pool.QueryRow(ctx, `SELECT id, pipeline_id, version, definition_hash, definition, created_at, updated_at
		FROM pipeline_definition_versions WHERE pipeline_id=$1 AND version=$2`, id, version).Scan(
		&record.Version.ID,
		&record.Version.PipelineID,
		&record.Version.Version,
		&record.Version.DefinitionHash,
		&definitionJSON,
		&record.Version.CreatedAt,
		&record.Version.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		current, currentErr := s.GetDefinition(ctx, id)
		if currentErr == nil && current.Version.Version == version {
			return pipelineusecase.DefinitionVersionRecord{Version: current.Version, Definition: current.Definition}, nil
		}
		return pipelineusecase.DefinitionVersionRecord{}, fmt.Errorf("%w: pipeline %q version %d", pipelineusecase.ErrPipelineDefinitionNotFound, id, version)
	}
	if err != nil {
		return pipelineusecase.DefinitionVersionRecord{}, err
	}
	if err := json.Unmarshal(definitionJSON, &record.Definition); err != nil {
		return pipelineusecase.DefinitionVersionRecord{}, err
	}
	return record, nil
}

func (s *PipelineDefinitionStore) UpdateDefinition(ctx context.Context, record pipelineusecase.DefinitionRecord) (pipelineusecase.DefinitionRecord, error) {
	if err := s.insertOrUpdate(ctx, record, true); err != nil {
		if duplicateKey(err) {
			return pipelineusecase.DefinitionRecord{}, fmt.Errorf("%w: pipeline %q", pipelineusecase.ErrPipelineDefinitionAlreadyExists, record.Pipeline.ID)
		}
		return pipelineusecase.DefinitionRecord{}, err
	}
	return clonePipelineDefinitionRecord(record), nil
}

func (s *PipelineDefinitionStore) insertOrUpdate(ctx context.Context, record pipelineusecase.DefinitionRecord, update bool) error {
	definitionJSON, _ := json.Marshal(record.Definition)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if !update {
		_, err := tx.Exec(ctx, `INSERT INTO pipeline_definitions (id, project_id, name, description, labels, metadata, enabled, version_id, version, definition_hash, definition, created_at, updated_at, version_created_at, version_updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			record.Pipeline.ID, record.Pipeline.ProjectID, record.Pipeline.Name, record.Pipeline.Description, mapJSON(record.Pipeline.Labels), mapJSON(record.Pipeline.Metadata), record.Pipeline.Enabled,
			record.Version.ID, record.Version.Version, record.Version.DefinitionHash, definitionJSON, record.Pipeline.CreatedAt, record.Pipeline.UpdatedAt, record.Version.CreatedAt, record.Version.UpdatedAt)
		if err != nil {
			return err
		}
		if err := insertPipelineDefinitionVersion(ctx, tx, record, definitionJSON); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	tag, err := tx.Exec(ctx, `UPDATE pipeline_definitions SET project_id=$2, name=$3, description=$4, labels=$5, metadata=$6, enabled=$7, version_id=$8, version=$9, definition_hash=$10, definition=$11, updated_at=$12, version_created_at=$13, version_updated_at=$14 WHERE id=$1`,
		record.Pipeline.ID, record.Pipeline.ProjectID, record.Pipeline.Name, record.Pipeline.Description, mapJSON(record.Pipeline.Labels), mapJSON(record.Pipeline.Metadata), record.Pipeline.Enabled,
		record.Version.ID, record.Version.Version, record.Version.DefinitionHash, definitionJSON, record.Pipeline.UpdatedAt, record.Version.CreatedAt, record.Version.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: pipeline %q", pipelineusecase.ErrPipelineDefinitionNotFound, record.Pipeline.ID)
	}
	if err := insertPipelineDefinitionVersion(ctx, tx, record, definitionJSON); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type pipelineDefinitionVersionWriter interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertPipelineDefinitionVersion(ctx context.Context, tx pipelineDefinitionVersionWriter, record pipelineusecase.DefinitionRecord, definitionJSON []byte) error {
	_, err := tx.Exec(ctx, `INSERT INTO pipeline_definition_versions (id, pipeline_id, version, definition_hash, definition, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (pipeline_id, version) DO UPDATE SET
			id=EXCLUDED.id,
			definition_hash=EXCLUDED.definition_hash,
			definition=EXCLUDED.definition,
			updated_at=EXCLUDED.updated_at`,
		record.Version.ID, record.Pipeline.ID, record.Version.Version, record.Version.DefinitionHash, definitionJSON, record.Version.CreatedAt, record.Version.UpdatedAt)
	return err
}

func scanPipelineDefinition(row scanner) (pipelineusecase.DefinitionRecord, error) {
	var record pipelineusecase.DefinitionRecord
	var labels, metadata, definitionJSON []byte
	if err := row.Scan(
		&record.Pipeline.ID,
		&record.Pipeline.ProjectID,
		&record.Pipeline.Name,
		&record.Pipeline.Description,
		&labels,
		&metadata,
		&record.Pipeline.Enabled,
		&record.Version.ID,
		&record.Version.Version,
		&record.Version.DefinitionHash,
		&definitionJSON,
		&record.Pipeline.CreatedAt,
		&record.Pipeline.UpdatedAt,
		&record.Version.CreatedAt,
		&record.Version.UpdatedAt,
	); err != nil {
		return pipelineusecase.DefinitionRecord{}, err
	}
	record.Version.PipelineID = record.Pipeline.ID
	record.Pipeline.Labels = readMap(labels)
	record.Pipeline.Metadata = readMap(metadata)
	if err := json.Unmarshal(definitionJSON, &record.Definition); err != nil {
		return pipelineusecase.DefinitionRecord{}, err
	}
	return record, nil
}

func clonePipelineDefinitionRecord(record pipelineusecase.DefinitionRecord) pipelineusecase.DefinitionRecord {
	record.Pipeline.Labels = readMap(mapJSON(record.Pipeline.Labels))
	record.Pipeline.Metadata = readMap(mapJSON(record.Pipeline.Metadata))
	var definition pipelineusecase.Definition
	body, _ := json.Marshal(record.Definition)
	_ = json.Unmarshal(body, &definition)
	record.Definition = definition
	if record.Version.PipelineID == "" {
		record.Version.PipelineID = record.Pipeline.ID
	}
	return record
}
