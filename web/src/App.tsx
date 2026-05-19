import { useEffect, useMemo, useState } from "react";
import type { DependencyList, ReactNode } from "react";
import { api } from "./api";
import {
  DAGPlaceholder,
  EmptyState,
  ErrorState,
  LoadingState,
  ResourceTable,
  RunnerTable,
  StatusBadge,
  SummaryStrip,
  TargetTable,
  Timeline
} from "./components";
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

type Page = "dashboard" | "pipeline" | "deployment" | "release" | "runners" | "security";

const nav: Array<{ page: Page; label: string }> = [
  { page: "dashboard", label: "Dashboard" },
  { page: "pipeline", label: "PipelineRun" },
  { page: "deployment", label: "Deployment" },
  { page: "release", label: "Release execution" },
  { page: "runners", label: "Runners" },
  { page: "security", label: "Security" }
];

export function App() {
  const [page, setPage] = useState<Page>("dashboard");

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
        <p className="sidebar-note">Phase 4.1 web foundation. Backend APIs remain the source of truth.</p>
      </aside>
      <main>
        {page === "dashboard" ? <DashboardPage /> : null}
        {page === "pipeline" ? <PipelinePage /> : null}
        {page === "deployment" ? <DeploymentPage /> : null}
        {page === "release" ? <ReleasePage /> : null}
        {page === "runners" ? <RunnersPage /> : null}
        {page === "security" ? <SecurityPage /> : null}
      </main>
    </div>
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

function ReleasePage() {
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
  return (
    <PageFrame title="Runners" eyebrow="Execution plane" description="Runner status and executor capability summary.">
      <AsyncBlock state={state} render={(summary) => (
        <>
          <SummaryStrip summary={summary} />
          <section className="panel"><h2>Runner table</h2><RunnerTable summary={summary} /></section>
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
};

function AsyncBlock<T>({ state, render }: { state: AsyncState<T>; render: (data?: T) => React.ReactNode }) {
  if (state.loading) return <LoadingState />;
  if (state.error) return <ErrorState message={state.error} />;
  return <>{render(state.data)}</>;
}

function useFetch<T>(loader: () => Promise<T>, deps: DependencyList = []): AsyncState<T> {
  const [state, setState] = useState<AsyncState<T>>({ loading: true });

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
  }, deps);

  return state;
}
