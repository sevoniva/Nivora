# Web UI

Phase 4.1 adds a minimal React + TypeScript + Vite web UI foundation under `web/`.

Phase 6.4 extends that foundation into the first web console surface. See [Web Console](web-console.md) for the current page map and API usage.

The web UI consumes existing runtime and visualization APIs. It does not introduce new backend runtime behavior, does not claim production readiness, and does not include credentials or sample secrets.

## Run Locally

Start the backend:

```sh
make run-server
```

In another shell, start the web app:

```sh
make run-web
```

The Vite dev server listens on `http://localhost:5173` and proxies `/api` requests to `http://localhost:8080`.

To point the proxy at another backend port, keep browser requests on `/api` and change the proxy target:

```sh
cd web
NIVORA_WEB_PROXY_TARGET=http://localhost:18080 npm run dev
```

Avoid setting `VITE_NIVORA_API_BASE_URL` to another origin unless the backend is configured for CORS. The local development path uses the Vite proxy to avoid cross-origin browser failures.

## Checks

```sh
make verify-web
```

This runs:

- `npm ci`
- `npm run typecheck`
- `npm run build`

## Current Pages

- Dashboard
- PipelineRuns
- PipelineRun detail
- Deployments
- Deployment detail
- Releases
- Release execution detail
- Runners
- Security summary
- Audit timeline
- Environment topology

## Current Components

- Timeline
- DAG placeholder
- Status badge
- Resource table
- Runner table
- Finding table

## Limitations

- This is a minimal UI foundation, not a complete product frontend.
- Pages require existing backend records and IDs for detail views.
- Empty, loading, and error states are intentionally visible.
- There is no authentication UI yet.
- There is no charting library or heavy design system.
