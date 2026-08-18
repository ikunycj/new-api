#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: bootstrap-config.sh [options]

Generate or validate the isolated stack .env, start only its PostgreSQL and
Redis services, and create the non-superuser new-api database role. Run as
root (normally: sudo bash ./bootstrap-config.sh). Secret values are never
printed.

Options:
  --deploy-dir PATH       Deployment directory (default: /opt/new-api)
  --env-file PATH         Env file relative to deploy dir (default: .env)
  --compose-file PATH     Compose file relative to deploy dir
                          (default: docker-compose.1panel.yml)
  --project-name NAME     Compose project name (default: ikun-new-api)
  --database NAME         Application database for a new install (new_api)
  --role NAME             Application login role for a new install (new_api_app)
  --admin-user NAME       PostgreSQL bootstrap administrator (ikun_pg_admin)
  --redis-db INDEX        Dedicated Redis logical database (default: 0)
EOF
}

DEPLOY_DIR='/opt/new-api'
ENV_FILE='.env'
COMPOSE_FILE='docker-compose.1panel.yml'
PROJECT_NAME='ikun-new-api'
DATABASE='new_api'
APP_ROLE='new_api_app'
ADMIN_USER='ikun_pg_admin'
REDIS_DB='0'

while (($#)); do
  case "$1" in
    --deploy-dir) DEPLOY_DIR=${2:?missing value}; shift 2 ;;
    --env-file) ENV_FILE=${2:?missing value}; shift 2 ;;
    --compose-file) COMPOSE_FILE=${2:?missing value}; shift 2 ;;
    --project-name) PROJECT_NAME=${2:?missing value}; shift 2 ;;
    --database) DATABASE=${2:?missing value}; shift 2 ;;
    --role) APP_ROLE=${2:?missing value}; shift 2 ;;
    --admin-user) ADMIN_USER=${2:?missing value}; shift 2 ;;
    --redis-db) REDIS_DB=${2:?missing value}; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[[ $EUID -eq 0 ]] || { echo 'Run this script as root (sudo).' >&2; exit 1; }
[[ $DEPLOY_DIR == /* ]] || { echo 'Deployment directory must be absolute' >&2; exit 2; }
[[ $ENV_FILE != /* && $ENV_FILE != *..* ]] || { echo 'Env file must stay inside deploy dir' >&2; exit 2; }
[[ $COMPOSE_FILE != /* && $COMPOSE_FILE != *..* ]] || { echo 'Compose file must stay inside deploy dir' >&2; exit 2; }
[[ $PROJECT_NAME =~ ^[A-Za-z0-9][A-Za-z0-9_-]*$ ]] || { echo 'Invalid compose project name' >&2; exit 2; }
[[ $DATABASE =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid database name' >&2; exit 2; }
[[ $APP_ROLE =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid application role' >&2; exit 2; }
[[ $ADMIN_USER =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid administrator role' >&2; exit 2; }
[[ $APP_ROLE != "$ADMIN_USER" ]] || { echo 'Application role must differ from administrator role' >&2; exit 2; }
[[ $REDIS_DB =~ ^[0-9]+$ && $REDIS_DB -le 15 ]] || { echo 'Redis database index must be 0..15' >&2; exit 2; }

command -v openssl >/dev/null || { echo 'openssl is required' >&2; exit 1; }
command -v sha256sum >/dev/null || { echo 'sha256sum is required' >&2; exit 1; }
docker info >/dev/null 2>&1 || { echo 'Docker is unavailable to root.' >&2; exit 1; }
docker compose version >/dev/null 2>&1 || { echo 'Docker Compose is unavailable.' >&2; exit 1; }

ENV_PATH="$DEPLOY_DIR/$ENV_FILE"
COMPOSE_PATH="$DEPLOY_DIR/$COMPOSE_FILE"
mkdir -p "$DEPLOY_DIR"
[[ -f $COMPOSE_PATH ]] || { echo "Missing compose file: $COMPOSE_PATH" >&2; exit 1; }

env_value() {
  local key=$1
  awk -F= -v wanted="$key" '$1 == wanted {print substr($0, index($0, "=") + 1); found++} END {if (found != 1) exit 1}' "$ENV_PATH"
}

require_value() {
  local key=$1
  local value
  value=$(env_value "$key") || { echo "Missing $key in $ENV_PATH" >&2; exit 1; }
  [[ -n $value ]] || { echo "$key is empty" >&2; exit 1; }
  printf '%s' "$value"
}

validate_secret() {
  local key=$1
  local value=$2
  [[ $value =~ ^[A-Fa-f0-9]{64}$ ]] || {
    echo "$key must be a 64-character hexadecimal value" >&2
    exit 1
  }
}

write_new_env() {
  local tmp_env
  local postgres_admin_password postgres_app_password redis_password
  local session_secret crypto_secret
  postgres_admin_password=$(openssl rand -hex 32)
  postgres_app_password=$(openssl rand -hex 32)
  redis_password=$(openssl rand -hex 32)
  session_secret=$(openssl rand -hex 32)
  crypto_secret=$(openssl rand -hex 32)

  tmp_env=$(mktemp "$DEPLOY_DIR/.env.tmp.XXXXXX")
  trap 'rm -f "$tmp_env"' RETURN
  umask 077
  cat >"$tmp_env" <<EOF
COMPOSE_PROJECT_NAME=$PROJECT_NAME
NETWORK_NAME=ikun-new-api-network
POSTGRES_VOLUME=ikun-new-api-postgres-data
REDIS_VOLUME=ikun-new-api-redis-data
APP_PORT=3000
METRICS_HOST_PORT=8006
METRICS_PORT=8006
IMAGE_TAG=new-api:ikun
CONTAINER=ikun-new-api
POSTGRES_IMAGE=postgres:18.4-alpine
POSTGRES_CONTAINER=ikun-new-api-postgres
POSTGRES_ADMIN_USER=$ADMIN_USER
POSTGRES_ADMIN_PASSWORD=$postgres_admin_password
POSTGRES_ADMIN_DB=postgres
POSTGRES_DB=$DATABASE
POSTGRES_APP_USER=$APP_ROLE
POSTGRES_APP_PASSWORD=$postgres_app_password
REDIS_IMAGE=redis:8.8.0
REDIS_CONTAINER=ikun-new-api-redis
REDIS_PASSWORD=$redis_password
REDIS_DB=$REDIS_DB
SESSION_SECRET=$session_secret
CRYPTO_SECRET=$crypto_secret
SESSION_COOKIE_SECURE=false
SESSION_COOKIE_TRUSTED_URL=
TRUSTED_REDIRECT_DOMAINS=ikun.love
TZ=Asia/Shanghai
NODE_NAME=ikun-new-api-1
NODE_TYPE=master
ERROR_LOG_ENABLED=true
BATCH_UPDATE_ENABLED=true
ENABLE_METRICS=true
METRICS_BIND_ADDRESS=0.0.0.0
SQL_MAX_IDLE_CONNS=10
SQL_MAX_OPEN_CONNS=30
SQL_MAX_LIFETIME=60
REDIS_POOL_SIZE=10
FAILOVER_PROMETHEUS_URL=
FAILOVER_ALERTMANAGER_URL=
FAILOVER_GRAFANA_PUBLIC_URL=
FAILOVER_MONITORING_USERNAME=
FAILOVER_MONITORING_PASSWORD=
FAILOVER_MONITORING_BEARER_TOKEN=
EOF
  chmod 600 "$tmp_env"
  mv -f "$tmp_env" "$ENV_PATH"
  trap - RETURN
}

if [[ -e $ENV_PATH ]]; then
  mode=$(stat -c '%a' "$ENV_PATH")
  [[ $mode == 600 ]] || { echo "Existing env file has mode $mode, expected 600" >&2; exit 1; }
else
  mkdir -p "$DEPLOY_DIR/data" "$DEPLOY_DIR/logs"
  chmod 750 "$DEPLOY_DIR" "$DEPLOY_DIR/data" "$DEPLOY_DIR/logs"
  write_new_env
fi

# Read persisted values so retries use exactly the same credentials and names.
COMPOSE_PROJECT_NAME=$(require_value COMPOSE_PROJECT_NAME)
NETWORK_NAME=$(require_value NETWORK_NAME)
POSTGRES_CONTAINER=$(require_value POSTGRES_CONTAINER)
POSTGRES_ADMIN_USER=$(require_value POSTGRES_ADMIN_USER)
POSTGRES_ADMIN_PASSWORD=$(require_value POSTGRES_ADMIN_PASSWORD)
POSTGRES_ADMIN_DB=$(require_value POSTGRES_ADMIN_DB)
POSTGRES_DB=$(require_value POSTGRES_DB)
POSTGRES_APP_USER=$(require_value POSTGRES_APP_USER)
POSTGRES_APP_PASSWORD=$(require_value POSTGRES_APP_PASSWORD)
REDIS_CONTAINER=$(require_value REDIS_CONTAINER)
REDIS_PASSWORD=$(require_value REDIS_PASSWORD)
REDIS_DB=$(require_value REDIS_DB)

validate_secret POSTGRES_ADMIN_PASSWORD "$POSTGRES_ADMIN_PASSWORD"
validate_secret POSTGRES_APP_PASSWORD "$POSTGRES_APP_PASSWORD"
validate_secret REDIS_PASSWORD "$REDIS_PASSWORD"
validate_secret SESSION_SECRET "$(require_value SESSION_SECRET)"
validate_secret CRYPTO_SECRET "$(require_value CRYPTO_SECRET)"
[[ $POSTGRES_ADMIN_USER =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid persisted PostgreSQL administrator' >&2; exit 1; }
[[ $POSTGRES_APP_USER =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid persisted PostgreSQL application role' >&2; exit 1; }
[[ $POSTGRES_APP_USER != "$POSTGRES_ADMIN_USER" ]] || { echo 'Application and administrator roles must differ' >&2; exit 1; }
[[ $POSTGRES_DB =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid persisted database name' >&2; exit 1; }
[[ $POSTGRES_ADMIN_DB =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid persisted administrator database name' >&2; exit 1; }
[[ $COMPOSE_PROJECT_NAME == ikun-new-api ]] || { echo 'Refusing a non-isolated Compose project name' >&2; exit 1; }
[[ $REDIS_DB =~ ^[0-9]+$ && $REDIS_DB -le 15 ]] || { echo 'Invalid persisted Redis database index' >&2; exit 1; }
[[ $POSTGRES_CONTAINER =~ ^[A-Za-z0-9_.-]+$ && $REDIS_CONTAINER =~ ^[A-Za-z0-9_.-]+$ ]] || {
  echo 'Invalid persisted container name' >&2
  exit 1
}
[[ $NETWORK_NAME == ikun-new-api-network ]] || { echo 'Refusing a non-isolated Docker network' >&2; exit 1; }
[[ $POSTGRES_CONTAINER == ikun-new-api-postgres && $REDIS_CONTAINER == ikun-new-api-redis ]] || {
  echo 'Refusing non-dedicated data-service container names' >&2
  exit 1
}

compose() {
  docker compose --project-name "$COMPOSE_PROJECT_NAME" --env-file "$ENV_PATH" -f "$COMPOSE_PATH" "$@"
}

compose up -d postgres redis >/dev/null

wait_healthy() {
  local container=$1
  local deadline=$((SECONDS + 180))
  local state health
  while ((SECONDS < deadline)); do
    state=$(docker inspect "$container" --format '{{.State.Status}}' 2>/dev/null || true)
    health=$(docker inspect "$container" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' 2>/dev/null || true)
    if [[ $state == running && $health == healthy ]]; then
      return 0
    fi
    sleep 2
  done
  echo "Dependency health timeout: container=${state:-missing} health=${health:-missing}" >&2
  docker logs --tail 100 "$container" >&2 2>/dev/null || true
  return 1
}

wait_healthy "$POSTGRES_CONTAINER"
wait_healthy "$REDIS_CONTAINER"

pg_admin() {
  local database=$1
  shift
  docker exec -e "PGPASSWORD=$POSTGRES_ADMIN_PASSWORD" "$POSTGRES_CONTAINER" \
    psql -U "$POSTGRES_ADMIN_USER" -d "$database" -v ON_ERROR_STOP=1 "$@"
}

role_flags=$(pg_admin "$POSTGRES_ADMIN_DB" -Atqc \
  "SELECT rolsuper || '|' || rolcreatedb || '|' || rolcreaterole || '|' || rolcanlogin FROM pg_roles WHERE rolname = '$POSTGRES_APP_USER'")
if [[ -z $role_flags ]]; then
  pg_admin "$POSTGRES_ADMIN_DB" -c \
    "CREATE ROLE \"$POSTGRES_APP_USER\" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD '$POSTGRES_APP_PASSWORD'" >/dev/null
else
  [[ $role_flags == 'f|f|f|t' ]] || {
    echo "Existing application role has unsafe flags: $role_flags" >&2
    exit 1
  }
  pg_admin "$POSTGRES_ADMIN_DB" -c \
    "ALTER ROLE \"$POSTGRES_APP_USER\" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD '$POSTGRES_APP_PASSWORD'" >/dev/null
fi

db_exists=$(pg_admin "$POSTGRES_ADMIN_DB" -Atqc \
  "SELECT 1 FROM pg_database WHERE datname = '$POSTGRES_DB'")
if [[ -z $db_exists ]]; then
  pg_admin "$POSTGRES_ADMIN_DB" -c \
    "CREATE DATABASE \"$POSTGRES_DB\" OWNER \"$POSTGRES_APP_USER\"" >/dev/null
else
  db_owner=$(pg_admin "$POSTGRES_ADMIN_DB" -Atqc \
    "SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = '$POSTGRES_DB'")
  [[ $db_owner == "$POSTGRES_APP_USER" ]] || {
    echo "Existing database owner is not the isolated application role: $db_owner" >&2
    exit 1
  }
fi

pg_admin "$POSTGRES_ADMIN_DB" -c \
  "REVOKE ALL ON DATABASE \"$POSTGRES_DB\" FROM PUBLIC; GRANT CONNECT ON DATABASE \"$POSTGRES_DB\" TO \"$POSTGRES_APP_USER\"" >/dev/null
pg_admin "$POSTGRES_DB" -c \
  "REVOKE ALL ON SCHEMA public FROM PUBLIC; GRANT USAGE, CREATE ON SCHEMA public TO \"$POSTGRES_APP_USER\"" >/dev/null

app_probe=$(docker exec -e "PGPASSWORD=$POSTGRES_APP_PASSWORD" "$POSTGRES_CONTAINER" \
  psql -U "$POSTGRES_APP_USER" -d "$POSTGRES_DB" -Atqc \
  "SELECT current_database() || '|' || current_user")
[[ $app_probe == "$POSTGRES_DB|$POSTGRES_APP_USER" ]] || {
  echo 'Application PostgreSQL probe did not use the isolated role/database' >&2
  exit 1
}

redis_ping=$(docker exec -e "REDISCLI_AUTH=$REDIS_PASSWORD" "$REDIS_CONTAINER" \
  redis-cli --no-auth-warning -n "$REDIS_DB" PING)
[[ $redis_ping == PONG ]] || { echo 'Dedicated Redis readiness check failed' >&2; exit 1; }

printf 'database=%s\n' "$POSTGRES_DB"
printf 'role=%s superuser=false createdb=false createrole=false\n' "$POSTGRES_APP_USER"
printf 'redis_database=%s\n' "$REDIS_DB"
printf 'env_mode='; stat -c '%a' "$ENV_PATH"
printf 'env_sha256='; sha256sum "$ENV_PATH" | awk '{print $1}'
echo 'status=bootstrapped'
