#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: bootstrap-config.sh [options]

Create the isolated new-api PostgreSQL database/role, verify an empty Redis
logical database, and write /opt/new-api/.env atomically. Run as root (the
normal invocation is `sudo bash ./bootstrap-config.sh`). No secret values are
printed.
EOF
}

DEPLOY_DIR='/opt/new-api'
ENV_FILE='.env'
POSTGRES_CONTAINER='1Panel-postgresql-8Kr6'
REDIS_CONTAINER='1Panel-redis-xsdn'
DATABASE='new_api'
ROLE='new_api_app'
REDIS_DB='1'
IMAGE_TAG='new-api:ikun'
APP_CONTAINER='ikun-new-api'

while (($#)); do
  case "$1" in
    --deploy-dir) DEPLOY_DIR=${2:?missing value}; shift 2 ;;
    --env-file) ENV_FILE=${2:?missing value}; shift 2 ;;
    --postgres-container) POSTGRES_CONTAINER=${2:?missing value}; shift 2 ;;
    --redis-container) REDIS_CONTAINER=${2:?missing value}; shift 2 ;;
    --database) DATABASE=${2:?missing value}; shift 2 ;;
    --role) ROLE=${2:?missing value}; shift 2 ;;
    --redis-db) REDIS_DB=${2:?missing value}; shift 2 ;;
    --image-tag) IMAGE_TAG=${2:?missing value}; shift 2 ;;
    --container) APP_CONTAINER=${2:?missing value}; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[[ $EUID -eq 0 ]] || { echo 'Run this script as root (sudo).' >&2; exit 1; }
[[ $DATABASE =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid database name' >&2; exit 2; }
[[ $ROLE =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid role name' >&2; exit 2; }
[[ $REDIS_DB =~ ^[0-9]+$ ]] || { echo 'Invalid Redis database index' >&2; exit 2; }
[[ $ENV_FILE != /* && $ENV_FILE != *..* ]] || { echo 'Env file must stay inside deploy dir' >&2; exit 2; }

if docker info >/dev/null 2>&1; then
  DOCKER=(docker)
else
  echo 'Docker is unavailable to root.' >&2
  exit 1
fi
command -v openssl >/dev/null || { echo 'openssl is required' >&2; exit 1; }

ENV_PATH="$DEPLOY_DIR/$ENV_FILE"
if [[ -e $ENV_PATH ]]; then
  mode=$(stat -c '%a' "$ENV_PATH")
  [[ $mode == 600 ]] || { echo "Existing env file has mode $mode, expected 600" >&2; exit 1; }
  echo 'status=already-configured'
  printf 'env_sha256='; sha256sum "$ENV_PATH" | awk '{print $1}'
  exit 0
fi

mkdir -p "$DEPLOY_DIR/data" "$DEPLOY_DIR/logs"
chmod 750 "$DEPLOY_DIR" "$DEPLOY_DIR/data" "$DEPLOY_DIR/logs"

pg_env=$("${DOCKER[@]}" inspect "$POSTGRES_CONTAINER" --format '{{range .Config.Env}}{{println .}}{{end}}')
pg_user=$(printf '%s\n' "$pg_env" | awk -F= '$1=="POSTGRES_USER"{print substr($0,index($0,"=")+1)}')
pg_password=$(printf '%s\n' "$pg_env" | awk -F= '$1=="POSTGRES_PASSWORD"{print substr($0,index($0,"=")+1)}')
[[ -n $pg_user && -n $pg_password ]] || { echo 'Could not read PostgreSQL container credentials internally' >&2; exit 1; }

redis_password=$("${DOCKER[@]}" inspect "$REDIS_CONTAINER" --format '{{range .Config.Cmd}}{{println .}}{{end}}' | awk '$1=="--requirepass"{getline; print; exit}')
[[ -n $redis_password ]] || { echo 'Could not read Redis authentication internally' >&2; exit 1; }
[[ $redis_password =~ ^[A-Za-z0-9._~-]+$ ]] || { echo 'Redis password needs URL encoding; refusing to guess' >&2; exit 1; }

pg() {
  local database=$1
  shift
  "${DOCKER[@]}" exec -e "PGPASSWORD=$pg_password" "$POSTGRES_CONTAINER" psql \
    -U "$pg_user" -d "$database" -v ON_ERROR_STOP=1 "$@"
}

role_exists=$(pg postgres -Atqc "SELECT 1 FROM pg_roles WHERE rolname = '$ROLE'")
db_exists=$(pg postgres -Atqc "SELECT 1 FROM pg_database WHERE datname = '$DATABASE'")
[[ -z $role_exists && -z $db_exists ]] || {
  echo 'Target database or role already exists; refusing to overwrite unknown state' >&2
  exit 1
}

redis_ping=$("${DOCKER[@]}" exec "$REDIS_CONTAINER" redis-cli --no-auth-warning -a "$redis_password" -n "$REDIS_DB" PING)
[[ $redis_ping == PONG ]] || { echo 'Redis authentication/readiness check failed' >&2; exit 1; }
redis_keys=$("${DOCKER[@]}" exec "$REDIS_CONTAINER" redis-cli --no-auth-warning -a "$redis_password" -n "$REDIS_DB" DBSIZE)
[[ $redis_keys == 0 ]] || { echo "Redis database $REDIS_DB is not empty; refusing to share it" >&2; exit 1; }

db_password=$(openssl rand -hex 32)
session_secret=$(openssl rand -hex 32)
crypto_secret=$(openssl rand -hex 32)

pg postgres -c "CREATE ROLE \"$ROLE\" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD '$db_password'" >/dev/null
pg postgres -c "CREATE DATABASE \"$DATABASE\" OWNER \"$ROLE\"" >/dev/null
pg postgres -c "REVOKE ALL ON DATABASE \"$DATABASE\" FROM PUBLIC; GRANT CONNECT ON DATABASE \"$DATABASE\" TO \"$ROLE\"" >/dev/null
pg "$DATABASE" -c "REVOKE ALL ON SCHEMA public FROM PUBLIC; GRANT USAGE, CREATE ON SCHEMA public TO \"$ROLE\"" >/dev/null

role_flags=$(pg postgres -Atqc "SELECT rolsuper || '|' || rolcreatedb || '|' || rolcreaterole || '|' || rolcanlogin FROM pg_roles WHERE rolname = '$ROLE'")
[[ $role_flags == 'f|f|f|t' ]] || {
  echo 'New-api role flags are not restricted as expected' >&2
  exit 1
}

app_probe=$("${DOCKER[@]}" exec -e "PGPASSWORD=$db_password" "$POSTGRES_CONTAINER" psql \
  -U "$ROLE" -d "$DATABASE" -Atqc "SELECT current_database() || '|' || current_user")
[[ $app_probe == "$DATABASE|$ROLE" ]] || {
  echo 'PostgreSQL bootstrap probe did not use the isolated application role' >&2
  exit 1
}

tmp_env=$(mktemp "$DEPLOY_DIR/.env.tmp.XXXXXX")
trap 'rm -f "$tmp_env"' EXIT
umask 077
cat > "$tmp_env" <<EOF
APP_PORT=3000
METRICS_HOST_PORT=8006
METRICS_PORT=8006
SQLITE_PATH=/data/new-api.db?_busy_timeout=30000
SQL_DSN=postgresql://${ROLE}:${db_password}@${POSTGRES_CONTAINER}:5432/${DATABASE}?sslmode=disable
REDIS_CONN_STRING=redis://:${redis_password}@${REDIS_CONTAINER}:6379/${REDIS_DB}
SESSION_SECRET=${session_secret}
CRYPTO_SECRET=${crypto_secret}
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
IMAGE_TAG=${IMAGE_TAG}
CONTAINER=${APP_CONTAINER}
EOF
chmod 600 "$tmp_env"
mv -f "$tmp_env" "$ENV_PATH"
trap - EXIT

printf 'database=%s\n' "$DATABASE"
printf 'role=%s superuser=false createdb=false createrole=false\n' "$ROLE"
printf 'redis_database=%s keys=%s\n' "$REDIS_DB" "$redis_keys"
printf 'env_mode='; stat -c '%a' "$ENV_PATH"
printf 'env_sha256='; sha256sum "$ENV_PATH" | awk '{print $1}'
echo 'status=bootstrapped'
