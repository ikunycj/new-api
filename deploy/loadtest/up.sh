#!/bin/sh
set -eu

cd "$(dirname "$0")"
mkdir -p results
export COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-4}"

# Build the application before pulling the monitoring stack. This keeps large
# Grafana/Prometheus image downloads from starving `go mod download`.
docker compose -f compose.yml build mock-upstream alert-sink new-api
docker compose -f compose.yml up -d postgres redis mock-upstream pyroscope new-api postgres-exporter redis-exporter cadvisor alert-sink alertmanager prometheus grafana
sh ./forward-colima-ports.sh
docker compose -f compose.yml run --rm seed

echo "Grafana:      http://localhost:${LOADTEST_GRAFANA_PORT:-3001}/d/new-api-loadtest"
echo "Prometheus:   http://localhost:${LOADTEST_PROMETHEUS_PORT:-9090}"
echo "Alertmanager: http://localhost:${LOADTEST_ALERTMANAGER_PORT:-9093}"
echo "Alert log:    http://localhost:${LOADTEST_ALERT_SINK_PORT:-19094}/alerts"
echo "Pyroscope:    http://localhost:${LOADTEST_PYROSCOPE_PORT:-4040}"
