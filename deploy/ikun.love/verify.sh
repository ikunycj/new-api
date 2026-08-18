#!/usr/bin/env bash
set -Eeuo pipefail

DEPLOY_DIR=${DEPLOY_DIR:-/opt/new-api}
ENV_FILE=${ENV_FILE:-.env}
COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.1panel.yml}
LOCAL_STATUS_URL=${LOCAL_STATUS_URL:-http://127.0.0.1:3000/api/status}
PUBLIC_STATUS_URL=${PUBLIC_STATUS_URL:-}
EXPECTED_BINARY_SHA=${EXPECTED_BINARY_SHA:-}
EXISTING_LOCAL_HEALTH_URL=${EXISTING_LOCAL_HEALTH_URL:-http://127.0.0.1:8080/api/v1/settings/public}
EXISTING_PUBLIC_HEALTH_URL=${EXISTING_PUBLIC_HEALTH_URL:-https://ikun.love/api/v1/settings/public}

cd "$DEPLOY_DIR"
ENV_PATH="$DEPLOY_DIR/$ENV_FILE"
COMPOSE_PATH="$DEPLOY_DIR/$COMPOSE_FILE"
[[ -f $ENV_PATH && -f $COMPOSE_PATH ]] || { echo 'Deployment env or compose file is missing' >&2; exit 1; }
env_mode=$(stat -c '%a' "$ENV_PATH")
[[ $env_mode == 600 ]] || { echo "Refusing verification: env mode is $env_mode, expected 600" >&2; exit 1; }

# The bootstrap-generated file is root-owned deployment state. It contains
# only this stack's credentials; do not print or copy its values.
set -a
. "$ENV_PATH"
set +a

APP_CONTAINER=${CONTAINER:-ikun-new-api}
POSTGRES_CONTAINER=${POSTGRES_CONTAINER:-ikun-new-api-postgres}
REDIS_CONTAINER=${REDIS_CONTAINER:-ikun-new-api-redis}
NETWORK_NAME=${NETWORK_NAME:-ikun-new-api-network}
POSTGRES_ADMIN_USER=${POSTGRES_ADMIN_USER:?POSTGRES_ADMIN_USER is required}
POSTGRES_ADMIN_PASSWORD=${POSTGRES_ADMIN_PASSWORD:?POSTGRES_ADMIN_PASSWORD is required}
POSTGRES_ADMIN_DB=${POSTGRES_ADMIN_DB:-postgres}
POSTGRES_DB=${POSTGRES_DB:?POSTGRES_DB is required}
POSTGRES_APP_USER=${POSTGRES_APP_USER:?POSTGRES_APP_USER is required}
POSTGRES_APP_PASSWORD=${POSTGRES_APP_PASSWORD:?POSTGRES_APP_PASSWORD is required}
REDIS_PASSWORD=${REDIS_PASSWORD:?REDIS_PASSWORD is required}
REDIS_DB=${REDIS_DB:-0}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-ikun-new-api}
POSTGRES_VOLUME=${POSTGRES_VOLUME:-ikun-new-api-postgres-data}
REDIS_VOLUME=${REDIS_VOLUME:-ikun-new-api-redis-data}

if [[ -n $EXPECTED_BINARY_SHA && ! $EXPECTED_BINARY_SHA =~ ^[0-9a-fA-F]{64}$ ]]; then
  echo 'EXPECTED_BINARY_SHA must be a SHA-256 value' >&2
  exit 2
fi

if docker info >/dev/null 2>&1; then
  DOCKER=(docker)
elif sudo -n docker info >/dev/null 2>&1; then
  DOCKER=(sudo -n docker)
else
  echo 'Docker is unavailable for the current user and passwordless sudo.' >&2
  exit 1
fi

compose() {
  "${DOCKER[@]}" compose --project-name "$COMPOSE_PROJECT_NAME" --env-file "$ENV_PATH" -f "$COMPOSE_PATH" "$@"
}

container_state() {
  local container=$1
  "${DOCKER[@]}" inspect "$container" --format '{{.State.Status}}'
}

container_health() {
  local container=$1
  "${DOCKER[@]}" inspect "$container" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
}

app_state=$(container_state "$APP_CONTAINER")
app_health=$(container_health "$APP_CONTAINER")
[[ $app_state == running && $app_health == healthy ]] || {
  echo "new-api is not healthy: state=$app_state health=$app_health" >&2
  exit 1
}
"${DOCKER[@]}" inspect "$APP_CONTAINER" --format 'container={{.Name}} state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restarts={{.RestartCount}} image={{.Config.Image}}'

if [[ -n $EXPECTED_BINARY_SHA ]]; then
  runtime_dir=$(mktemp -d /tmp/ikun-new-api-runtime.XXXXXX)
  trap 'case "$runtime_dir" in /tmp/ikun-new-api-runtime.*) rm -rf -- "$runtime_dir" ;; *) exit 1 ;; esac' EXIT
  "${DOCKER[@]}" cp "$APP_CONTAINER:/new-api" "$runtime_dir/new-api"
  runtime_sha=$(sha256sum "$runtime_dir/new-api" | awk '{print tolower($1)}')
  [[ $runtime_sha == ${EXPECTED_BINARY_SHA,,} ]] || {
    echo "Runtime binary SHA mismatch: $runtime_sha" >&2
    exit 1
  }
  echo "runtime_binary_sha256=$runtime_sha"
fi

postgres_state=$(container_state "$POSTGRES_CONTAINER")
postgres_health=$(container_health "$POSTGRES_CONTAINER")
[[ $postgres_state == running && $postgres_health == healthy ]] || {
  echo "Dedicated PostgreSQL is not healthy: state=$postgres_state health=$postgres_health" >&2
  exit 1
}
redis_state=$(container_state "$REDIS_CONTAINER")
redis_health=$(container_health "$REDIS_CONTAINER")
[[ $redis_state == running && $redis_health == healthy ]] || {
  echo "Dedicated Redis is not healthy: state=$redis_state health=$redis_health" >&2
  exit 1
}
echo 'dedicated postgres/redis: healthy'

network_members=$("${DOCKER[@]}" network inspect "$NETWORK_NAME" --format '{{range $id, $container := .Containers}}{{println $container.Name}}{{end}}')
for expected in "$APP_CONTAINER" "$POSTGRES_CONTAINER" "$REDIS_CONTAINER"; do
  grep -Fxq "$expected" <<<"$network_members" || {
    echo "Container $expected is not attached to $NETWORK_NAME" >&2
    exit 1
  }
done
if grep -Eq '(^|[[:space:]])(1Panel-|sub2api)' <<<"$network_members"; then
  echo 'Dedicated network unexpectedly contains a Sub2API/1Panel container' >&2
  exit 1
fi
echo "network=$NETWORK_NAME isolated"

"${DOCKER[@]}" volume inspect "$POSTGRES_VOLUME" >/dev/null || {
  echo "Dedicated PostgreSQL volume is missing: $POSTGRES_VOLUME" >&2
  exit 1
}
"${DOCKER[@]}" volume inspect "$REDIS_VOLUME" >/dev/null || {
  echo "Dedicated Redis volume is missing: $REDIS_VOLUME" >&2
  exit 1
}
echo "volumes=$POSTGRES_VOLUME,$REDIS_VOLUME present"

app_binding=$("${DOCKER[@]}" port "$APP_CONTAINER" 3000/tcp)
expected_binding="127.0.0.1:${APP_PORT:-3000}"
grep -Fq "$expected_binding" <<<"$app_binding" || {
  echo "new-api is not loopback-bound: $app_binding" >&2
  exit 1
}

pg_admin() {
  local database=$1
  shift
  "${DOCKER[@]}" exec -e "PGPASSWORD=$POSTGRES_ADMIN_PASSWORD" "$POSTGRES_CONTAINER" \
    psql -U "$POSTGRES_ADMIN_USER" -d "$database" -v ON_ERROR_STOP=1 "$@"
}

role_flags=$(pg_admin "$POSTGRES_ADMIN_DB" -Atqc \
  "SELECT rolsuper || '|' || rolcreatedb || '|' || rolcreaterole || '|' || rolcanlogin FROM pg_roles WHERE rolname = '$POSTGRES_APP_USER'")
[[ $role_flags == 'f|f|f|t' ]] || {
  echo "Application PostgreSQL role flags are unsafe: $role_flags" >&2
  exit 1
}
db_owner=$(pg_admin "$POSTGRES_ADMIN_DB" -Atqc \
  "SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = '$POSTGRES_DB'")
[[ $db_owner == "$POSTGRES_APP_USER" ]] || {
  echo "Application database owner mismatch: $db_owner" >&2
  exit 1
}
app_probe=$("${DOCKER[@]}" exec -e "PGPASSWORD=$POSTGRES_APP_PASSWORD" "$POSTGRES_CONTAINER" \
  psql -U "$POSTGRES_APP_USER" -d "$POSTGRES_DB" -Atqc \
  "SELECT current_database() || '|' || current_user")
[[ $app_probe == "$POSTGRES_DB|$POSTGRES_APP_USER" ]] || {
  echo 'Application PostgreSQL probe failed' >&2
  exit 1
}
redis_ping=$("${DOCKER[@]}" exec -e "REDISCLI_AUTH=$REDIS_PASSWORD" "$REDIS_CONTAINER" \
  redis-cli --no-auth-warning -n "$REDIS_DB" PING)
[[ $redis_ping == PONG ]] || { echo 'Application Redis probe failed' >&2; exit 1; }
echo "database=$POSTGRES_DB role=$POSTGRES_APP_USER redis_database=$REDIS_DB"

[[ $(systemctl is-active sub2api.service 2>/dev/null) == active ]] || {
  echo 'sub2api.service is not active' >&2
  exit 1
}
ss -lnt 2>/dev/null | awk '$4 ~ /:8080$/ {found=1} END {exit found ? 0 : 1}' || {
  echo 'Sub2API port 8080 is not bound' >&2
  exit 1
}
curl -fsS "$EXISTING_LOCAL_HEALTH_URL" >/dev/null
curl -fsS "$EXISTING_PUBLIC_HEALTH_URL" >/dev/null
echo 'sub2api local/public status: ok'

curl -fsS "$LOCAL_STATUS_URL" >/dev/null
echo 'new-api local status: ok'
if [[ -n $PUBLIC_STATUS_URL ]]; then
  curl -fsS "$PUBLIC_STATUS_URL" >/dev/null
  echo 'new-api public status: ok'
else
  echo 'new-api public status: skipped (loopback-only deployment)'
fi
