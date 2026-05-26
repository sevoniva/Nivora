# Web Console

Phase 6.4 adds the first Nivora web console foundation under `web/`.

The console is a React + TypeScript + Vite app that consumes existing backend runtime and visualization APIs. It does not introduce backend behavior, does not use mock production data, and does not claim to be a production UI.

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
- Runners
- Security
- Audit timeline
- Environment topology

## API Usage

The console calls existing APIs only:

- Runtime APIs such as `/api/v1/pipeline-runs`, `/api/v1/deployments`, `/api/v1/releases`, and `/api/v1/runners`.
- Visualization APIs under `/api/v1/visualization/*` for DAGs, timelines, resources, health, audit, security, runner summaries, and environment topology.

Empty states and request errors are shown directly so contributors can see whether a backend capability is missing, unimplemented, or simply has no data yet.

## Checks

```sh
make verify-web
```

This runs:

- `npm ci`
- `npm run typecheck`
- `npm run build`

## Limitations

- This is a web console foundation, not a complete product UI.
- Detail pages still require known IDs from existing backend records.
- Authentication UI, write workflows, charts, and advanced filtering remain future work.
- The project remains a hardened beta-candidate foundation and is not production-ready.
