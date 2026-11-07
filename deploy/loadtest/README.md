# new-api isolated test stack

This stack is isolated from production and uses deterministic OpenAI-compatible mock upstreams. It creates test users and API tokens and exposes the application API, mock controls, and pprof endpoints for integration and performance testing with an external client.

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

## Deterministic channel failover

Each successful mock response consumes exactly 30 tokens. Channel A has a deliberately small balance so requests eventually move to Channel B:

```sh
curl -X POST 'http://localhost:18080/control/reset?tokens=300'
curl -X POST 'http://localhost:18081/control/reset?tokens=3000'
```

Control the mock state directly:

```sh
curl -X POST 'http://localhost:18080/control/exhaust'
curl -X POST 'http://localhost:18080/control/disable'
curl -X POST 'http://localhost:18080/control/enable'
curl 'http://localhost:18080/control/state'
```

Send requests with a seeded token through the local API:

```sh
curl 'http://localhost:3100/v1/chat/completions' \
  -H 'Authorization: Bearer sk-loadtest00001' \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}'
```

Each successful mock response consumes exactly 30 tokens. Repeated requests eventually exhaust Channel A and exercise fallback to Channel B. Compare application logs with the mock control state to verify the request sequence and error code.

The startup scripts bind host ports to `127.0.0.1`. Go module download settings are configurable in `.env` through `GOPROXY`, `GOSUMDB`, and `GOTOOLCHAIN`. On Colima, `forward-colima-ports.sh` repairs the API, database, Redis, mock, and pprof forwards when needed.

## Inspecting a run

1. Compare the external client's request, error, latency, and token-usage results.
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
