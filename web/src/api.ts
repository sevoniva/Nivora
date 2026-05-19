import type {
  DashboardSummary,
  DiffView,
  EnvironmentTopology,
  GraphResponse,
  HealthView,
  ReleaseOverview,
  ResourceNode,
  RunnerSummary,
  SecuritySummary,
  TargetExecution,
  TimelineItem
} from "./types";

const API_BASE = import.meta.env.VITE_NIVORA_API_BASE_URL ?? "/api/v1";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: { Accept: "application/json" }
  });
  if (!response.ok) {
    let detail = response.statusText;
    try {
      const body = (await response.json()) as { message?: string; code?: string };
      detail = body.message ?? body.code ?? detail;
    } catch {
      // Keep the HTTP status text when the body is not JSON.
    }
    throw new Error(`${response.status} ${detail}`);
  }
  return (await response.json()) as T;
}

export const api = {
  runnerSummary: () => request<RunnerSummary>("/visualization/runners/summary"),
  securitySummary: () => request<SecuritySummary>("/visualization/security/summary"),
  auditTimeline: () => request<TimelineItem[]>("/visualization/audit/timeline"),
  environmentTopology: (id: string) => request<EnvironmentTopology>(`/visualization/environments/${encodeURIComponent(id)}/topology`),
  pipelineSummary: (id: string) => request<DashboardSummary>(`/visualization/pipeline-runs/${encodeURIComponent(id)}/summary`),
  pipelineDAG: (id: string) => request<GraphResponse>(`/visualization/pipeline-runs/${encodeURIComponent(id)}/dag`),
  pipelineTimeline: (id: string) => request<TimelineItem[]>(`/visualization/pipeline-runs/${encodeURIComponent(id)}/timeline`),
  deploymentTimeline: (id: string) => request<TimelineItem[]>(`/visualization/deployments/${encodeURIComponent(id)}/timeline`),
  deploymentResources: (id: string) => request<ResourceNode[]>(`/visualization/deployments/${encodeURIComponent(id)}/resources`),
  deploymentDiff: (id: string) => request<DiffView>(`/visualization/deployments/${encodeURIComponent(id)}/diff`),
  deploymentHealth: (id: string) => request<HealthView>(`/visualization/deployments/${encodeURIComponent(id)}/health`),
  releaseOverview: (id: string) => request<ReleaseOverview>(`/visualization/releases/${encodeURIComponent(id)}/overview`),
  releaseTimeline: (id: string) => request<TimelineItem[]>(`/visualization/releases/executions/${encodeURIComponent(id)}/timeline`),
  releaseTargets: (id: string) => request<TargetExecution[]>(`/visualization/releases/executions/${encodeURIComponent(id)}/targets`)
};
