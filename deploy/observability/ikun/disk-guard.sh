#!/bin/sh
set -eu

STACK_DIR=${STACK_DIR:-/opt/alltoken-observability/ikun}
LOKI_DATA_DIR=${LOKI_DATA_DIR:-$STACK_DIR/data/loki}
LOKI_MAX_KIB=${LOKI_MAX_KIB:-15728640}
ROOT_MIN_FREE_KIB=${ROOT_MIN_FREE_KIB:-20971520}

loki_kib=$(du -sk "$LOKI_DATA_DIR" 2>/dev/null | awk '{print $1}')
loki_kib=${loki_kib:-0}
root_free_kib=$(df -Pk / | awk 'NR == 2 {print $4}')

if [ "$loki_kib" -le "$LOKI_MAX_KIB" ] && [ "$root_free_kib" -ge "$ROOT_MIN_FREE_KIB" ]; then
  exit 0
fi

logger -t alltoken-observability-disk-guard "stopping ikun log collection: loki_kib=$loki_kib limit_kib=$LOKI_MAX_KIB root_free_kib=$root_free_kib minimum_kib=$ROOT_MIN_FREE_KIB"
cd "$STACK_DIR"
docker compose stop alloy loki
