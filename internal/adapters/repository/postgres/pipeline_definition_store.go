package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	if !update {
		_, err := s.pool.Exec(ctx, `INSERT INTO pipeline_definitions (id, project_id, name, description, labels, metadata, enabled, version_id, version, definition_hash, definition, created_at, updated_at, version_created_at, version_updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			record.Pipeline.ID, record.Pipeline.ProjectID, record.Pipeline.Name, record.Pipeline.Description, mapJSON(record.Pipeline.Labels), mapJSON(record.Pipeline.Metadata), record.Pipeline.Enabled,
			record.Version.ID, record.Version.Version, record.Version.DefinitionHash, definitionJSON, record.Pipeline.CreatedAt, record.Pipeline.UpdatedAt, record.Version.CreatedAt, record.Version.UpdatedAt)
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE pipeline_definitions SET project_id=$2, name=$3, description=$4, labels=$5, metadata=$6, enabled=$7, version_id=$8, version=$9, definition_hash=$10, definition=$11, updated_at=$12, version_created_at=$13, version_updated_at=$14 WHERE id=$1`,
		record.Pipeline.ID, record.Pipeline.ProjectID, record.Pipeline.Name, record.Pipeline.Description, mapJSON(record.Pipeline.Labels), mapJSON(record.Pipeline.Metadata), record.Pipeline.Enabled,
		record.Version.ID, record.Version.Version, record.Version.DefinitionHash, definitionJSON, record.Pipeline.UpdatedAt, record.Version.CreatedAt, record.Version.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: pipeline %q", pipelineusecase.ErrPipelineDefinitionNotFound, record.Pipeline.ID)
	}
	return nil
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
