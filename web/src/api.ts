import type {
  ArtifactListResponse,
  DashboardSummary,
  DeploymentRunRecord,
  DiffView,
  EvidenceBundleListResponse,
  EnvironmentTopology,
  GraphResponse,
  HealthView,
  IntegrationListResponse,
  PipelineRunRecord,
  PluginManifest,
  PolicyResultsResponse,
  ReleaseExecutionRecord,
  ReleaseRecord,
  ReleaseOverview,
  ResourceNode,
  RunnerRecord,
  RunnerSummary,
  SecuritySummary,
  SystemRuntimeStatus,
  TargetExecution,
  TimelineItem
} from "./types";

export const API_BASE = import.meta.env.VITE_NIVORA_API_BASE_URL ?? "/api/v1";

export interface VersionInfo {
  version?: string;
  commit?: string;
  builtAt?: string;
}

async function request<T>(path: string): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      headers: { Accept: "application/json" }
    });
  } catch (error) {
    const detail = error instanceof Error && error.message ? error.message : "network request failed";
    throw new Error(
      `Cannot reach the Nivora API at ${API_BASE}. Start the backend with make run-server, or set NIVORA_WEB_PROXY_TARGET before npm run dev. Browser error: ${detail}`
    );
  }
  if (!response.ok) {
    let detail = response.statusText;
    try {
      const body = (await response.json()) as { message?: string; code?: string };
      detail = body.message ?? body.code ?? detail;
    } catch {
      // Keep the HTTP status text when the body is not JSON.
    }
    if (response.status >= 500) {
      detail = `${detail}. If this comes from the Vite dev proxy, confirm the backend is running and NIVORA_WEB_PROXY_TARGET points at it.`;
    }
    if (response.status === 401 || response.status === 403) {
      detail = `${detail}. The experimental web console has no login flow; use a local backend auth mode that permits read access or call protected APIs directly.`;
    }
    throw new Error(`${response.status} ${detail}`);
  }
  return (await response.json()) as T;
}

export const api = {
  version: () => request<VersionInfo>("/version"),
  pipelineRuns: () => request<PipelineRunRecord[]>("/pipeline-runs"),
  deploymentRuns: () => request<DeploymentRunRecord[]>("/deployments"),
  releases: () => request<ReleaseRecord[]>("/releases"),
  releaseExecutions: (id: string) => request<ReleaseExecutionRecord[]>(`/releases/${encodeURIComponent(id)}/executions`),
  artifacts: () => request<ArtifactListResponse>("/artifacts"),
  policyResults: () => request<PolicyResultsResponse>("/policies/results"),
  evidenceBundles: () => request<EvidenceBundleListResponse>("/evidence/bundles"),
  integrations: () => request<IntegrationListResponse>("/integrations"),
  plugins: () => request<PluginManifest[]>("/plugins"),
  systemRuntime: () => request<SystemRuntimeStatus>("/system/runtime"),
  runners: () => request<RunnerRecord[]>("/runners"),
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
