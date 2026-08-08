#!/bin/sh
set -eu

cd "$(dirname "$0")"

profile="${1:-capacity}"
case "$profile" in
  smoke|step|steady|spike|burst|stream|mixed|soak|capacity) ;;
  *)
    echo "Unknown profile: $profile" >&2
    echo "Valid profiles: smoke step steady spike burst stream mixed soak capacity" >&2
    exit 2
    ;;
esac

: "${REMOTE_BASE_URL:?Set REMOTE_BASE_URL to the approved remote API URL}"
: "${LOADTEST_TOKENS:?Set LOADTEST_TOKENS to dedicated remote test tokens, comma-separated}"
if [ "${CONFIRM_REMOTE_LOADTEST:-}" != "yes" ]; then
  echo "Refusing remote load test: set CONFIRM_REMOTE_LOADTEST=yes after approving target, cost, rate limit, and test window." >&2
  exit 2
fi

mkdir -p results
export COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-4}"
export BASE_URL="$REMOTE_BASE_URL"
export LOAD_PROFILE="$profile"
export K6_PROMETHEUS_RW_SERVER_URL="${K6_PROMETHEUS_RW_SERVER_URL:-http://prometheus:9090/api/v1/write}"

echo "Remote target: $BASE_URL"
echo "Prometheus remote write: $K6_PROMETHEUS_RW_SERVER_URL"
echo "Running k6 profile: $profile"
docker compose -f compose.yml run --rm --no-deps k6
