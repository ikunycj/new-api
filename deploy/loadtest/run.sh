#!/bin/sh
set -eu

cd "$(dirname "$0")"

profile="${1:-smoke}"
case "$profile" in
  smoke|step|steady|spike|burst|stream|mixed|soak|capacity|channel-failover) ;;
  *)
    echo "Unknown profile: $profile" >&2
    echo "Valid profiles: smoke step steady spike burst stream mixed soak capacity channel-failover" >&2
    exit 2
    ;;
esac

mkdir -p results
export COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-4}"

docker compose -f compose.yml build mock-upstream mock-upstream-b new-api
docker compose -f compose.yml up -d postgres redis mock-upstream mock-upstream-b new-api
sh ./forward-colima-ports.sh
docker compose -f compose.yml run --rm seed

echo "Load-test API: http://localhost:${LOADTEST_API_PORT:-3100}/api/status"
echo "pprof:         http://localhost:${LOADTEST_PPROF_PORT:-8005}/debug/pprof/"
echo "Running k6 profile: $profile"

LOAD_PROFILE="$profile" docker compose -f compose.yml run --rm k6
