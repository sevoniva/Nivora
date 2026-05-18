# Observability Model

Nivora should make delivery behavior explainable through logs, events, metrics, traces, and timelines.

## Logs

Logs explain execution behavior. Runner and Executor logs should be correlated with PipelineRuns, JobRuns, StepRuns, DeploymentRuns, and AuditLogs.

## Events

Events should describe lifecycle changes such as PipelineRun created, DeploymentRun started, runner heartbeat, and policy violation detected. Phase 0 includes an AsyncAPI skeleton and in-memory EventBus.

## Metrics and Traces

Future phases should expose metrics for API requests, workflow latency, runner health, job duration, deployment duration, policy failures, and event delivery. OpenTelemetry is the likely direction but is not wired in Phase 0.

## Correlation IDs

Requests, events, logs, and audit records should share correlation IDs where practical.

## Timelines

Future visualization APIs should support pipeline timelines, deployment timelines, runner heartbeat history, and audit timelines.

