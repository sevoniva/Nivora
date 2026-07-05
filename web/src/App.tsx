import { useEffect, useMemo, useState } from "react";
import type { DependencyList, ReactNode } from "react";
import { API_BASE, api } from "./api";
import {
  DAGPlaceholder,
  EmptyState,
  ErrorState,
  FindingTable,
  LoadingState,
  ResourceTable,
  RunnerRecordTable,
  RunnerTable,
  StatusBadge,
  SummaryStrip,
  TargetTable,
  Timeline
} from "./components";
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
  DevOpsPlan,
  DevOpsReadinessReview,
  RepositoryIntelligence,
  RepositoryListResponse,
  RepositoryRecord,
  RepositorySnapshotListResponse,
  ResourceNode,
  RunnerSummary,
  SecuritySummary,
  SystemRuntimeStatus,
  TargetExecution,
  TimelineItem,
  WorkflowListResponse,
  WorkflowPlan,
  WorkflowPlanRecord,
  WorkflowValidationResponse
} from "./types";

type Page =
  | "dashboard"
  | "pipelines"
  | "pipeline"
  | "deployments"
  | "deployment"
  | "releases"
  | "release-execution"
  | "repositories"
  | "workflows"
  | "artifacts"
  | "policy"
  | "evidence"
  | "runners"
  | "security"
  | "audit"
  | "environment"
  | "mcp";

const nav: Array<{ page: Page; label: string }> = [
  { page: "dashboard", label: "Dashboard" },
  { page: "pipelines", label: "PipelineRuns" },
  { page: "deployments", label: "Deployments" },
  { page: "releases", label: "Releases" },
  { page: "repositories", label: "Repositories" },
  { page: "workflows", label: "Workflows" },
  { page: "artifacts", label: "Artifacts" },
  { page: "policy", label: "Policy Results" },
  { page: "evidence", label: "Evidence" },
  { page: "runners", label: "Runners" },
  { page: "security", label: "Security" },
  { page: "audit", label: "Audit" },
  { page: "environment", label: "Environment" },
  { page: "mcp", label: "MCP Safety" }
];

const defaultWorkflowContent = `apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: web-console-plan
on: [manual]
jobs:
  build:
    runsOn: [linux]
    steps:
      - name: test
        run: go test ./...
`;

function useHashPage(): [Page, (page: Page) => void] {
  const readPage = () => {
    const value = window.location.hash.replace(/^#\/?/, "") as Page;
    return nav.some((item) => item.page === value) || value === "pipeline" || value === "deployment" || value === "release-execution"
      ? value
      : "dashboard";
  };
  const [page, setPageState] = useState<Page>(readPage);
  useEffect(() => {
    const onHashChange = () => setPageState(readPage());
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);
  const setPage = (next: Page) => {
    window.location.hash = next;
    setPageState(next);
  };
  return [page, setPage];
}

export function App() {
  const [page, setPage] = useHashPage();
  const backend = useFetch(api.version);

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">N</span>
          <div>
            <strong>Nivora</strong>
            <small>Delivery Control Plane</small>
          </div>
        </div>
        <nav>
          {nav.map((item) => (
            <button className={page === item.page ? "active" : ""} key={item.page} onClick={() => setPage(item.page)}>
              {item.label}
            </button>
          ))}
        </nav>
        <p className="sidebar-note">Experimental console. Backend APIs remain the source of truth.</p>
      </aside>
      <main>
        {backend.loading ? <BackendLoadingPage /> : null}
        {backend.error ? <BackendConnectionPage error={backend.error} onRetry={backend.reload} /> : null}
        {!backend.loading && !backend.error ? (
          <>
            {page === "dashboard" ? <DashboardPage /> : null}
            {page === "pipelines" ? <PipelineRunsPage /> : null}
            {page === "pipeline" ? <PipelinePage /> : null}
            {page === "deployments" ? <DeploymentsPage /> : null}
            {page === "deployment" ? <DeploymentPage /> : null}
            {page === "releases" ? <ReleasesPage /> : null}
            {page === "release-execution" ? <ReleaseExecutionPage /> : null}
            {page === "repositories" ? <RepositoriesPage /> : null}
            {page === "workflows" ? <WorkflowsPage /> : null}
            {page === "artifacts" ? <ArtifactsPage /> : null}
            {page === "policy" ? <PolicyResultsPage /> : null}
            {page === "evidence" ? <EvidencePage /> : null}
            {page === "runners" ? <RunnersPage /> : null}
            {page === "security" ? <SecurityPage /> : null}
            {page === "audit" ? <AuditPage /> : null}
            {page === "environment" ? <EnvironmentPage /> : null}
            {page === "mcp" ? <MCPSafetyPage /> : null}
          </>
        ) : null}
      </main>
    </div>
  );
}

function BackendLoadingPage() {
  return (
    <PageFrame title="Connecting" eyebrow="API status" description="Checking the configured Nivora backend before loading runtime views.">
      <LoadingState />
    </PageFrame>
  );
}

function BackendConnectionPage({ error, onRetry }: { error: string; onRetry: () => void }) {
  return (
    <PageFrame title="Backend unavailable" eyebrow="API status" description="The web console could not read the configured Nivora API.">
      <section className="panel connection-panel">
        <div>
          <p className="eyebrow">Configured API base</p>
          <code>{API_BASE}</code>
        </div>
        <ErrorState title="Connection check failed" message={error} actionLabel="Retry connection" onAction={onRetry} />
        <div className="command-list" aria-label="Local development commands">
          <code>make run-server</code>
          <code>make run-web</code>
          <code>cd web && NIVORA_WEB_PROXY_TARGET=http://localhost:8080 npm run dev</code>
        </div>
        <p className="muted">If auth is enabled, use a backend configuration that allows this experimental console to read the existing APIs. No token or credential is stored by the web app.</p>
      </section>
    </PageFrame>
  );
}

function DashboardPage() {
  const runners = useFetch(api.runnerSummary);
  const security = useFetch(api.securitySummary);
  const audit = useFetch(api.auditTimeline);
  const [environmentId, setEnvironmentId] = useState("dev");
  const topology = useFetch(() => api.environmentTopology(environmentId), [environmentId]);

  return (
    <PageFrame title="Dashboard" eyebrow="Backend visualization APIs" description="Read-only control plane views for future frontend work.">
      <div className="toolbar">
        <label>
          Environment
          <input value={environmentId} onChange={(event) => setEnvironmentId(event.target.value)} />
        </label>
      </div>
      <div className="summary-grid">
        <AsyncSummary state={runners} fallbackTitle="Runner summary" />
        <AsyncSummary state={security} fallbackTitle="Security summary" />
        <TopologySummary state={topology} />
      </div>
      <section className="panel">
        <h2>Audit timeline</h2>
        <AsyncBlock state={audit} render={(items) => <Timeline items={items?.slice(-8)} />} />
      </section>
    </PageFrame>
  );
}

function PipelineRunsPage() {
  const state = useFetch(api.pipelineRuns);
  return (
    <PageFrame title="PipelineRuns" eyebrow="CI runtime" description="Queued, running, and completed PipelineRun records from the backend runtime API.">
      <AsyncBlock state={state} render={(records) => (
        <section className="panel">
          <h2>Runs</h2>
          <PipelineRunTable records={records} />
        </section>
      )} />
    </PageFrame>
  );
}

function PipelinePage() {
  const [id, setId] = useState("");
  const activeId = id.trim();
  const summary = useFetch(() => activeId ? api.pipelineSummary(activeId) : Promise.resolve(undefined), [activeId]);
  const graph = useFetch(() => activeId ? api.pipelineDAG(activeId) : Promise.resolve(undefined), [activeId]);
  const timeline = useFetch(() => activeId ? api.pipelineTimeline(activeId) : Promise.resolve(undefined), [activeId]);

  return (
    <PageFrame title="PipelineRun detail" eyebrow="DAG and timeline" description="Inspect a PipelineRun returned by the backend visualization API.">
      <LookupBar label="PipelineRun ID" value={id} onChange={setId} placeholder="prun-..." />
      {!activeId ? <EmptyState title="Enter a PipelineRun ID" detail="Run a local pipeline first, then paste the PipelineRun ID here." /> : null}
      {activeId ? (
        <>
          <AsyncBlock state={summary} render={(data) => <SummaryStrip summary={data} />} />
          <section className="panel"><h2>DAG</h2><AsyncBlock state={graph} render={(data) => <DAGPlaceholder graph={data} />} /></section>
          <section className="panel"><h2>Timeline</h2><AsyncBlock state={timeline} render={(data) => <Timeline items={data} />} /></section>
        </>
      ) : null}
    </PageFrame>
  );
}

function DeploymentsPage() {
  const state = useFetch(api.deploymentRuns);
  return (
    <PageFrame title="Deployments" eyebrow="CD runtime" description="DeploymentRun records across YAML, GitOps, host, and future target types.">
      <AsyncBlock state={state} render={(records) => (
        <section className="panel">
          <h2>Runs</h2>
          <DeploymentRunTable records={records} />
        </section>
      )} />
    </PageFrame>
  );
}

function DeploymentPage() {
  const [id, setId] = useState("");
  const activeId = id.trim();
  const timeline = useFetch(() => activeId ? api.deploymentTimeline(activeId) : Promise.resolve(undefined), [activeId]);
  const resources = useFetch(() => activeId ? api.deploymentResources(activeId) : Promise.resolve(undefined), [activeId]);
  const diff = useFetch(() => activeId ? api.deploymentDiff(activeId) : Promise.resolve(undefined), [activeId]);
  const health = useFetch(() => activeId ? api.deploymentHealth(activeId) : Promise.resolve(undefined), [activeId]);

  return (
    <PageFrame title="Deployment detail" eyebrow="Resources, health, diff" description="Inspect DeploymentRun visualization projections.">
      <LookupBar label="DeploymentRun ID" value={id} onChange={setId} placeholder="drun-..." />
      {!activeId ? <EmptyState title="Enter a DeploymentRun ID" detail="Create a dry-run deployment and paste the run ID here." /> : null}
      {activeId ? (
        <>
          <section className="summary-grid">
            <AsyncBlock state={health} render={(data) => <HealthCard health={data} />} />
            <AsyncBlock state={diff} render={(data) => <DiffCard diff={data} />} />
          </section>
          <section className="panel"><h2>Resources</h2><AsyncBlock state={resources} render={(data) => <ResourceTable resources={data} />} /></section>
          <section className="panel"><h2>Timeline</h2><AsyncBlock state={timeline} render={(data) => <Timeline items={data} />} /></section>
        </>
      ) : null}
    </PageFrame>
  );
}

function ReleasesPage() {
  const state = useFetch(api.releases);
  const [releaseId, setReleaseId] = useState("");
  const releaseKey = releaseId.trim();
  const executions = useFetch(() => releaseKey ? api.releaseExecutions(releaseKey) : Promise.resolve(undefined), [releaseKey]);
  return (
    <PageFrame title="Releases" eyebrow="Artifact intent" description="Release records and their target execution history.">
      <AsyncBlock state={state} render={(records) => (
        <section className="panel">
          <h2>Releases</h2>
          <ReleaseTable records={records} onSelect={setReleaseId} />
        </section>
      )} />
      <LookupBar label="Release ID for executions" value={releaseId} onChange={setReleaseId} placeholder="rel-..." />
      {releaseKey ? (
        <section className="panel">
          <h2>Executions</h2>
          <AsyncBlock state={executions} render={(records) => <ReleaseExecutionTable records={records} />} />
        </section>
      ) : null}
    </PageFrame>
  );
}

function ReleaseExecutionPage() {
  const [releaseId, setReleaseId] = useState("");
  const [executionId, setExecutionId] = useState("");
  const releaseKey = releaseId.trim();
  const executionKey = executionId.trim();
  const overview = useFetch(() => releaseKey ? api.releaseOverview(releaseKey) : Promise.resolve(undefined), [releaseKey]);
  const timeline = useFetch(() => executionKey ? api.releaseTimeline(executionKey) : Promise.resolve(undefined), [executionKey]);
  const targets = useFetch(() => executionKey ? api.releaseTargets(executionKey) : Promise.resolve(undefined), [executionKey]);

  return (
    <PageFrame title="Release execution detail" eyebrow="Overview and targets" description="Connect release intent with target-level execution state.">
      <div className="lookup-grid">
        <LookupBar label="Release ID" value={releaseId} onChange={setReleaseId} placeholder="rel-..." />
        <LookupBar label="Execution ID" value={executionId} onChange={setExecutionId} placeholder="rexec-..." />
      </div>
      {releaseKey ? <AsyncBlock state={overview} render={(data) => <ReleaseOverviewCard overview={data} />} /> : null}
      {executionKey ? (
        <>
          <section className="panel"><h2>Targets</h2><AsyncBlock state={targets} render={(data) => <TargetTable targets={data} />} /></section>
          <section className="panel"><h2>Timeline</h2><AsyncBlock state={timeline} render={(data) => <Timeline items={data} />} /></section>
        </>
      ) : null}
      {!releaseKey && !executionKey ? <EmptyState title="Enter release IDs" detail="Use a Release ID for overview and a ReleaseExecution ID for target timeline data." /> : null}
    </PageFrame>
  );
}

function RepositoriesPage() {
  const [projectId, setProjectId] = useState("");
  const [repositoryId, setRepositoryId] = useState("");
  const projectKey = projectId.trim();
  const repositoryKey = repositoryId.trim();
  const repositories = useFetch(() => api.repositories(projectKey || undefined), [projectKey]);
  const detail = useFetch(() => repositoryKey ? api.repository(repositoryKey) : Promise.resolve(undefined), [repositoryKey]);
  const snapshots = useFetch(() => repositoryKey ? api.repositorySnapshots(repositoryKey) : Promise.resolve(undefined), [repositoryKey]);
  const intelligence = useFetch(() => repositoryKey ? api.repositoryIntelligence(repositoryKey) : Promise.resolve(undefined), [repositoryKey]);
  const [planState, setPlanState] = useState<AsyncActionState<DevOpsPlan>>();
  const [reviewState, setReviewState] = useState<AsyncActionState<DevOpsReadinessReview>>();

  async function loadPlan() {
    if (!repositoryKey) return;
    setPlanState({ loading: true });
    try {
      const result = await api.repositoryDevOpsPlan(repositoryKey);
      setPlanState({ data: result.plan });
    } catch (error) {
      setPlanState({ error: error instanceof Error ? error.message : "Unknown error" });
    }
  }

  async function loadReview() {
    if (!repositoryKey) return;
    setReviewState({ loading: true });
    try {
      const result = await api.repositoryReadinessReview(repositoryKey);
      setReviewState({ data: result.review });
    } catch (error) {
      setReviewState({ error: error instanceof Error ? error.message : "Unknown error" });
    }
  }

  return (
    <PageFrame title="Repositories" eyebrow="Repository intelligence" description="Read-only repository catalog, static snapshots, intelligence, and plan-only DevOps review output.">
      <div className="lookup-grid">
        <LookupBar label="Project filter" value={projectId} onChange={setProjectId} placeholder="project-id (optional)" />
        <LookupBar label="Selected repository ID" value={repositoryId} onChange={setRepositoryId} placeholder="repo-..." />
      </div>
      <AsyncBlock state={repositories} render={(payload) => (
        <section className="panel">
          <h2>Repository catalog</h2>
          <RepositoryTable payload={payload} onSelect={(id) => {
            setRepositoryId(id);
            setPlanState(undefined);
            setReviewState(undefined);
          }} />
        </section>
      )} />
      {!repositoryKey ? <EmptyState title="Select a repository" detail="Choose a repository row or paste a Repository ID to inspect snapshots and intelligence." /> : null}
      {repositoryKey ? (
        <>
          <AsyncBlock state={detail} render={(repository) => <RepositoryDetailCard repository={repository} />} />
          <section className="panel">
            <h2>Snapshots</h2>
            <AsyncBlock state={snapshots} render={(payload) => <RepositorySnapshotTable payload={payload} />} />
          </section>
          <section className="panel">
            <h2>Intelligence</h2>
            <AsyncBlock state={intelligence} render={(data) => <RepositoryIntelligencePanel intelligence={data} />} />
          </section>
          <section className="panel">
            <div className="panel-heading">
              <h2>Plan-only DevOps review</h2>
              <div className="action-row">
                <button className="inline-action" type="button" onClick={loadPlan}>Load plan</button>
                <button className="inline-action" type="button" onClick={loadReview}>Load readiness</button>
              </div>
            </div>
            <div className="split-grid">
              <AsyncActionBlock state={planState} emptyTitle="No DevOps plan loaded" emptyDetail="Click Load plan to request metadata-only build/test/package/security recommendations." render={(plan) => <DevOpsPlanPanel plan={plan} />} />
              <AsyncActionBlock state={reviewState} emptyTitle="No readiness review loaded" emptyDetail="Click Load readiness to request a plan-only release readiness review." render={(review) => <ReadinessReviewPanel review={review} />} />
            </div>
          </section>
        </>
      ) : null}
    </PageFrame>
  );
}

function WorkflowsPage() {
  const [repositoryId, setRepositoryId] = useState("");
  const [workflowId, setWorkflowId] = useState("");
  const [content, setContent] = useState(defaultWorkflowContent);
  const [validation, setValidation] = useState<AsyncActionState<WorkflowValidationResponse>>();
  const [planned, setPlanned] = useState<AsyncActionState<WorkflowPlan>>();
  const repositoryKey = repositoryId.trim();
  const workflowKey = workflowId.trim();
  const workflows = useFetch(() => api.workflows(repositoryKey || undefined), [repositoryKey]);
  const plans = useFetch(() => api.workflowPlans(repositoryKey || undefined), [repositoryKey]);
  const latestPlan = useFetch(() => workflowKey ? api.workflowLatestPlan(workflowKey) : Promise.resolve(undefined), [workflowKey]);

  async function validateWorkflow() {
    setValidation({ loading: true });
    try {
      const result = await api.workflowValidate(content, repositoryKey || undefined);
      setValidation({ data: result });
    } catch (error) {
      setValidation({ error: error instanceof Error ? error.message : "Unknown error" });
    }
  }

  async function planWorkflow() {
    setPlanned({ loading: true });
    try {
      const result = await api.workflowPlan(content, repositoryKey || undefined);
      setPlanned({ data: result });
    } catch (error) {
      setPlanned({ error: error instanceof Error ? error.message : "Unknown error" });
    }
  }

  return (
    <PageFrame title="Workflows" eyebrow="Plan-only authoring" description="Inspect stored Nivora Workflow plans and validate or plan workflow YAML without executing jobs.">
      <div className="lookup-grid">
        <LookupBar label="Repository filter" value={repositoryId} onChange={setRepositoryId} placeholder="repo-id (optional)" />
        <LookupBar label="Workflow ID for latest plan" value={workflowId} onChange={setWorkflowId} placeholder="workflow-id" />
      </div>
      <section className="summary-grid">
        <AsyncBlock state={workflows} render={(payload) => <WorkflowSummaryCard payload={payload} />} />
        <AsyncBlock state={plans} render={(payload) => <WorkflowPlanCountCard payload={payload} />} />
        <section className="panel compact">
          <h2>Execution boundary</h2>
          <StatusBadge status="plan-only" />
          <p className="muted">This page calls validate and plan endpoints only. It does not run workflows, claim jobs, deploy, sync, or approve anything.</p>
        </section>
      </section>
      <section className="panel">
        <h2>Workflow catalog</h2>
        <AsyncBlock state={workflows} render={(payload) => <WorkflowTable payload={payload} onSelect={setWorkflowId} />} />
      </section>
      {workflowKey ? (
        <section className="panel">
          <h2>Latest plan</h2>
          <AsyncBlock state={latestPlan} render={(plan) => <WorkflowPlanPanel plan={plan} />} />
        </section>
      ) : null}
      <section className="panel">
        <div className="panel-heading">
          <h2>Validate or plan YAML</h2>
          <div className="action-row">
            <button className="inline-action" type="button" onClick={validateWorkflow}>Validate</button>
            <button className="inline-action" type="button" onClick={planWorkflow}>Plan</button>
          </div>
        </div>
        <textarea value={content} onChange={(event) => setContent(event.target.value)} spellCheck={false} />
        <div className="split-grid">
          <AsyncActionBlock state={validation} emptyTitle="No validation result" emptyDetail="Click Validate to parse the workflow without storing a plan record." render={(result) => <WorkflowValidationPanel result={result} />} />
          <AsyncActionBlock state={planned} emptyTitle="No planned workflow" emptyDetail="Click Plan to store and render a plan-only WorkflowPlan record." render={(plan) => <WorkflowPlanPanel plan={plan} />} />
        </div>
      </section>
    </PageFrame>
  );
}

function RunnersPage() {
  const state = useFetch(api.runnerSummary);
  const records = useFetch(api.runners);
  return (
    <PageFrame title="Runners" eyebrow="Execution plane" description="Runner status and executor capability summary.">
      <AsyncBlock state={state} render={(summary) => (
        <>
          <SummaryStrip summary={summary} />
          <section className="panel"><h2>Runner table</h2><RunnerTable summary={summary} /></section>
        </>
      )} />
      <section className="panel">
        <h2>Runtime runner records</h2>
        <AsyncBlock state={records} render={(runners) => <RunnerRecordTable runners={runners} />} />
      </section>
    </PageFrame>
  );
}

function ArtifactsPage() {
  const state = useFetch(api.artifacts);
  return (
    <PageFrame title="Artifacts" eyebrow="Immutable identity" description="Tracked artifact records returned by the backend artifact catalog.">
      <AsyncBlock state={state} render={(data) => <ArtifactTable payload={data} />} />
    </PageFrame>
  );
}

function PolicyResultsPage() {
  const state = useFetch(api.policyResults);
  return (
    <PageFrame title="Policy results" eyebrow="Governance signal" description="Stored policy decisions across artifacts, manifests, deployment plans, and releases.">
      <AsyncBlock state={state} render={(data) => <PolicyResultTable payload={data} />} />
    </PageFrame>
  );
}

function EvidencePage() {
  const state = useFetch(api.evidenceBundles);
  return (
    <PageFrame title="Evidence bundles" eyebrow="Audit evidence" description="Persisted evidence bundle metadata. Secret values are not rendered by this console.">
      <AsyncBlock state={state} render={(data) => <EvidenceBundleTable payload={data} />} />
    </PageFrame>
  );
}

function MCPSafetyPage() {
  const state = useFetch(() => Promise.all([api.systemRuntime(), api.integrations(), api.plugins()]).then(([runtime, integrations, plugins]) => ({
    runtime,
    integrations,
    plugins
  })));
  return (
    <PageFrame title="MCP safety" eyebrow="Read-only control plane" description="Runtime, integration, and plugin metadata used to reason about the AI/MCP control-plane boundary.">
      <AsyncBlock state={state} render={(data) => (
        <>
          <MCPRuntimeCard runtime={data?.runtime} />
          <IntegrationTable payload={data?.integrations} />
          <PluginTable plugins={data?.plugins} />
        </>
      )} />
    </PageFrame>
  );
}

function SecurityPage() {
  const state = useFetch(api.securitySummary);
  return (
    <PageFrame title="Security summary" eyebrow="Policy gate signal" description="Aggregate scan and finding counts from the backend security foundation.">
      <AsyncBlock state={state} render={(summary) => (
        <>
          <SummaryStrip summary={summary} />
          <section className="panel"><h2>Finding table</h2><FindingTable findings={summary?.findings} /></section>
          <section className="panel finding-grid">
            {Object.entries(summary?.findings ?? {}).map(([severity, count]) => (
              <div className="finding" key={severity}>
                <span>{severity}</span>
                <strong>{count}</strong>
              </div>
            ))}
          </section>
        </>
      )} />
    </PageFrame>
  );
}

function AuditPage() {
  const audit = useFetch(api.auditTimeline);
  return (
    <PageFrame title="Audit timeline" eyebrow="Evidence trail" description="Audit-oriented timeline projection across pipeline, deployment, release, and security records.">
      <section className="panel">
        <h2>Timeline</h2>
        <AsyncBlock state={audit} render={(items) => <Timeline items={items} />} />
      </section>
    </PageFrame>
  );
}

function EnvironmentPage() {
  const [environmentId, setEnvironmentId] = useState("dev");
  const activeId = environmentId.trim();
  const topology = useFetch(() => activeId ? api.environmentTopology(activeId) : Promise.resolve(undefined), [activeId]);
  return (
    <PageFrame title="Environment topology" eyebrow="Delivery map" description="Applications, targets, latest deployments, and resources grouped by environment.">
      <LookupBar label="Environment ID" value={environmentId} onChange={setEnvironmentId} placeholder="dev" />
      <AsyncBlock state={topology} render={(data) => (
        <>
          <TopologySummary state={{ data, loading: false, reload: () => undefined }} />
          <section className="summary-grid">
            <TopologyPanel title="Applications" resources={data?.applications} />
            <TopologyPanel title="Targets" resources={data?.targets} />
            <TopologyPanel title="Latest deployments" resources={data?.latestDeployments} />
          </section>
          <section className="panel"><h2>Resources</h2><ResourceTable resources={data?.resources} /></section>
        </>
      )} />
    </PageFrame>
  );
}

function RepositoryTable({ payload, onSelect }: { payload?: RepositoryListResponse; onSelect: (id: string) => void }) {
  const repositories = payload?.repositories ?? [];
  if (!repositories.length) return <EmptyState title="No repositories" detail="No repository catalog records were returned by the backend." />;
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Repository</th><th>Provider</th><th>Project</th><th>Status</th><th>Default ref</th><th>Inspect</th></tr>
        </thead>
        <tbody>
          {repositories.map((repository) => {
            const id = repository.id || "";
            return (
              <tr key={id || repository.name}>
                <td>
                  <strong>{repository.name || id || "-"}</strong>
                  <small>{repository.url || repository.webUrl || "-"}</small>
                </td>
                <td>{repository.provider || "-"}</td>
                <td>{repository.projectId || "-"}</td>
                <td><StatusBadge status={repository.status} /></td>
                <td>{repository.defaultBranch || "-"}</td>
                <td><button className="table-action" type="button" disabled={!id} onClick={() => onSelect(id)}>Load</button></td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function RepositoryDetailCard({ repository }: { repository?: RepositoryRecord }) {
  if (!repository?.id) return <EmptyState title="Repository not found" detail="The backend did not return repository metadata for this ID." />;
  return (
    <section className="summary-grid">
      <SummaryStrip summary={{
        title: repository.name || repository.id,
        status: { value: repository.status || "unknown" },
        counts: {
          labels: Object.keys(repository.labels ?? {}).length,
          metadata: Object.keys(repository.metadata ?? {}).length
        },
        metadata: {
          provider: repository.provider || "-",
          project: repository.projectId || "-"
        }
      }} />
      <section className="panel compact">
        <h2>Repository identity</h2>
        <p><code>{repository.id}</code></p>
        <p>{repository.url || repository.webUrl || "-"}</p>
        <p className="muted">CredentialRef metadata: {repository.credentialRef || "not set"}. Secret values are not returned by this API.</p>
      </section>
      <section className="panel compact">
        <h2>Freshness</h2>
        <p>Created: {formatDate(repository.createdAt)}</p>
        <p>Updated: {formatDate(repository.updatedAt)}</p>
        <p className="muted">Default branch/ref: {repository.defaultBranch || "-"}</p>
      </section>
    </section>
  );
}

function RepositorySnapshotTable({ payload }: { payload?: RepositorySnapshotListResponse }) {
  const snapshots = payload?.snapshots ?? [];
  if (!snapshots.length) return <EmptyState title="No snapshots" detail="Create a metadata-only repository snapshot before inspecting intelligence." />;
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Snapshot</th><th>Ref</th><th>Signals</th><th>Files</th><th>Created</th></tr>
        </thead>
        <tbody>
          {snapshots.map((snapshot) => (
            <tr key={snapshot.id}>
              <td>
                <code>{snapshot.id || "-"}</code>
                <small>{snapshot.treeHash ? shortDigest(snapshot.treeHash) : "-"}</small>
              </td>
              <td>{snapshot.ref || snapshot.branch || snapshot.tag || "-"}</td>
              <td>
                <TagList values={[...(snapshot.detectedLanguages ?? []), ...(snapshot.detectedFrameworks ?? []), ...(snapshot.detectedBuildTools ?? [])]} />
              </td>
              <td>{snapshot.files?.length ?? 0}</td>
              <td>{formatDate(snapshot.createdAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function RepositoryIntelligencePanel({ intelligence }: { intelligence?: RepositoryIntelligence }) {
  if (!intelligence?.repositoryId) return <EmptyState title="No intelligence" detail="Analyze the latest snapshot before reading repository intelligence." />;
  return (
    <div className="split-grid">
      <section>
        <h3>Detected stack</h3>
        <TagList values={[...(intelligence.languageSummary ?? []), ...(intelligence.frameworkSummary ?? [])]} />
        <h3>Build</h3>
        <CommandList commands={intelligence.buildCommandCandidates} />
        <h3>Test</h3>
        <CommandList commands={intelligence.testCommandCandidates} />
        <h3>Package</h3>
        <CommandList commands={intelligence.packageCommandCandidates} />
      </section>
      <section>
        <h3>Delivery hints</h3>
        <TagList values={intelligence.deploymentTargetCandidates} />
        <h3>Security hints</h3>
        <TagList values={intelligence.securityScanCandidates} />
        <h3>Warnings</h3>
        <WarningList values={intelligence.warnings} />
      </section>
      <section className="wide">
        <h3>Recommended Nivora workflow draft</h3>
        <pre>{intelligence.recommendedNivoraWorkflowDraft || "No draft returned."}</pre>
      </section>
    </div>
  );
}

function DevOpsPlanPanel({ plan }: { plan?: DevOpsPlan }) {
  if (!plan?.repositoryId) return <EmptyState title="No plan" detail="The backend did not return a DevOps plan." />;
  return (
    <section>
      <StatusBadge status={plan.releaseReady ? "release-ready" : "plan-only"} />
      <h3>Build</h3>
      <CommandList commands={plan.build?.commands} />
      <h3>Test</h3>
      <CommandList commands={plan.test?.commands} />
      <h3>Package</h3>
      <CommandList commands={plan.package?.commands} />
      <h3>Deployment targets</h3>
      <TagList values={plan.deploymentTargets} />
      <WarningList values={[...(plan.warnings ?? []), ...(plan.releaseCandidate?.warnings ?? [])]} />
    </section>
  );
}

function ReadinessReviewPanel({ review }: { review?: DevOpsReadinessReview }) {
  if (!review?.repositoryId) return <EmptyState title="No readiness review" detail="The backend did not return a readiness review." />;
  return (
    <section>
      <StatusBadge status={review.status || "unknown"} />
      <div className="mini-metrics">
        <span>build: <strong>{review.buildPlanAvailable ? "yes" : "no"}</strong></span>
        <span>test: <strong>{review.testPlanAvailable ? "yes" : "no"}</strong></span>
        <span>package: <strong>{review.packagePlanAvailable ? "yes" : "no"}</strong></span>
        <span>security: <strong>{review.securityPlanAvailable ? "yes" : "no"}</strong></span>
      </div>
      <h3>Next actions</h3>
      <TagList values={review.recommendedNextActions} />
      <h3>Warnings and blockers</h3>
      <WarningList values={[...(review.blockers ?? []), ...(review.warnings ?? [])]} />
    </section>
  );
}

function WorkflowSummaryCard({ payload }: { payload?: WorkflowListResponse }) {
  const workflows = payload?.workflows ?? [];
  return <SummaryStrip summary={{ title: "Workflow catalog", status: { value: workflows.length ? "available" : "empty" }, counts: { workflows: workflows.length } }} />;
}

function WorkflowPlanCountCard({ payload }: { payload?: { plans?: WorkflowPlanRecord[] } }) {
  const plans = payload?.plans ?? [];
  return <SummaryStrip summary={{ title: "Stored plans", status: { value: plans.length ? "available" : "empty" }, counts: { plans: plans.length } }} />;
}

function WorkflowTable({ payload, onSelect }: { payload?: WorkflowListResponse; onSelect: (id: string) => void }) {
  const workflows = payload?.workflows ?? [];
  if (!workflows.length) return <EmptyState title="No workflows" detail="No stored WorkflowPlan summaries were returned by the backend." />;
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Workflow</th><th>Repository</th><th>Plans</th><th>Latest plan</th><th>Updated</th><th>Inspect</th></tr>
        </thead>
        <tbody>
          {workflows.map((workflow) => {
            const id = workflow.workflowId || "";
            return (
              <tr key={id || workflow.latestPlanId}>
                <td>
                  <strong>{workflow.name || id || "-"}</strong>
                  <small>{workflow.contentHash ? shortDigest(workflow.contentHash) : "-"}</small>
                </td>
                <td>{workflow.repositoryId || "-"}</td>
                <td>{workflow.planCount ?? 0}</td>
                <td><code>{workflow.latestPlanId || "-"}</code></td>
                <td>{formatDate(workflow.updatedAt)}</td>
                <td><button className="table-action" type="button" disabled={!id} onClick={() => onSelect(id)}>Load</button></td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function WorkflowPlanPanel({ plan }: { plan?: WorkflowPlan }) {
  if (!plan?.workflowId) return <EmptyState title="No workflow plan" detail="No plan data is available for this workflow." />;
  return (
    <div className="split-grid">
      <section>
        <StatusBadge status={plan.conversionReady ? "conversion-ready" : "plan-only"} />
        <h3>{plan.name || plan.workflowId}</h3>
        <p className="muted">Mode: {plan.estimatedExecutionMode || "-"} · triggers: {plan.triggers?.join(", ") || "-"}</p>
        <div className="mini-metrics">
          <span>jobs: <strong>{plan.jobs?.length ?? 0}</strong></span>
          <span>steps: <strong>{plan.steps?.length ?? 0}</strong></span>
          <span>edges: <strong>{plan.edges?.length ?? 0}</strong></span>
          <span>matrix: <strong>{plan.matrixExpansions?.length ?? 0}</strong></span>
        </div>
      </section>
      <section>
        <h3>Runner requirements</h3>
        <TagList values={plan.runnerRequirements?.flatMap((item) => item.runsOn ?? [])} />
        <h3>Warnings</h3>
        <WarningList values={[...(plan.warnings ?? []), ...(plan.securityWarnings ?? []), ...(plan.unsupportedFeatures ?? [])]} />
      </section>
      <section className="wide">
        <h3>Jobs</h3>
        <WorkflowJobTable plan={plan} />
      </section>
    </div>
  );
}

function WorkflowValidationPanel({ result }: { result?: WorkflowValidationResponse }) {
  if (!result) return <EmptyState title="No validation result" detail="No validation response is available." />;
  if (!result.valid) return <ErrorState title="Workflow validation failed" message={result.error || "The workflow is invalid."} />;
  return <WorkflowPlanPanel plan={result.plan} />;
}

function WorkflowJobTable({ plan }: { plan?: WorkflowPlan }) {
  const jobs = plan?.jobs ?? [];
  if (!jobs.length) return <EmptyState title="No jobs" detail="The workflow plan did not include planned jobs." />;
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Job</th><th>Needs</th><th>Runs on</th><th>Steps</th><th>Matrix</th></tr>
        </thead>
        <tbody>
          {jobs.map((job) => (
            <tr key={job.id}>
              <td>{job.name || job.id || "-"}</td>
              <td>{job.needs?.join(", ") || "-"}</td>
              <td>{job.runsOn?.join(", ") || "-"}</td>
              <td>{job.stepCount ?? 0}</td>
              <td>{job.matrix ? Object.entries(job.matrix).map(([key, value]) => `${key}=${value}`).join(", ") : "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function CommandList({ commands }: { commands?: Array<{ name?: string; command?: string; source?: string }> }) {
  if (!commands?.length) return <p className="muted">No command candidates.</p>;
  return (
    <div className="command-list">
      {commands.map((command, index) => (
        <code key={`${command.name || "command"}-${index}`}>{command.command || command.name || "-"}{command.source ? ` · ${command.source}` : ""}</code>
      ))}
    </div>
  );
}

function TagList({ values }: { values?: string[] }) {
  const unique = Array.from(new Set((values ?? []).filter(Boolean)));
  if (!unique.length) return <p className="muted">No values reported.</p>;
  return <div className="tag-list">{unique.map((value) => <span key={value}>{value}</span>)}</div>;
}

function WarningList({ values }: { values?: string[] }) {
  const unique = Array.from(new Set((values ?? []).filter(Boolean)));
  if (!unique.length) return <p className="muted">No warnings reported.</p>;
  return (
    <ul className="warning-list">
      {unique.map((value) => <li key={value}>{value}</li>)}
    </ul>
  );
}

function ArtifactTable({ payload }: { payload?: ArtifactListResponse }) {
  const artifacts = payload?.artifacts ?? [];
  if (!artifacts.length) return <EmptyState title="No artifacts" detail="No tracked artifact records were returned by the backend." />;
  return (
    <section className="panel">
      <h2>Tracked artifacts</h2>
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Name</th><th>Type</th><th>Version</th><th>Registry</th><th>Digest</th><th>Created</th></tr>
          </thead>
          <tbody>
            {artifacts.map((artifact) => (
              <tr key={artifact.id || artifact.reference}>
                <td>
                  <strong>{artifact.name || "-"}</strong>
                  <small>{artifact.reference || "-"}</small>
                </td>
                <td>{artifact.type || "-"}</td>
                <td>{artifact.version || "-"}</td>
                <td>{artifact.registry || artifact.repository || "-"}</td>
                <td>{artifact.digest ? <code>{shortDigest(artifact.digest)}</code> : <span className="muted">not pinned</span>}</td>
                <td>{formatDate(artifact.createdAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function PolicyResultTable({ payload }: { payload?: PolicyResultsResponse }) {
  const results = payload?.results ?? [];
  if (!results.length) return <EmptyState title="No policy results" detail="No stored policy decisions were returned by the backend." />;
  return (
    <section className="panel">
      <h2>Decisions</h2>
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Decision</th><th>Subject</th><th>Policy</th><th>Scope</th><th>Findings</th><th>Evaluated</th></tr>
          </thead>
          <tbody>
            {results.map((result) => (
              <tr key={result.id}>
                <td><StatusBadge status={result.decision} /></td>
                <td><code>{result.subjectType || "-"}/{result.subjectId || "-"}</code></td>
                <td>{result.policyId || "-"}</td>
                <td>{formatScope(result.projectId, result.environmentId)}</td>
                <td>{result.findings?.length ?? 0}</td>
                <td>{formatDate(result.evaluatedAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function EvidenceBundleTable({ payload }: { payload?: EvidenceBundleListResponse }) {
  const bundles = payload?.bundles ?? [];
  if (!bundles.length) return <EmptyState title="No evidence bundles" detail="No persisted evidence bundle metadata was returned by the backend." />;
  return (
    <section className="panel">
      <h2>Bundles</h2>
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Bundle</th><th>Subject</th><th>Scope</th><th>Digest</th><th>Generated</th><th>Summary</th></tr>
          </thead>
          <tbody>
            {bundles.map((bundle) => (
              <tr key={bundle.id}>
                <td><code>{bundle.id || "-"}</code></td>
                <td>{bundle.subjectType || "-"}/{bundle.subjectId || "-"}</td>
                <td>{bundle.scopeType && bundle.scopeId ? `${bundle.scopeType}:${bundle.scopeId}` : "-"}</td>
                <td>{bundle.digest ? <code>{shortDigest(bundle.digest)}</code> : "-"}</td>
                <td>{formatDate(bundle.generatedAt)}</td>
                <td>{bundle.summary || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="muted">{payload?.count ?? bundles.length} bundle(s) reported by the backend.</p>
    </section>
  );
}

function MCPRuntimeCard({ runtime }: { runtime?: SystemRuntimeStatus }) {
  return (
    <section className="summary-grid">
      <SummaryStrip summary={{
        title: runtime?.app || "Nivora runtime",
        status: { value: runtime?.runtime_mode || "unknown" },
        counts: {
          telemetry: runtime?.telemetry?.enabled ? 1 : 0
        },
        metadata: {
          environment: runtime?.environment || "-",
          tracing: runtime?.telemetry?.tracing || "-"
        }
      }} />
      <section className="panel compact">
        <h2>MCP boundary</h2>
        <StatusBadge status="read-only / guarded" />
        <p className="muted">This console only reads runtime, plugin, and integration metadata. Action tools remain governed by backend MCP policy and are not exposed here.</p>
      </section>
      <section className="panel compact">
        <h2>Telemetry</h2>
        <p>{runtime?.telemetry?.metrics_endpoint || "/metrics"}</p>
        <p className="muted">{runtime?.telemetry?.endpoint || "No external telemetry endpoint reported."}</p>
      </section>
    </section>
  );
}

function IntegrationTable({ payload }: { payload?: IntegrationListResponse }) {
  const integrations = payload?.integrations ?? [];
  if (!integrations.length) return <EmptyState title="No integrations" detail="No integration capability metadata was returned by the backend." />;
  return (
    <section className="panel">
      <h2>Integration boundaries</h2>
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Name</th><th>Type</th><th>Status</th><th>Maturity</th><th>Safe by default</th><th>Mutates</th><th>Capabilities</th></tr>
          </thead>
          <tbody>
            {integrations.map((integration) => (
              <tr key={integration.name}>
                <td>{integration.name || "-"}</td>
                <td>{integration.type || "-"}</td>
                <td><StatusBadge status={integration.status} /></td>
                <td>{integration.maturity || "-"}</td>
                <td>{integration.safeByDefault ? "yes" : "no"}</td>
                <td>{integration.mutatesExternalSystems ? "yes" : "no"}</td>
                <td>{integration.capabilities?.map((capability) => capability.name).filter(Boolean).join(", ") || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {payload?.warnings?.length ? <p className="muted">{payload.warnings.join(" ")}</p> : null}
    </section>
  );
}

function PluginTable({ plugins }: { plugins?: PluginManifest[] }) {
  if (!plugins?.length) return <EmptyState title="No plugins" detail="No plugin manifests were returned by the backend." />;
  return (
    <section className="panel">
      <h2>Plugin capability registry</h2>
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Name</th><th>Type</th><th>Status</th><th>Protocol</th><th>Version</th><th>Capabilities</th></tr>
          </thead>
          <tbody>
            {plugins.map((plugin) => (
              <tr key={plugin.name}>
                <td>{plugin.name || "-"}</td>
                <td>{plugin.type || "-"}</td>
                <td><StatusBadge status={plugin.status} /></td>
                <td>{plugin.protocol || "-"}</td>
                <td>{plugin.version || "-"}</td>
                <td>{plugin.capabilities?.map((capability) => capability.name).filter(Boolean).join(", ") || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function PipelineRunTable({ records }: { records?: PipelineRunRecord[] }) {
  if (!records?.length) return <EmptyState title="No PipelineRuns" detail="No PipelineRun records were returned by the backend." />;
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>ID</th><th>Pipeline</th><th>Status</th><th>Updated</th></tr>
        </thead>
        <tbody>
          {records.map((record) => (
            <tr key={record.run?.id}>
              <td><code>{record.run?.id || "-"}</code></td>
              <td>{record.run?.pipelineId || "-"}</td>
              <td><StatusBadge status={record.run?.status} /></td>
              <td>{formatDate(record.run?.updatedAt || record.run?.createdAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DeploymentRunTable({ records }: { records?: DeploymentRunRecord[] }) {
  if (!records?.length) return <EmptyState title="No DeploymentRuns" detail="No DeploymentRun records were returned by the backend." />;
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>ID</th><th>Target</th><th>Environment</th><th>Status</th><th>Reason</th></tr>
        </thead>
        <tbody>
          {records.map((record) => (
            <tr key={record.run?.id}>
              <td><code>{record.run?.id || "-"}</code></td>
              <td>{record.run?.targetType || record.run?.targetId || "-"}</td>
              <td>{record.run?.environmentId || "-"}</td>
              <td><StatusBadge status={record.run?.status} /></td>
              <td>{record.run?.reason || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ReleaseTable({ records, onSelect }: { records?: ReleaseRecord[]; onSelect: (id: string) => void }) {
  if (!records?.length) return <EmptyState title="No Releases" detail="No release records were returned by the backend." />;
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Release</th><th>Version</th><th>Environment</th><th>Status</th><th>Executions</th></tr>
        </thead>
        <tbody>
          {records.map((record) => {
            const id = record.release?.id || "";
            return (
              <tr key={id || record.release?.name}>
                <td><code>{id || "-"}</code></td>
                <td>{record.release?.version || "-"}</td>
                <td>{record.release?.environmentId || "-"}</td>
                <td><StatusBadge status={record.release?.status} /></td>
                <td><button className="table-action" type="button" disabled={!id} onClick={() => onSelect(id)}>Load</button></td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function ReleaseExecutionTable({ records }: { records?: ReleaseExecutionRecord[] }) {
  if (!records?.length) return <EmptyState title="No ReleaseExecutions" detail="No execution records were returned for this release." />;
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>ID</th><th>Release</th><th>Environment</th><th>Status</th><th>Reason</th></tr>
        </thead>
        <tbody>
          {records.map((record) => (
            <tr key={record.execution?.id}>
              <td><code>{record.execution?.id || "-"}</code></td>
              <td>{record.execution?.releaseId || "-"}</td>
              <td>{record.execution?.environmentId || "-"}</td>
              <td><StatusBadge status={record.execution?.status} /></td>
              <td>{record.execution?.reason || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TopologyPanel({ title, resources }: { title: string; resources?: ResourceNode[] }) {
  return (
    <section className="panel compact">
      <h2>{title}</h2>
      <ResourceTable resources={resources} />
    </section>
  );
}

function PageFrame({ title, eyebrow, description, children }: { title: string; eyebrow: string; description: string; children: ReactNode }) {
  return (
    <div className="page">
      <header className="page-header">
        <p className="eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        <p>{description}</p>
      </header>
      {children}
    </div>
  );
}

function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function formatScope(projectId?: string, environmentId?: string) {
  const parts = [];
  if (projectId) parts.push(`project:${projectId}`);
  if (environmentId) parts.push(`env:${environmentId}`);
  return parts.length ? parts.join(" / ") : "-";
}

function shortDigest(value: string) {
  if (value.length <= 28) return value;
  return `${value.slice(0, 18)}...${value.slice(-8)}`;
}

function LookupBar({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (value: string) => void; placeholder: string }) {
  return (
    <div className="lookup">
      <label>
        {label}
        <input value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} />
      </label>
    </div>
  );
}

function AsyncSummary<T extends DashboardSummary>({ state, fallbackTitle }: { state: AsyncState<T>; fallbackTitle: string }) {
  return <AsyncBlock state={state} render={(summary) => <SummaryStrip summary={summary ?? { title: fallbackTitle }} />} />;
}

function TopologySummary({ state }: { state: AsyncState<EnvironmentTopology> }) {
  return (
    <AsyncBlock state={state} render={(topology) => (
      <SummaryStrip summary={{
        title: `Environment ${topology?.environmentId ?? "-"}`,
        status: topology?.healthSummary?.status,
        counts: {
          applications: topology?.applications?.length ?? 0,
          targets: topology?.targets?.length ?? 0,
          deployments: topology?.latestDeployments?.length ?? 0,
          resources: topology?.resources?.length ?? 0
        },
        updatedAt: topology?.healthSummary?.updatedAt
      }} />
    )} />
  );
}

function HealthCard({ health }: { health?: HealthView }) {
  return (
    <section className="panel compact">
      <h2>Health</h2>
      <StatusBadge status={health?.status} />
      <pre>{JSON.stringify(health?.summary ?? {}, null, 2)}</pre>
    </section>
  );
}

function DiffCard({ diff }: { diff?: DiffView }) {
  const changes = useMemo(() => [
    ["added", diff?.addedResources?.length ?? 0],
    ["removed", diff?.removedResources?.length ?? 0],
    ["changed", diff?.changedResources?.length ?? 0],
    ["unknown", diff?.unknownLiveState?.length ?? 0]
  ], [diff]);
  return (
    <section className="panel compact">
      <h2>Diff</h2>
      <p>{diff?.summary || "No diff summary available."}</p>
      <div className="mini-metrics">
        {changes.map(([label, value]) => <span key={label}>{label}: <strong>{value}</strong></span>)}
      </div>
    </section>
  );
}

function ReleaseOverviewCard({ overview }: { overview?: ReleaseOverview }) {
  return (
    <section className="panel">
      <h2>Release overview</h2>
      <SummaryStrip summary={overview?.summary} />
    </section>
  );
}

type AsyncState<T> = {
  data?: T;
  error?: string;
  loading: boolean;
  reload: () => void;
};

type AsyncActionState<T> = {
  data?: T;
  error?: string;
  loading?: boolean;
};

function AsyncActionBlock<T>({ state, emptyTitle, emptyDetail, render }: { state?: AsyncActionState<T>; emptyTitle: string; emptyDetail: string; render: (data?: T) => React.ReactNode }) {
  if (!state) return <EmptyState title={emptyTitle} detail={emptyDetail} />;
  if (state.loading) return <LoadingState />;
  if (state.error) return <ErrorState message={state.error} />;
  return <>{render(state.data)}</>;
}

function AsyncBlock<T>({ state, render }: { state: AsyncState<T>; render: (data?: T) => React.ReactNode }) {
  if (state.loading) return <LoadingState />;
  if (state.error) return <ErrorState message={state.error} actionLabel="Retry" onAction={state.reload} />;
  return <>{render(state.data)}</>;
}

function useFetch<T>(loader: () => Promise<T>, deps: DependencyList = []): AsyncState<T> {
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<Omit<AsyncState<T>, "reload">>({ loading: true });

  useEffect(() => {
    let canceled = false;
    setState({ loading: true });
    loader()
      .then((data) => {
        if (!canceled) setState({ data, loading: false });
      })
      .catch((error: unknown) => {
        if (!canceled) setState({ error: error instanceof Error ? error.message : "Unknown error", loading: false });
      });
    return () => {
      canceled = true;
    };
  }, [...deps, attempt]);

  return {
    ...state,
    reload: () => setAttempt((value) => value + 1)
  };
}
