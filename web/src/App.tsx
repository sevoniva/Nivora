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
  DashboardSummary,
  DeploymentRunRecord,
  DiffView,
  EnvironmentTopology,
  GraphResponse,
  HealthView,
  PipelineRunRecord,
  ReleaseExecutionRecord,
  ReleaseRecord,
  ReleaseOverview,
  ResourceNode,
  RunnerSummary,
  SecuritySummary,
  TargetExecution,
  TimelineItem
} from "./types";

type Page =
  | "dashboard"
  | "pipelines"
  | "pipeline"
  | "deployments"
  | "deployment"
  | "releases"
  | "release-execution"
  | "runners"
  | "security"
  | "audit"
  | "environment";

const nav: Array<{ page: Page; label: string }> = [
  { page: "dashboard", label: "Dashboard" },
  { page: "pipelines", label: "PipelineRuns" },
  { page: "deployments", label: "Deployments" },
  { page: "releases", label: "Releases" },
  { page: "runners", label: "Runners" },
  { page: "security", label: "Security" },
  { page: "audit", label: "Audit" },
  { page: "environment", label: "Environment" }
];

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
            {page === "runners" ? <RunnersPage /> : null}
            {page === "security" ? <SecurityPage /> : null}
            {page === "audit" ? <AuditPage /> : null}
            {page === "environment" ? <EnvironmentPage /> : null}
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
