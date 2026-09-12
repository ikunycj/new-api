# new-api load test

This stack is isolated from production and uses deterministic OpenAI-compatible mock upstreams. It creates test users and API tokens, drives requests with k6, writes the k6 summary to `results/summary.json`, and can expose the application API and pprof endpoints for inspection.

## Safety boundary

Seed data points only to `http://mock-upstream:8080` and the stack contains no real-provider credentials. Do not replace a mock channel with a real provider until the target, cost ceiling, rate limit, and test window have been explicitly approved.

## Prerequisites

- Docker Engine 24+ and Docker Compose v2
- At least 4 CPU cores and 8 GB RAM for the local stack
- Host ports in [`.env.example`](.env.example) available

## Start the local stack

```sh
cd deploy/loadtest
cp .env.example .env
./up.sh
```

Useful endpoints:

- Load-test API: `http://localhost:3100/api/status`
- pprof: `http://localhost:8005/debug/pprof/`
- Mock channel A: `http://localhost:18080`
- Mock channel B: `http://localhost:18081`

## Run profiles

```sh
./run.sh smoke            # 10 VUs for 1 minute
./run.sh step             # stepped concurrency
./run.sh steady           # sustained concurrency
./run.sh spike            # short ramp-up
./run.sh burst            # one request per configured user
./run.sh stream           # concurrent SSE connections
./run.sh mixed            # mixed stream, request, and model-list traffic
./run.sh soak             # long-running leak check
./run.sh capacity         # configured arrival-rate steps
./run.sh channel-failover # deterministic primary-to-fallback scenario
```

The default profile is `smoke`. Each run writes `results/summary.json`. The `capacity` profile holds each rate from `CAPACITY_RATES` after a short ramp and records request latency, errors, and token usage returned by the mock upstream. The default mock returns 10 prompt and 20 completion tokens per successful request.

## Deterministic channel failover

Each successful mock response consumes exactly 30 tokens. Channel A has a deliberately small balance so requests eventually move to Channel B:

```sh
curl -X POST 'http://localhost:18080/control/reset?tokens=300'
curl -X POST 'http://localhost:18081/control/reset?tokens=3000'
./run.sh channel-failover
```

Control the mock state directly:

```sh
curl -X POST 'http://localhost:18080/control/exhaust'
curl -X POST 'http://localhost:18080/control/disable'
curl -X POST 'http://localhost:18080/control/enable'
curl 'http://localhost:18080/control/state'
```

The expected sequence is Channel A consumption, a stable exhaustion error, and then a request through Channel B. Check the k6 summary and application logs for the request sequence and error code.

For a short capacity run:

```sh
CAPACITY_RATES=100,200,300,400,500,600 \
CAPACITY_RAMP_DURATION=10s \
CAPACITY_STAGE_DURATION=30s \
./run.sh capacity
```

For planning, hold every step for at least two minutes and repeat the highest passing rate for 30 minutes. Do not point the profile at a paid or production provider without explicit approval.

## Local k6 against a remote API

The load generator can run locally while the target API is remote. Use dedicated remote test tokens and the confirmation gate:

```sh
CONFIRM_REMOTE_LOADTEST=yes \
REMOTE_BASE_URL=https://approved-test-api.example.com \
LOADTEST_TOKENS=sk-test-1,sk-test-2,sk-test-3 \
CAPACITY_RATES=100,200,300 \
./run-remote.sh capacity
```

Alternatively set `LOADTEST_TOKEN_FILE=/absolute/path/tokens.txt`; each line may contain a token or `name token`. The file is read locally and is not part of the repository. `run-remote.sh` only runs k6 and does not start or seed local application services.

The startup scripts bind host ports to `127.0.0.1`. Go module download settings are configurable in `.env` through `GOPROXY`, `GOSUMDB`, and `GOTOOLCHAIN`. On Colima, `forward-colima-ports.sh` repairs the API, database, Redis, mock, and pprof forwards when needed.

## Inspecting a run

1. Compare completed requests, errors, latency percentiles, and token usage in `results/summary.json`.
2. Compare application logs with the mock control state when testing failover.
3. Use pprof for point-in-time captures, for example:

   ```sh
   go tool pprof http://localhost:8005/debug/pprof/profile?seconds=30
   ```

4. Treat a run as invalid when the load generator itself is CPU-, memory-, network-, or file-descriptor-bound.

## Shutdown and reset

```sh
./down.sh   # stop containers and keep volumes
./reset.sh  # stop containers and remove only this stack's volumes
```

`reset.sh` removes only the Docker volumes declared by this Compose file (`loadtest_postgres`); it does not touch normal development volumes.
