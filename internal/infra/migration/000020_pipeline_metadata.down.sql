DROP INDEX IF EXISTS idx_runtime_pipeline_summaries_run_created_at;
DROP INDEX IF EXISTS idx_runtime_pipeline_annotations_level;
DROP INDEX IF EXISTS idx_runtime_pipeline_annotations_run_created_at;
DROP INDEX IF EXISTS idx_runtime_pipeline_cache_run_created_at;
DROP INDEX IF EXISTS idx_runtime_pipeline_artifacts_job;
DROP INDEX IF EXISTS idx_runtime_pipeline_artifacts_run_created_at;

DROP TABLE IF EXISTS runtime_pipeline_step_summaries;
DROP TABLE IF EXISTS runtime_pipeline_annotations;
DROP TABLE IF EXISTS runtime_pipeline_cache_entries;
DROP TABLE IF EXISTS runtime_pipeline_artifacts;
