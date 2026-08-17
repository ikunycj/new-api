#!/bin/sh
set -eu

cd "$(dirname "$0")"

: "${COST_FIRST_TOKEN_FILE:?Set COST_FIRST_TOKEN_FILE to keys for the cost-first package}"
: "${BALANCED_TOKEN_FILE:?Set BALANCED_TOKEN_FILE to keys for the balanced package}"
: "${STABILITY_FIRST_TOKEN_FILE:?Set STABILITY_FIRST_TOKEN_FILE to keys for the stability-first package}"
: "${REMOTE_BASE_URL:?Set REMOTE_BASE_URL to the approved remote API URL}"
if [ "${CONFIRM_REMOTE_LOADTEST:-}" != "yes" ]; then
  echo "Refusing remote strategy comparison: set CONFIRM_REMOTE_LOADTEST=yes after approving target, cost, rate limit, and test window." >&2
  exit 2
fi

profile="${1:-ramp}"
run_stamp=$(date -u +%Y%m%dT%H%M%SZ)

run_strategy() {
  strategy="$1"
  token_file="$2"
  echo "Running package strategy: $strategy"
  LOADTEST_TOKEN_FILE="$token_file" \
    LOADTEST_ROUTING_STRATEGY="$strategy" \
    LOADTEST_RUN_ID="strategy-${strategy}-${run_stamp}" \
    FAILOVER_MODE= \
    ./run-remote.sh "$profile"
  cp results/summary.json "results/${run_stamp}-${strategy}.json"
}

run_strategy cost_first "$COST_FIRST_TOKEN_FILE"
run_strategy balanced "$BALANCED_TOKEN_FILE"
run_strategy stability_first "$STABILITY_FIRST_TOKEN_FILE"

echo "Strategy summaries: results/${run_stamp}-*.json"
