import type {
  DashboardSummary,
  GraphResponse,
  ResourceNode,
  RunnerSummary,
  StatusBadgeModel,
  TargetExecution,
  TimelineItem
} from "./types";

export function StatusBadge({ status }: { status?: StatusBadgeModel | string }) {
  const model = typeof status === "string" ? { value: status } : status;
  const value = model?.value || "Unknown";
  const tone = model?.tone || "neutral";
  return <span className={`status status-${tone}`}>{value}</span>;
}

export function SummaryStrip({ summary }: { summary?: DashboardSummary }) {
  if (!summary) {
    return <EmptyState title="No summary" detail="The backend did not return a summary for this view." />;
  }
  const counts = Object.entries(summary.counts ?? {});
  return (
    <section className="summary-strip">
      <div>
        <p className="eyebrow">Summary</p>
        <h2>{summary.title}</h2>
      </div>
      <StatusBadge status={summary.status} />
      {counts.map(([label, value]) => (
        <div className="metric" key={label}>
          <span>{label}</span>
          <strong>{value}</strong>
        </div>
      ))}
    </section>
  );
}

export function Timeline({ items }: { items?: TimelineItem[] }) {
  if (!items?.length) {
    return <EmptyState title="No timeline yet" detail="Create or run a delivery object, then refresh this view." />;
  }
  return (
    <ol className="timeline">
      {items.map((item) => (
        <li key={item.id}>
          <div className="timeline-time">{formatTime(item.time)}</div>
          <div className="timeline-body">
            <div className="timeline-title">
              <span>{item.type}</span>
              <StatusBadge status={item.status} />
            </div>
            <p>{item.message || item.subject || "Lifecycle event"}</p>
            {item.subject ? <code>{item.subject}</code> : null}
          </div>
        </li>
      ))}
    </ol>
  );
}

export function DAGPlaceholder({ graph }: { graph?: GraphResponse }) {
  if (!graph?.nodes?.length) {
    return <EmptyState title="No graph nodes" detail="The PipelineRun DAG is empty or the run was not found." />;
  }
  return (
    <div className="dag">
      <div className="dag-lane">
        {graph.nodes.map((node) => (
          <div className="dag-node" key={node.id}>
            <span>{node.type}</span>
            <strong>{node.label}</strong>
            <StatusBadge status={node.status} />
          </div>
        ))}
      </div>
      <p className="muted">{graph.edges?.length ?? 0} relationship(s) returned by the backend DAG API.</p>
    </div>
  );
}

export function ResourceTable({ resources }: { resources?: ResourceNode[] }) {
  if (!resources?.length) {
    return <EmptyState title="No resources" detail="No resource inventory is available for this view." />;
  }
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Kind</th>
            <th>Name</th>
            <th>Namespace</th>
            <th>Status</th>
            <th>Health</th>
          </tr>
        </thead>
        <tbody>
          {resources.map((resource) => (
            <tr key={resource.id}>
              <td>{resource.type}</td>
              <td>{resource.name}</td>
              <td>{resource.namespace || "-"}</td>
              <td><StatusBadge status={resource.status} /></td>
              <td><StatusBadge status={resource.health} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function RunnerTable({ summary }: { summary?: RunnerSummary }) {
  return <ResourceTable resources={summary?.runners} />;
}

export function TargetTable({ targets }: { targets?: TargetExecution[] }) {
  if (!targets?.length) {
    return <EmptyState title="No targets" detail="No target execution rows are available." />;
  }
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Order</th>
            <th>Target</th>
            <th>Type</th>
            <th>Status</th>
            <th>DeploymentRun</th>
          </tr>
        </thead>
        <tbody>
          {targets.map((target, index) => (
            <tr key={`${target.targetId || target.targetName}-${index}`}>
              <td>{target.order ?? index + 1}</td>
              <td>{target.targetName || target.targetId || "-"}</td>
              <td>{target.targetType || "-"}</td>
              <td><StatusBadge status={target.status} /></td>
              <td>{target.deploymentRunId ? <code>{target.deploymentRunId}</code> : "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function EmptyState({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="empty">
      <strong>{title}</strong>
      <p>{detail}</p>
    </div>
  );
}

export function ErrorState({ message }: { message: string }) {
  return (
    <div className="error">
      <strong>Request failed</strong>
      <p>{message}</p>
    </div>
  );
}

export function LoadingState() {
  return <div className="loading">Loading visualization data...</div>;
}

function formatTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value || "-";
  }
  return date.toLocaleString();
}
