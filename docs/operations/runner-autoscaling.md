# Runner Autoscaling Policy

Nivora runners can be scaled horizontally based on job queue depth and claim latency. Operators configure autoscaling in their deployment platform (Kubernetes HPA, Docker Compose replicas, VM auto-scaling groups). Nivora exposes the metrics needed to make scaling decisions.

## Scaling Metrics

### Queue Depth

```bash
curl http://localhost:8080/api/v1/system/runtime/recovery
```

Key fields: `queuedRuns`, `expiredJobClaims`. Scale up when `queuedRuns > 0` persistently. Scale down when `queuedRuns == 0` for >5 minutes.

### Claim Latency

```bash
curl http://localhost:8080/metrics | grep job_claim_latency
```

If P50 claim latency exceeds 500ms or P95 exceeds 2s, add more runners to reduce contention.

### Runner Concurrency

Each runner has a `maxConcurrency` setting. Total cluster capacity = `runners × maxConcurrency`.

```yaml
# Per-runner config
runner:
  name: prod-runner
  group: default
  # maxConcurrency is set at registration or in runner config
```

## Kubernetes HPA Example

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: nivora-runner-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nivora-runner
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Pods
    pods:
      metric:
        name: nivora_job_claim_total
      target:
        type: AverageValue
        averageValue: "50"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60
```

## Docker Compose Scaling

```bash
# Scale runners up
docker compose -f deployments/docker-compose/docker-compose.yaml up -d --scale nivora-runner=5

# Scale down
docker compose -f deployments/docker-compose/docker-compose.yaml up -d --scale nivora-runner=2
```

## Operator Responsibilities

- Monitor `/metrics` and `/api/v1/system/runtime/recovery` for scaling signals.
- Set `minReplicas` based on baseline workload; `maxReplicas` based on infrastructure capacity.
- Ensure new runners can reach the PostgreSQL database and Nivora server API.
- Runner registration is automatic on startup; deregister offline runners periodically.
- Rotate runner tokens when scaling events expose credentials to new hosts.

## Current Limitations

- Nivora does not include a built-in autoscaler. Operators must configure their platform's scaling mechanism.
- No predictive scaling based on historical load patterns.
- No scale-to-zero support (runners cannot be fully offline if any PipelineRuns are expected).
- Runner group/project affinity is not enforced at the autoscaling level.
