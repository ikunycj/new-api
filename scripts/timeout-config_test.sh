#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/timeout-config-test.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT

cp "$script_dir/../config/timeouts.env" "$tmp_dir/valid.env"
"$script_dir/timeout-config.sh" check "$tmp_dir/valid.env"
"$script_dir/timeout-config.sh" render "$tmp_dir/valid.env" "$tmp_dir/generated"
grep -q '^proxy_read_timeout 600s;$' "$tmp_dir/generated/openresty-timeouts.conf"

sed 's/GATEWAY_READ_TIMEOUT_SECONDS=600/GATEWAY_READ_TIMEOUT_SECONDS=300/' "$tmp_dir/valid.env" > "$tmp_dir/invalid.env"
if "$script_dir/timeout-config.sh" check "$tmp_dir/invalid.env" >/dev/null 2>&1; then
  echo 'expected gateway/read timeout validation to fail' >&2
  exit 1
fi

echo '✅ timeout config tests passed'
