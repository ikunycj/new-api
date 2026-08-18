#!/usr/bin/env bash
set -Eeuo pipefail

LOCAL_STATUS_URL=${LOCAL_STATUS_URL:-http://127.0.0.1:3000/api/status}
PUBLIC_STATUS_URL=${PUBLIC_STATUS_URL:-}
EXISTING_LOCAL_HEALTH_URL=${EXISTING_LOCAL_HEALTH_URL:-http://127.0.0.1:8080/api/v1/settings/public}
EXISTING_PUBLIC_HEALTH_URL=${EXISTING_PUBLIC_HEALTH_URL:-https://ikun.love/api/v1/settings/public}
CONTAINER=${CONTAINER:-ikun-new-api}
POSTGRES_CONTAINER=${POSTGRES_CONTAINER:-1Panel-postgresql-8Kr6}
REDIS_CONTAINER=${REDIS_CONTAINER:-1Panel-redis-xsdn}

if docker info >/dev/null 2>&1; then
  DOCKER=(docker)
elif sudo -n docker info >/dev/null 2>&1; then
  DOCKER=(sudo -n docker)
else
  echo 'Docker is unavailable for the current user and passwordless sudo.' >&2
  exit 1
fi

printf 'local status: '
curl -fsS "$LOCAL_STATUS_URL" >/dev/null
echo 'ok'

state=$("${DOCKER[@]}" inspect "$CONTAINER" --format '{{.State.Status}}')
health=$("${DOCKER[@]}" inspect "$CONTAINER" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}')
[[ $state == running && $health == healthy ]] || {
  echo "new-api is not healthy: state=$state health=$health" >&2
  exit 1
}
"${DOCKER[@]}" inspect "$CONTAINER" --format 'container={{.Name}} state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restarts={{.RestartCount}}'

postgres_state=$("${DOCKER[@]}" inspect "$POSTGRES_CONTAINER" --format '{{.State.Status}}')
postgres_health=$("${DOCKER[@]}" inspect "$POSTGRES_CONTAINER" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}')
[[ $postgres_state == running && $postgres_health == healthy ]] || {
  echo "PostgreSQL is not healthy: state=$postgres_state health=$postgres_health" >&2
  exit 1
}
redis_state=$("${DOCKER[@]}" inspect "$REDIS_CONTAINER" --format '{{.State.Status}}')
[[ $redis_state == running ]] || {
  echo "Redis is not running: state=$redis_state" >&2
  exit 1
}
echo 'postgres/redis dependencies: ok'

[[ $(systemctl is-active sub2api.service) == active ]] || {
  echo 'sub2api.service is not active' >&2
  exit 1
}
curl -fsS "$EXISTING_LOCAL_HEALTH_URL" >/dev/null
curl -fsS "$EXISTING_PUBLIC_HEALTH_URL" >/dev/null
echo 'sub2api local/public status: ok'

if [[ -n $PUBLIC_STATUS_URL ]]; then
  curl -fsS "$PUBLIC_STATUS_URL" >/dev/null
  echo 'new-api public status: ok'
else
  echo 'new-api public status: skipped (loopback-only deployment)'
fi
