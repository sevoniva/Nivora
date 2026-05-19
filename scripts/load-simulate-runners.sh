#!/usr/bin/env bash
set -euo pipefail

NIVORA_URL="${NIVORA_URL:-http://127.0.0.1:8080}"
NIVORA_LOAD_RUNNERS="${NIVORA_LOAD_RUNNERS:-10}"
NIVORA_LOAD_HEARTBEATS="${NIVORA_LOAD_HEARTBEATS:-20}"

python3 - "$NIVORA_URL" "$NIVORA_LOAD_RUNNERS" "$NIVORA_LOAD_HEARTBEATS" <<'PY'
import json
import sys
import time
import urllib.error
import urllib.request

base = sys.argv[1].rstrip("/")
runner_count = int(sys.argv[2])
heartbeat_count = int(sys.argv[3])
runners = []

for i in range(runner_count):
    runner = {
        "id": f"load-runner-{i}",
        "name": f"load-runner-{i}",
        "status": "online",
        "executors": ["shell"],
        "labels": {"load": "true"},
        "maxConcurrency": 1,
    }
    body = json.dumps(runner).encode()
    req = urllib.request.Request(base + "/api/v1/runners/register", data=body, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read().decode())
    except urllib.error.HTTPError as exc:
        print(f"runner register {i} failed: HTTP {exc.code} {exc.read().decode(errors='ignore')}", file=sys.stderr)
        sys.exit(1)
    runners.append((runner["id"], data["token"]["token"]))

latencies = []
for i in range(heartbeat_count):
    runner_id, token = runners[i % len(runners)]
    req = urllib.request.Request(
        base + f"/api/v1/runners/{runner_id}/heartbeat",
        data=b"",
        headers={"X-Nivora-Runner-Token": token},
        method="POST",
    )
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            resp.read()
    except urllib.error.HTTPError as exc:
        print(f"heartbeat {i} failed: HTTP {exc.code} {exc.read().decode(errors='ignore')}", file=sys.stderr)
        sys.exit(1)
    latencies.append((time.perf_counter() - started) * 1000)

latencies.sort()
avg = sum(latencies) / len(latencies) if latencies else 0
p95 = latencies[int(len(latencies) * 0.95) - 1] if latencies else 0
print(f"registered_runners={runner_count} heartbeats={heartbeat_count} avg_ms={avg:.2f} p95_ms={p95:.2f}")
PY
