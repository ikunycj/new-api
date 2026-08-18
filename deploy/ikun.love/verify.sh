#!/usr/bin/env bash
set -Eeuo pipefail

LOCAL_STATUS_URL=${LOCAL_STATUS_URL:-http://127.0.0.1:3000/api/status}
PUBLIC_STATUS_URL=${PUBLIC_STATUS_URL:-https://ikun.love/api/status}
CONTAINER=${CONTAINER:-new-api}

printf 'local status: '
curl -fsS "$LOCAL_STATUS_URL" >/dev/null
echo 'ok'

printf 'public status: '
curl -fsS "$PUBLIC_STATUS_URL" >/dev/null
echo 'ok'

docker inspect "$CONTAINER" --format 'container={{.Name}} state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restarts={{.RestartCount}}'
