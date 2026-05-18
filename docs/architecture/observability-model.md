# Observability Model

Nivora should make delivery behavior explainable through logs, events, metrics, traces, and timelines.

## Logs

Logs explain execution behavior. Runner and Executor logs should be correlated with PipelineRuns, JobRuns, StepRuns, DeploymentRuns, and AuditLogs.

Phase 1.5 stores stdout and stderr as ordered LogChunks for each PipelineRun. Log streaming, external log storage, and retention policies are future work.

## Events

Events should describe lifecycle changes such as PipelineRun created, queued, started, completed, failed, canceled, JobRun assigned, JobRun started, JobRun completed, JobRun failed, runner heartbeat, DeploymentRun started, and policy violation detected.

Phase 1.5 stores PipelineRun events in the in-memory runtime and exposes them through events and timeline APIs.

## Metrics and Traces

Future phases should expose metrics for API requests, workflow latency, runner health, job duration, deployment duration, policy failures, and event delivery. OpenTelemetry is the likely direction but is not wired in Phase 0.

## Correlation IDs

Requests, events, logs, and audit records should share correlation IDs where practical.

## Timelines

Phase 1.5 exposes a minimal PipelineRun timeline from stored events. Future visualization APIs should support richer pipeline timelines, deployment timelines, runner heartbeat history, and audit timelines.
