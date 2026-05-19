export type Tone = "neutral" | "success" | "danger" | "progress" | "warning";

export interface StatusBadgeModel {
  value?: string;
  tone?: Tone;
}

export interface DashboardSummary {
  id?: string;
  title: string;
  status?: StatusBadgeModel;
  counts?: Record<string, number>;
  metadata?: Record<string, string>;
  updatedAt?: string;
}

export interface SecuritySummary extends DashboardSummary {
  findings?: Record<string, number>;
}

export interface RunnerSummary extends DashboardSummary {
  runners?: ResourceNode[];
}

export interface TimelineItem {
  id: string;
  type: string;
  time: string;
  subject?: string;
  status?: StatusBadgeModel;
  message?: string;
  data?: Record<string, unknown>;
}

export interface GraphNode {
  id: string;
  type: string;
  label: string;
  status?: StatusBadgeModel;
  metadata?: Record<string, unknown>;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  label?: string;
}

export interface GraphResponse {
  nodes?: GraphNode[];
  edges?: GraphEdge[];
}

export interface ResourceNode {
  id: string;
  type: string;
  name: string;
  namespace?: string;
  status?: StatusBadgeModel;
  health?: StatusBadgeModel;
  metadata?: Record<string, unknown>;
}

export interface EnvironmentTopology {
  environmentId: string;
  applications?: ResourceNode[];
  targets?: ResourceNode[];
  latestDeployments?: ResourceNode[];
  resources?: ResourceNode[];
  healthSummary?: DashboardSummary;
}

export interface ReleaseOverview {
  release?: Record<string, unknown>;
  plan?: Record<string, unknown>;
  summary?: DashboardSummary;
  executions?: Record<string, unknown>[];
}

export interface TargetExecution {
  targetId?: string;
  targetName?: string;
  targetType?: string;
  deploymentRunId?: string;
  status?: string;
  order?: number;
  dependencies?: string[];
  warnings?: string[];
}

export interface DiffView {
  summary?: string;
  warnings?: string[];
  addedResources?: string[];
  removedResources?: string[];
  changedResources?: string[];
  unchangedResources?: string[];
  unknownLiveState?: string[];
}

export interface HealthView {
  status?: StatusBadgeModel;
  summary?: Record<string, unknown>;
}

export interface PipelineRunRecord {
  run?: {
    id?: string;
    pipelineId?: string;
    status?: string;
    createdAt?: string;
    updatedAt?: string;
    startedAt?: string;
    finishedAt?: string;
  };
}

export interface DeploymentRunRecord {
  run?: {
    id?: string;
    releaseId?: string;
    environmentId?: string;
    targetId?: string;
    targetType?: string;
    status?: string;
    reason?: string;
    createdAt?: string;
    updatedAt?: string;
  };
}

export interface ReleaseRecord {
  release?: {
    id?: string;
    name?: string;
    version?: string;
    environmentId?: string;
    status?: string;
    createdAt?: string;
  };
  warnings?: string[];
}

export interface ReleaseExecutionRecord {
  execution?: {
    id?: string;
    releaseId?: string;
    environmentId?: string;
    status?: string;
    reason?: string;
    createdAt?: string;
    updatedAt?: string;
  };
}

export interface RunnerRecord {
  id?: string;
  name?: string;
  status?: string;
  executors?: string[];
  labels?: Record<string, string>;
  lastHeartbeatAt?: string;
}
