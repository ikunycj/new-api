#!/bin/sh
set -eu

cd "$(dirname "$0")"
export COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-4}"

# Build the application before starting the load-test services so dependency
# downloads do not compete with unrelated container startup work.
docker compose -f compose.yml build mock-upstream mock-upstream-b new-api
docker compose -f compose.yml up -d postgres redis mock-upstream mock-upstream-b new-api
sh ./forward-colima-ports.sh
docker compose -f compose.yml run --rm seed

echo "Load-test API: http://localhost:${LOADTEST_API_PORT:-3100}/api/status"
echo "pprof:         http://localhost:${LOADTEST_PPROF_PORT:-8005}/debug/pprof/"
