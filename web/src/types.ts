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

export interface ArtifactRecord {
  id?: string;
  type?: string;
  name?: string;
  version?: string;
  reference?: string;
  digest?: string;
  registry?: string;
  repository?: string;
  mediaType?: string;
  sizeBytes?: number;
  manifestSchema?: string;
  metadata?: Record<string, string>;
  createdAt?: string;
}

export interface ArtifactListResponse {
  artifacts?: ArtifactRecord[];
}

export interface RepositoryRecord {
  id?: string;
  name?: string;
  provider?: string;
  url?: string;
  webUrl?: string;
  defaultBranch?: string;
  credentialRef?: string;
  projectId?: string;
  environmentId?: string;
  labels?: Record<string, string>;
  metadata?: Record<string, string>;
  status?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface RepositoryListResponse {
  repositories?: RepositoryRecord[];
}

export interface RepositoryFileRecord {
  path?: string;
  size?: number;
  hash?: string;
}

export interface RepositorySnapshotRecord {
  id?: string;
  repositoryId?: string;
  ref?: string;
  commitSha?: string;
  branch?: string;
  tag?: string;
  treeHash?: string;
  files?: RepositoryFileRecord[];
  detectedLanguages?: string[];
  detectedFrameworks?: string[];
  detectedBuildTools?: string[];
  detectedPackageManagers?: string[];
  detectedDeploymentFiles?: string[];
  detectedWorkflowFiles?: string[];
  detectedSecurityFiles?: string[];
  warnings?: string[];
  metadata?: Record<string, string>;
  createdAt?: string;
}

export interface RepositorySnapshotListResponse {
  snapshots?: RepositorySnapshotRecord[];
}

export interface CommandCandidate {
  name?: string;
  command?: string;
  source?: string;
}

export interface RepositoryIntelligence {
  repositoryId?: string;
  snapshotId?: string;
  languageSummary?: string[];
  frameworkSummary?: string[];
  buildCommandCandidates?: CommandCandidate[];
  testCommandCandidates?: CommandCandidate[];
  packageCommandCandidates?: CommandCandidate[];
  deploymentTargetCandidates?: string[];
  securityScanCandidates?: string[];
  recommendedNivoraWorkflowDraft?: string;
  warnings?: string[];
  createdAt?: string;
}

export interface DevOpsPlan {
  repositoryId?: string;
  snapshotId?: string;
  build?: {
    commands?: CommandCandidate[];
    warnings?: string[];
  };
  test?: {
    commands?: CommandCandidate[];
    warnings?: string[];
  };
  package?: {
    commands?: CommandCandidate[];
    warnings?: string[];
  };
  security?: {
    candidates?: string[];
    warnings?: string[];
  };
  releaseCandidate?: {
    eligible?: boolean;
    artifactCandidates?: string[];
    requiredChecks?: string[];
    warnings?: string[];
  };
  deploymentTargets?: string[];
  releaseReady?: boolean;
  warnings?: string[];
  createdAt?: string;
}

export interface DevOpsPlanResponse {
  plan?: DevOpsPlan;
  mutated?: boolean;
}

export interface DevOpsReadinessReview {
  repositoryId?: string;
  snapshotId?: string;
  status?: string;
  planOnly?: boolean;
  releaseReady?: boolean;
  buildPlanAvailable?: boolean;
  testPlanAvailable?: boolean;
  packagePlanAvailable?: boolean;
  securityPlanAvailable?: boolean;
  deploymentTargets?: string[];
  strengths?: string[];
  blockers?: string[];
  warnings?: string[];
  recommendedNextActions?: string[];
  createdAt?: string;
}

export interface DevOpsReadinessReviewResponse {
  review?: DevOpsReadinessReview;
  mutated?: boolean;
}

export interface WorkflowSummaryRecord {
  workflowId?: string;
  name?: string;
  repositoryId?: string;
  latestPlanId?: string;
  contentHash?: string;
  ref?: string;
  planCount?: number;
  updatedAt?: string;
}

export interface WorkflowListResponse {
  workflows?: WorkflowSummaryRecord[];
}

export interface WorkflowPlanRecord {
  id?: string;
  workflowId?: string;
  repositoryId?: string;
  repositorySnapshotId?: string;
  path?: string;
  ref?: string;
  name?: string;
  contentHash?: string;
  plan?: WorkflowPlan;
  createdAt?: string;
}

export interface WorkflowPlan {
  planId?: string;
  workflowId?: string;
  repositoryId?: string;
  repositorySnapshotId?: string;
  sourcePath?: string;
  ref?: string;
  contentHash?: string;
  name?: string;
  triggers?: string[];
  jobs?: WorkflowPlannedJob[];
  steps?: WorkflowPlannedStep[];
  edges?: WorkflowEdge[];
  matrixExpansions?: WorkflowMatrixExpansion[];
  runnerRequirements?: WorkflowRunnerRequirement[];
  artifactOutputs?: Record<string, unknown>[];
  cacheHints?: Record<string, unknown>[];
  securityWarnings?: string[];
  unsupportedFeatures?: string[];
  estimatedExecutionMode?: string;
  conversionReady?: boolean;
  warnings?: string[];
  createdAt?: string;
}

export interface WorkflowPlannedJob {
  id?: string;
  baseId?: string;
  name?: string;
  needs?: string[];
  runsOn?: string[];
  labels?: Record<string, string>;
  matrix?: Record<string, string>;
  timeoutMinutes?: number;
  stepCount?: number;
  conversionReady?: boolean;
}

export interface WorkflowPlannedStep {
  id?: string;
  jobId?: string;
  name?: string;
  run?: string;
  uses?: string;
  timeoutMinutes?: number;
  continueOnError?: boolean;
  conversionReady?: boolean;
}

export interface WorkflowEdge {
  from?: string;
  to?: string;
}

export interface WorkflowMatrixExpansion {
  jobId?: string;
  values?: Record<string, string>;
}

export interface WorkflowRunnerRequirement {
  jobId?: string;
  runsOn?: string[];
}

export interface WorkflowValidationResponse {
  valid?: boolean;
  plan?: WorkflowPlan;
  error?: string;
}

export interface SecurityFindingRecord {
  id?: string;
  severity?: string;
  category?: string;
  target?: string;
  title?: string;
}

export interface PolicyResultRecord {
  id?: string;
  policyId?: string;
  subjectType?: string;
  subjectId?: string;
  projectId?: string;
  environmentId?: string;
  decision?: string;
  reason?: string;
  findings?: SecurityFindingRecord[];
  evaluatedAt?: string;
}

export interface PolicyResultsResponse {
  results?: PolicyResultRecord[];
}

export interface EvidenceBundleRecord {
  id?: string;
  subjectType?: string;
  subjectId?: string;
  scopeType?: string;
  scopeId?: string;
  summary?: string;
  digest?: string;
  generatedBy?: string;
  generatedAt?: string;
}

export interface EvidenceBundleListResponse {
  bundles?: EvidenceBundleRecord[];
  count?: number;
}

export interface IntegrationCapability {
  name?: string;
  description?: string;
}

export interface IntegrationRecord {
  name?: string;
  type?: string;
  status?: string;
  protocol?: string;
  maturity?: string;
  capabilities?: IntegrationCapability[];
  safeByDefault?: boolean;
  mutatesExternalSystems?: boolean;
  notes?: string[];
  updatedAt?: string;
}

export interface IntegrationListResponse {
  integrations?: IntegrationRecord[];
  count?: number;
  warnings?: string[];
}

export interface PluginCapability {
  name?: string;
  description?: string;
}

export interface PluginManifest {
  name?: string;
  type?: string;
  version?: string;
  protocol?: string;
  status?: string;
  capabilities?: PluginCapability[];
}

export interface SystemRuntimeStatus {
  app?: string;
  environment?: string;
  runtime_mode?: string;
  telemetry?: {
    enabled?: boolean;
    endpoint?: string;
    metrics_endpoint?: string;
    tracing?: string;
  };
  request_id?: string;
  correlation_id?: string;
  trace_id?: string;
}
