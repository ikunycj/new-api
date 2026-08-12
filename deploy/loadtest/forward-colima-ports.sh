#!/bin/sh
set -eu

[ "$(docker context show 2>/dev/null)" = "colima" ] || exit 0

ssh_config="${HOME}/.colima/_lima/colima/ssh.config"
[ -f "$ssh_config" ] || exit 0

forward_service_port() {
  published_address=$(docker compose -f compose.yml port "$1" "$2" 2>/dev/null || true)
  [ -n "$published_address" ] || return 0

  host_port=${published_address##*:}
  /usr/bin/ssh -F "$ssh_config" -O forward \
    -L "127.0.0.1:${host_port}:127.0.0.1:${host_port}" lima-colima \
    >/dev/null 2>&1 || true
}

# Colima normally creates these forwards automatically. Re-requesting them is
# harmless and repairs the stale-control-socket failure seen on long-lived VMs.
forward_service_port postgres 5432
forward_service_port redis 6379
forward_service_port mock-upstream 8080
forward_service_port pyroscope 4040
forward_service_port new-api 3000
forward_service_port new-api 8005
forward_service_port new-api 8006
forward_service_port prometheus 9090
forward_service_port alert-sink 8080
forward_service_port alertmanager 9093
forward_service_port grafana 3000
