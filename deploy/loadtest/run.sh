#!/bin/sh
set -eu

cd "$(dirname "$0")"

profile="${1:-smoke}"
case "$profile" in
  smoke|step|steady|spike|burst|stream|mixed|soak|capacity) ;;
  *)
    echo "Unknown profile: $profile" >&2
    echo "Valid profiles: smoke step steady spike burst stream mixed soak capacity" >&2
    exit 2
    ;;
esac

mkdir -p results
export COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-4}"

docker compose -f compose.yml build mock-upstream alert-sink new-api
docker compose -f compose.yml up -d postgres redis mock-upstream pyroscope new-api postgres-exporter redis-exporter cadvisor alert-sink alertmanager prometheus grafana
sh ./forward-colima-ports.sh
docker compose -f compose.yml run --rm seed

echo "Grafana:      http://localhost:${LOADTEST_GRAFANA_PORT:-3001}/d/new-api-loadtest"
echo "Prometheus:   http://localhost:${LOADTEST_PROMETHEUS_PORT:-9090}"
echo "Alertmanager: http://localhost:${LOADTEST_ALERTMANAGER_PORT:-9093}"
echo "Alert log:    http://localhost:${LOADTEST_ALERT_SINK_PORT:-19094}/alerts"
echo "Pyroscope:    http://localhost:${LOADTEST_PYROSCOPE_PORT:-4040}"
echo "Running k6 profile: $profile"

LOAD_PROFILE="$profile" docker compose -f compose.yml run --rm k6
