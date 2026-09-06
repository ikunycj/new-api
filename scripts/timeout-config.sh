#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
Usage:
  scripts/timeout-config.sh check [config]
  scripts/timeout-config.sh render [config] [output-dir]
EOF
  exit 2
}

CONFIG=${2:-config/timeouts.env}
OUTPUT_DIR=${3:-config/generated}
allowed=(TIMEOUT_CONFIG_VERSION RELAY_TIMEOUT STREAMING_TIMEOUT RELAY_IDLE_CONN_TIMEOUT STREAM_CLIENT_WRITE_TIMEOUT SHUTDOWN_TIMEOUT_SECONDS GATEWAY_CONNECT_TIMEOUT_SECONDS GATEWAY_READ_TIMEOUT_SECONDS GATEWAY_SEND_TIMEOUT_SECONDS UPSTREAM_FAIL_TIMEOUT_SECONDS)
values=()

is_allowed() {
  local key=$1 candidate
  for candidate in "${allowed[@]}"; do
    [[ "$candidate" == "$key" ]] && return 0
  done
  return 1
}

fail() { echo "timeout-config: $*" >&2; exit 1; }

get_value() {
  local wanted=$1 pair
  for pair in "${values[@]-}"; do
    if [[ ${pair%%=*} == "$wanted" ]]; then
      echo "${pair#*=}"
      return 0
    fi
  done
  return 1
}

has_key() {
  local wanted=$1 pair
  for pair in "${values[@]-}"; do
    [[ ${pair%%=*} == "$wanted" ]] && return 0
  done
  return 1
}

load_config() {
  [[ -f "$CONFIG" ]] || fail "config file not found: $CONFIG"
  local line key value
  while IFS= read -r line || [[ -n "$line" ]]; do
    line=${line%%#*}
    [[ -z "${line//[[:space:]]/}" ]] && continue
    [[ "$line" =~ ^[[:space:]]*([A-Z][A-Z0-9_]*)[[:space:]]*=[[:space:]]*([0-9]+)[[:space:]]*$ ]] || fail "invalid line"
    key=${BASH_REMATCH[1]}; value=${BASH_REMATCH[2]}
    is_allowed "$key" || fail "unknown key: $key"
    if has_key "$key"; then
      fail "duplicate key: $key"
    fi
    values+=("$key=$value")
  done < "$CONFIG"
  local key
  for key in "${allowed[@]}"; do
    has_key "$key" || fail "missing key: $key"
  done
  (( $(get_value TIMEOUT_CONFIG_VERSION) == 1 )) || fail "unsupported TIMEOUT_CONFIG_VERSION"
  (( $(get_value RELAY_TIMEOUT) >= 0 && $(get_value RELAY_TIMEOUT) <= 3600 )) || fail "RELAY_TIMEOUT must be 0..3600"
  (( $(get_value STREAMING_TIMEOUT) >= 1 && $(get_value STREAMING_TIMEOUT) <= 3600 )) || fail "STREAMING_TIMEOUT must be 1..3600"
  (( $(get_value RELAY_IDLE_CONN_TIMEOUT) >= 0 && $(get_value RELAY_IDLE_CONN_TIMEOUT) <= 3600 )) || fail "RELAY_IDLE_CONN_TIMEOUT must be 0..3600"
  (( $(get_value STREAM_CLIENT_WRITE_TIMEOUT) >= 1 && $(get_value STREAM_CLIENT_WRITE_TIMEOUT) <= 600 )) || fail "STREAM_CLIENT_WRITE_TIMEOUT must be 1..600"
  (( $(get_value SHUTDOWN_TIMEOUT_SECONDS) >= 1 && $(get_value SHUTDOWN_TIMEOUT_SECONDS) <= 900 )) || fail "SHUTDOWN_TIMEOUT_SECONDS must be 1..900"
  (( $(get_value GATEWAY_CONNECT_TIMEOUT_SECONDS) >= 1 && $(get_value GATEWAY_CONNECT_TIMEOUT_SECONDS) <= 300 )) || fail "GATEWAY_CONNECT_TIMEOUT_SECONDS must be 1..300"
  (( $(get_value GATEWAY_READ_TIMEOUT_SECONDS) >= $(get_value STREAMING_TIMEOUT) + 1 && $(get_value GATEWAY_READ_TIMEOUT_SECONDS) <= 3600 )) || fail "GATEWAY_READ_TIMEOUT_SECONDS must exceed STREAMING_TIMEOUT and be <=3600"
  (( $(get_value GATEWAY_SEND_TIMEOUT_SECONDS) >= 1 && $(get_value GATEWAY_SEND_TIMEOUT_SECONDS) <= 3600 )) || fail "GATEWAY_SEND_TIMEOUT_SECONDS must be 1..3600"
  (( $(get_value UPSTREAM_FAIL_TIMEOUT_SECONDS) >= 1 && $(get_value UPSTREAM_FAIL_TIMEOUT_SECONDS) <= 600 )) || fail "UPSTREAM_FAIL_TIMEOUT_SECONDS must be 1..600"
}

render() {
  mkdir -p "$OUTPUT_DIR"
  local app_tmp nginx_tmp
  app_tmp=$(mktemp "$OUTPUT_DIR/.app-timeouts.env.XXXXXX")
  nginx_tmp=$(mktemp "$OUTPUT_DIR/.openresty-timeouts.conf.XXXXXX")
  trap 'rm -f "$app_tmp" "$nginx_tmp"' EXIT
  {
    echo "# Generated from $CONFIG; do not edit."
    for key in RELAY_TIMEOUT STREAMING_TIMEOUT RELAY_IDLE_CONN_TIMEOUT STREAM_CLIENT_WRITE_TIMEOUT SHUTDOWN_TIMEOUT_SECONDS; do
      echo "$key=$(get_value "$key")"
    done
  } > "$app_tmp"
  {
    echo "# Generated from $CONFIG; include inside the gateway location/upstream."
    echo "proxy_connect_timeout $(get_value GATEWAY_CONNECT_TIMEOUT_SECONDS)s;"
    echo "proxy_read_timeout $(get_value GATEWAY_READ_TIMEOUT_SECONDS)s;"
    echo "proxy_send_timeout $(get_value GATEWAY_SEND_TIMEOUT_SECONDS)s;"
    echo "# Use UPSTREAM_FAIL_TIMEOUT_SECONDS for each upstream server fail_timeout."
    echo "# fail_timeout $(get_value UPSTREAM_FAIL_TIMEOUT_SECONDS)s;"
  } > "$nginx_tmp"
  mv "$app_tmp" "$OUTPUT_DIR/app-timeouts.env"
  mv "$nginx_tmp" "$OUTPUT_DIR/openresty-timeouts.conf"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$OUTPUT_DIR" && sha256sum app-timeouts.env openresty-timeouts.conf > timeout-config.sha256)
  else
    (cd "$OUTPUT_DIR" && shasum -a 256 app-timeouts.env openresty-timeouts.conf > timeout-config.sha256)
  fi
  trap - EXIT
  echo "✅ rendered timeout config to $OUTPUT_DIR"
}

case "${1:-}" in
  check) load_config; echo "✅ timeout config valid: $CONFIG" ;;
  render) load_config; render ;;
  *) usage ;;
esac
