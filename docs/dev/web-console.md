# Web Console

Phase 6.4 adds the first Nivora web console foundation under `web/`.

The console is a React + TypeScript + Vite app that consumes existing backend runtime and visualization APIs. It does not introduce backend behavior, does not use mock production data, and does not claim to be a production UI.

On load, the app checks `/api/v1/version` before rendering the runtime pages. If the backend is not reachable, the console shows a single connection diagnostics panel with the configured API base, retry button, and local startup commands instead of flooding the page with repeated fetch failures.

## Run Locally

Start the backend:

```sh
make run-server
```

In another shell, start the web console:

```sh
make run-web
```

The Vite development server listens on `http://localhost:5173` and proxies `/api` requests to `http://localhost:8080`.

Run the web server from the repository target above or from the `web/` directory after installing dependencies. Starting Vite from another directory can make the React plugin resolve dependencies from the wrong `node_modules` tree.

To point the proxy at another backend port, keep browser requests on `/api` and change the proxy target:

```sh
cd web
NIVORA_WEB_PROXY_TARGET=http://localhost:18080 npm run dev
```

Avoid setting `VITE_NIVORA_API_BASE_URL` to a different origin unless the backend is configured for CORS. The local development path uses the Vite proxy to avoid cross-origin browser failures.

## Pages

- Dashboard
- PipelineRuns
- PipelineRun detail
- Deployments
- Deployment detail
- Releases
- Release execution detail
- Artifacts
- Repositories
- Repository detail, snapshots, intelligence, DevOps plan, and readiness review
- Workflows
- Workflow validate and plan
- Policy results
- Evidence bundles
- Runners
- Security
- Audit timeline
- Environment topology
- MCP safety

## API Usage

The console calls existing APIs only:

- Runtime APIs such as `/api/v1/pipeline-runs`, `/api/v1/deployments`, `/api/v1/releases`, and `/api/v1/runners`.
- Read-only catalog and governance APIs such as `/api/v1/artifacts`, `/api/v1/policies/results`, `/api/v1/evidence/bundles`, `/api/v1/integrations`, `/api/v1/plugins`, and `/api/v1/system/runtime`.
- Repository and workflow planning APIs such as `/api/v1/repositories`, `/api/v1/repositories/{id}/snapshots`, `/api/v1/repositories/{id}/intelligence`, `/api/v1/devops/plan`, `/api/v1/devops/readiness-review`, `/api/v1/workflows`, `/api/v1/workflows/validate`, and `/api/v1/workflows/plan`.
- Visualization APIs under `/api/v1/visualization/*` for DAGs, timelines, resources, health, audit, security, runner summaries, and environment topology.

Empty states and request errors are shown directly so contributors can see whether a backend capability is missing, unimplemented, unauthorized, or simply has no data yet. Network-level failures are handled by the global connection diagnostics view.

The MCP safety view is read-only. It shows runtime status plus integration and plugin capability metadata that the backend already exposes. It does not call MCP action tools, does not approve/apply/sync/rollback anything, and does not read secret values.

The repository and workflow views are also guarded by design. Repository DevOps plans and readiness reviews are explicit button actions, and workflow validate/plan calls are parser/planner paths only. They do not run repository code, start PipelineRuns, deploy, sync Argo CD, apply Kubernetes manifests, or approve governance gates.

## Checks

```sh
make verify-web
```

This runs:

- `npm ci`
- `npm run typecheck`
- `npm run build`
- `scripts/smoke-web-console.sh`

The smoke script starts a local backend and Vite dev server when they are not already running, then verifies that the console root is reachable, `/api/v1/version` works through the Vite proxy, and the APIs behind the artifact, repository, workflow, policy, evidence, integration, plugin, and runtime status pages return JSON. This catches the common local failure mode where Vite is started from the wrong directory or the backend is not reachable, which otherwise appears in the browser as repeated request failures.

## Limitations

- This is a web console foundation, not a complete product UI.
- Detail pages still require known IDs from existing backend records.
- Authentication UI, workflow execution UX, charts, and advanced filtering remain future work.
- The project remains a hardened beta-candidate foundation and is not production-ready.
