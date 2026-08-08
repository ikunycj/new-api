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
if [ -n "${LOADTEST_TOKEN_FILE:-}" ]; then
  if [ ! -f "$LOADTEST_TOKEN_FILE" ]; then
    echo "Token file does not exist: $LOADTEST_TOKEN_FILE" >&2
    exit 2
  fi
  token_values=$(awk 'NF {if (NF > 2) exit 2; print (NF == 2 ? $2 : $1)}' "$LOADTEST_TOKEN_FILE") || {
    echo "Token file must contain one token per line or a name followed by a token." >&2
    exit 2
  }
  token_count=$(printf '%s\n' "$token_values" | awk 'NF {count++} END {print count + 0}')
  if [ "$token_count" -lt 1 ]; then
    echo "Token file contains no tokens." >&2
    exit 2
  fi
  LOADTEST_TOKENS=$(printf '%s\n' "$token_values" | paste -sd, -)
  export LOADTEST_TOKENS
else
  : "${LOADTEST_TOKENS:?Set LOADTEST_TOKENS or LOADTEST_TOKEN_FILE to dedicated remote test tokens}"
fi
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
