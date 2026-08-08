# new-api 1000-user load test

This stack is isolated from production and uses a local deterministic OpenAI-compatible upstream. It creates 1000 test users and API tokens, drives requests with k6, stores metrics in Prometheus, visualizes them in Grafana, sends alerts through Alertmanager, and collects Go profiles with pprof and Pyroscope.

## Safety boundary

The seed data points only to `http://mock-upstream:8080`. The stack has no real-provider credentials. Do not replace the mock channel with a real provider until cost, provider rate limits, and the target environment have been explicitly approved.

## Prerequisites

- Docker Engine 24+ and Docker Compose v2
- At least 4 CPU cores and 8 GB RAM for the complete observability stack
- For a 10,000-VU burst, use at least 8 CPU cores and 12 GB RAM, or distribute k6 across multiple load generators
- Host ports in [`.env.example`](.env.example) available

## Start monitoring

```sh
cd deploy/loadtest
cp .env.example .env
./up.sh
```

Open:

- Grafana: http://localhost:3001/d/new-api-loadtest (anonymous viewer access, or `admin` / `GRAFANA_ADMIN_PASSWORD`)
- Prometheus: http://localhost:9090
- Alertmanager: http://localhost:9093
- Alert sink: http://localhost:19094/alerts
- Pyroscope: http://localhost:4040
- Application metrics: http://localhost:8006/metrics
- Load-test API: http://localhost:3100/api/status
- pprof: http://localhost:8005/debug/pprof/

## Run profiles

```sh
./run.sh smoke   # 10 VUs for 1 minute, validates the stack
./run.sh step    # 50 -> 100 -> 250 -> 500 -> 750 -> 1000 VUs
./run.sh steady  # 1000 VUs for 30 minutes
./run.sh spike   # ramp from 50 to 1000 VUs in 10 seconds
./run.sh burst   # LOADTEST_USERS VUs, one request each
./run.sh stream  # 1000 concurrent SSE connections
./run.sh mixed   # 700 stream + 200 non-stream + 100 model-list VUs
./run.sh soak    # 500 VUs for 2 hours, leak check
./run.sh capacity # configured RPS steps against the single mock channel
```

The default is `smoke`. Results are written to `results/summary.json`. The k6 container publishes its metrics to Prometheus through remote write, so the Grafana panel can correlate load with server and dependency metrics.

The `capacity` profile uses an arrival-rate executor instead of VUs as the target. It holds each rate from `CAPACITY_RATES` after a short ramp and records input, output, and total Token TPS from the upstream `usage` response. The default mock returns 10 prompt and 20 completion tokens per successful request, so output Token TPS should be approximately `successful RPS * 20`. The highest valid step is the last rate with no dropped iterations, error rate below 1%, latency within the configured thresholds, and no sustained database or Redis pool alerts.

For a short preliminary run:

```sh
CAPACITY_RATES=100,200,300,400,500,600 \
CAPACITY_RAMP_DURATION=10s \
CAPACITY_STAGE_DURATION=30s \
./run.sh capacity
```

For a capacity result suitable for planning, keep each step for at least two minutes and repeat the highest passing rate for 30 minutes. Do not point this profile at a paid or production provider until the account, cost ceiling, rate limit, and maintenance window have been approved.

### Local k6 against a remote API

The load generator can run locally while the target API is remote. Use dedicated remote test tokens and an explicit confirmation gate:

```sh
CONFIRM_REMOTE_LOADTEST=yes \
REMOTE_BASE_URL=https://approved-test-api.example.com \
LOADTEST_TOKENS=sk-test-1,sk-test-2,sk-test-3 \
K6_PROMETHEUS_RW_SERVER_URL=https://approved-prometheus.example.com/api/v1/write \
CAPACITY_RATES=100,200,300 \
./run-remote.sh capacity
```

`run-remote.sh` does not start or seed the local new-api/PostgreSQL/Redis services; it only runs k6 on the existing Compose network. The remote Prometheus endpoint must accept authenticated remote-write traffic, or keep the default local endpoint and inspect only local k6 metrics. To see remote CPU, memory, Go, PostgreSQL and Redis panels, the remote Prometheus must scrape the corresponding remote exporters and the remote Grafana must use that Prometheus. Keep metrics and exporter endpoints on a private network or VPN; never expose them publicly just for a load test.

The startup scripts build `new-api` before pulling the monitoring images and use `goproxy.cn` by default. The Go proxy is configurable in `.env`; for a private mirror set `GOPROXY`, `GOSUMDB`, and optionally `GOTOOLCHAIN` there. A successful `go mod download` is cached by the legacy Docker layer keyed by `go.mod` and `go.sum`, so source-only changes do not download modules again. Host ports are bound to `127.0.0.1` so the stack stays local. On Colima, the scripts also repair missing SSH port forwards for long-lived VMs.

`Dockerfile.dev` builds for the container's native architecture and does not enable experimental Go runtime features. This avoids emulation overhead on Apple Silicon and keeps profiling stable during long runs.

The default application pool is 80 open and 40 idle PostgreSQL connections because the bundled PostgreSQL service defaults to 100 total connections. Increase both only after raising the database limit and watching connection pressure in Grafana.

## Alert rehearsal

To verify the full alert path without a load test, restart the application or inject mock errors:

```sh
MOCK_ERROR_RATE=1 docker compose -f compose.yml up -d mock-upstream
curl -s http://localhost:19094/alerts | jq
```

Restore the normal mock with `MOCK_ERROR_RATE=0 docker compose -f compose.yml up -d mock-upstream`. Alert rules cover application availability, relay 5xx rate, latency, in-flight requests, Go heap/goroutines, database pool utilization and waits, Redis pool timeouts, exporter availability, PostgreSQL connection pressure, Redis rejected connections, and unexpected mock errors.

## What to inspect during a run

1. In Grafana, watch relay RPS, P95/P99, in-flight requests, database pool waits, Redis pool timeouts, Go heap/goroutines, and container CPU/memory.
2. If P95 rises while mock latency stays flat, compare database/Redis waits and CPU. A flat RPS with rising latency marks the saturation point.
3. Open Pyroscope and inspect `new-api-loadtest` profiles for CPU, allocations, goroutines, mutex and block contention.
4. Use pprof endpoints for a point-in-time capture, for example `go tool pprof http://localhost:8005/debug/pprof/profile?seconds=30`.
5. Check `curl http://localhost:19094/alerts` for the last Alertmanager notification and inspect Prometheus `/alerts` for firing/resolved state.

## Shutdown and reset

```sh
./down.sh             # stop containers, keep volumes
./reset.sh            # stop and delete load-test volumes/data
```

`reset.sh` removes only the Docker volumes declared by this stack (`loadtest_*`); it does not touch the project's normal development volumes.

If a legacy-builder host reports `no space left on device`, inspect usage with `docker system df`, then run `docker image prune -f` to remove dangling images. This intentionally remains a manual recovery step because pruning also discards reusable Go dependency layers and makes the next build slower.
