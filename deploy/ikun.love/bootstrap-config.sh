#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: bootstrap-config.sh [options]

Validate the administrator-reviewed isolated stack .env, start only its
PostgreSQL and Redis services, and create the non-superuser new-api database
role. Run as root (normally: sudo bash ./bootstrap-config.sh). Secret values
are never printed or generated on the server.

Options:
  --deploy-dir PATH       Deployment directory (default: /opt/new-api)
  --env-file PATH         Env file relative to deploy dir (default: .env)
  --compose-file PATH     Compose file relative to deploy dir
                          (default: docker-compose.1panel.yml)
  --expected-env-sha SHA  Required SHA-256 of the reviewed local env file
EOF
}

DEPLOY_DIR='/opt/new-api'
ENV_FILE='.env'
COMPOSE_FILE='docker-compose.1panel.yml'
EXPECTED_ENV_SHA=''

while (($#)); do
  case "$1" in
    --deploy-dir) DEPLOY_DIR=${2:?missing value}; shift 2 ;;
    --env-file) ENV_FILE=${2:?missing value}; shift 2 ;;
    --compose-file) COMPOSE_FILE=${2:?missing value}; shift 2 ;;
    --expected-env-sha) EXPECTED_ENV_SHA=${2:?missing value}; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[[ $EUID -eq 0 ]] || { echo 'Run this script as root (sudo).' >&2; exit 1; }
[[ $DEPLOY_DIR == /* ]] || { echo 'Deployment directory must be absolute' >&2; exit 2; }
[[ $ENV_FILE != /* && $ENV_FILE != *..* ]] || { echo 'Env file must stay inside deploy dir' >&2; exit 2; }
[[ $COMPOSE_FILE != /* && $COMPOSE_FILE != *..* ]] || { echo 'Compose file must stay inside deploy dir' >&2; exit 2; }
[[ $EXPECTED_ENV_SHA =~ ^[0-9a-fA-F]{64}$ ]] || { echo 'A reviewed env SHA-256 is required' >&2; exit 2; }

command -v sha256sum >/dev/null || { echo 'sha256sum is required' >&2; exit 1; }
command -v ss >/dev/null || { echo 'ss is required for host-port collision checks' >&2; exit 1; }
docker info >/dev/null 2>&1 || { echo 'Docker is unavailable to root.' >&2; exit 1; }
docker compose version >/dev/null 2>&1 || { echo 'Docker Compose is unavailable.' >&2; exit 1; }

ENV_PATH="$DEPLOY_DIR/$ENV_FILE"
COMPOSE_PATH="$DEPLOY_DIR/$COMPOSE_FILE"
[[ -d $DEPLOY_DIR && ! -L $DEPLOY_DIR ]] || {
  echo "Deployment directory must be an installed, non-symlink directory: $DEPLOY_DIR" >&2
  exit 1
}
[[ -f $COMPOSE_PATH ]] || { echo "Missing compose file: $COMPOSE_PATH" >&2; exit 1; }
[[ -f $ENV_PATH && ! -L $ENV_PATH ]] || {
  echo "Install the reviewed deploy/ikun.love/.env before bootstrap: $ENV_PATH" >&2
  exit 1
}
mode=$(stat -c '%a' "$ENV_PATH")
[[ $mode == 600 ]] || { echo "Existing env file has mode $mode, expected 600" >&2; exit 1; }
actual_env_sha=$(sha256sum "$ENV_PATH" | awk '{print tolower($1)}')
[[ $actual_env_sha == ${EXPECTED_ENV_SHA,,} ]] || {
  echo 'Reviewed env SHA-256 mismatch' >&2
  exit 1
}
for state_dir in "$DEPLOY_DIR/data" "$DEPLOY_DIR/logs"; do
  if [[ -e $state_dir || -L $state_dir ]]; then
    [[ -d $state_dir && ! -L $state_dir ]] || {
      echo "Deployment state path must be a real directory: $state_dir" >&2
      exit 1
    }
  else
    mkdir "$state_dir"
  fi
done
chmod 750 "$DEPLOY_DIR" "$DEPLOY_DIR/data" "$DEPLOY_DIR/logs"

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

# Read the reviewed values so retries use exactly the same credentials and names.
COMPOSE_PROJECT_NAME=$(require_value COMPOSE_PROJECT_NAME)
NETWORK_NAME=$(require_value NETWORK_NAME)
POSTGRES_VOLUME=$(require_value POSTGRES_VOLUME)
REDIS_VOLUME=$(require_value REDIS_VOLUME)
APP_PORT=$(require_value APP_PORT)
METRICS_HOST_PORT=$(require_value METRICS_HOST_PORT)
METRICS_PORT=$(require_value METRICS_PORT)
IMAGE_TAG=$(require_value IMAGE_TAG)
CONTAINER=$(require_value CONTAINER)
POSTGRES_IMAGE=$(require_value POSTGRES_IMAGE)
POSTGRES_CONTAINER=$(require_value POSTGRES_CONTAINER)
POSTGRES_ADMIN_USER=$(require_value POSTGRES_ADMIN_USER)
POSTGRES_ADMIN_PASSWORD=$(require_value POSTGRES_ADMIN_PASSWORD)
POSTGRES_ADMIN_DB=$(require_value POSTGRES_ADMIN_DB)
POSTGRES_DB=$(require_value POSTGRES_DB)
POSTGRES_APP_USER=$(require_value POSTGRES_APP_USER)
POSTGRES_APP_PASSWORD=$(require_value POSTGRES_APP_PASSWORD)
REDIS_IMAGE=$(require_value REDIS_IMAGE)
REDIS_CONTAINER=$(require_value REDIS_CONTAINER)
REDIS_PASSWORD=$(require_value REDIS_PASSWORD)
REDIS_DB=$(require_value REDIS_DB)
NODE_TYPE=$(require_value NODE_TYPE)

validate_secret POSTGRES_ADMIN_PASSWORD "$POSTGRES_ADMIN_PASSWORD"
validate_secret POSTGRES_APP_PASSWORD "$POSTGRES_APP_PASSWORD"
validate_secret REDIS_PASSWORD "$REDIS_PASSWORD"
validate_secret SESSION_SECRET "$(require_value SESSION_SECRET)"
validate_secret CRYPTO_SECRET "$(require_value CRYPTO_SECRET)"

require_exact() {
  local key=$1
  local expected=$2
  local actual
  actual=$(require_value "$key")
  [[ $actual == "$expected" ]] || {
    echo "$key must be $expected for the isolated ikun.love stack" >&2
    exit 1
  }
}

require_exact COMPOSE_PROJECT_NAME ikun-new-api
require_exact NETWORK_NAME ikun-new-api-network
require_exact POSTGRES_VOLUME ikun-new-api-postgres-data
require_exact REDIS_VOLUME ikun-new-api-redis-data
require_exact APP_PORT 3000
require_exact METRICS_HOST_PORT 8006
require_exact METRICS_PORT 8006
require_exact IMAGE_TAG new-api:ikun
require_exact CONTAINER ikun-new-api
require_exact POSTGRES_IMAGE postgres:18.4-alpine
require_exact POSTGRES_CONTAINER ikun-new-api-postgres
require_exact POSTGRES_ADMIN_USER ikun_pg_admin
require_exact POSTGRES_ADMIN_DB postgres
require_exact POSTGRES_DB new_api
require_exact POSTGRES_APP_USER new_api_app
require_exact REDIS_IMAGE redis:8.8.0
require_exact REDIS_CONTAINER ikun-new-api-redis
require_exact REDIS_DB 0
require_exact NODE_TYPE master

[[ $POSTGRES_ADMIN_USER =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid persisted PostgreSQL administrator' >&2; exit 1; }
[[ $POSTGRES_APP_USER =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid persisted PostgreSQL application role' >&2; exit 1; }
[[ $POSTGRES_APP_USER != "$POSTGRES_ADMIN_USER" ]] || { echo 'Application and administrator roles must differ' >&2; exit 1; }
[[ $POSTGRES_DB =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid persisted database name' >&2; exit 1; }
[[ $POSTGRES_ADMIN_DB =~ ^[a-z_][a-z0-9_]*$ ]] || { echo 'Invalid persisted administrator database name' >&2; exit 1; }
[[ $POSTGRES_ADMIN_DB != "$POSTGRES_DB" ]] || { echo 'Administrator and application databases must differ' >&2; exit 1; }
[[ $POSTGRES_ADMIN_DB != template0 && $POSTGRES_ADMIN_DB != template1 && $POSTGRES_DB != template0 && $POSTGRES_DB != template1 ]] || {
  echo 'Template databases cannot be used for the application stack' >&2
  exit 1
}
[[ $APP_PORT =~ ^[0-9]+$ && $APP_PORT -ge 1 && $APP_PORT -le 65535 ]] || { echo 'Invalid persisted app port' >&2; exit 1; }
[[ $METRICS_HOST_PORT =~ ^[0-9]+$ && $METRICS_HOST_PORT -ge 1 && $METRICS_HOST_PORT -le 65535 ]] || { echo 'Invalid persisted metrics host port' >&2; exit 1; }
[[ $METRICS_PORT =~ ^[0-9]+$ && $METRICS_PORT -ge 1 && $METRICS_PORT -le 65535 ]] || { echo 'Invalid persisted metrics container port' >&2; exit 1; }
[[ $APP_PORT != "$METRICS_HOST_PORT" ]] || { echo 'App and metrics host ports must differ' >&2; exit 1; }
[[ $POSTGRES_CONTAINER =~ ^[A-Za-z0-9_.-]+$ && $REDIS_CONTAINER =~ ^[A-Za-z0-9_.-]+$ ]] || {
  echo 'Invalid persisted container name' >&2
  exit 1
}
[[ $POSTGRES_CONTAINER == ikun-new-api-postgres && $REDIS_CONTAINER == ikun-new-api-redis ]] || {
  echo 'Refusing non-dedicated data-service container names' >&2
  exit 1
}

compose() {
  docker compose --project-name "$COMPOSE_PROJECT_NAME" --env-file "$ENV_PATH" -f "$COMPOSE_PATH" "$@"
}

assert_container_contract() {
  local container=$1
  local service=$2
  local expected_image=$3
  local project_label service_label image networks network_count
  if ! docker inspect "$container" >/dev/null 2>&1; then
    return 0
  fi
  project_label=$(docker inspect "$container" --format '{{index .Config.Labels "com.docker.compose.project"}}')
  service_label=$(docker inspect "$container" --format '{{index .Config.Labels "com.docker.compose.service"}}')
  image=$(docker inspect "$container" --format '{{.Config.Image}}')
  [[ $project_label == "$COMPOSE_PROJECT_NAME" && $service_label == "$service" ]] || {
    echo "Docker container collision: $container is not owned by Compose project $COMPOSE_PROJECT_NAME" >&2
    exit 1
  }
  [[ $image == "$expected_image" ]] || {
    echo "Docker container $container has unexpected image: $image" >&2
    exit 1
  }
  networks=$(docker inspect "$container" --format '{{range $name, $cfg := .NetworkSettings.Networks}}{{println $name}}{{end}}')
  network_count=$(printf '%s\n' "$networks" | awk 'NF {count++} END {print count + 0}')
  [[ $network_count -eq 1 ]] && grep -Fxq "$NETWORK_NAME" <<< "$networks" || {
    echo "Docker container $container is attached to an unexpected network" >&2
    exit 1
  }
}

assert_volume_contract() {
  local volume=$1
  local compose_volume=$2
  local project_label volume_label
  if ! docker volume inspect "$volume" >/dev/null 2>&1; then
    return 0
  fi
  project_label=$(docker volume inspect "$volume" --format '{{index .Labels "com.docker.compose.project"}}')
  volume_label=$(docker volume inspect "$volume" --format '{{index .Labels "com.docker.compose.volume"}}')
  [[ $project_label == "$COMPOSE_PROJECT_NAME" && $volume_label == "$compose_volume" ]] || {
    echo "Docker volume collision: $volume is not owned by this isolated stack" >&2
    exit 1
  }
}

assert_network_contract() {
  local project_label network_label driver internal members member
  if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
    return 0
  fi
  project_label=$(docker network inspect "$NETWORK_NAME" --format '{{index .Labels "com.docker.compose.project"}}')
  network_label=$(docker network inspect "$NETWORK_NAME" --format '{{index .Labels "com.docker.compose.network"}}')
  driver=$(docker network inspect "$NETWORK_NAME" --format '{{.Driver}}')
  internal=$(docker network inspect "$NETWORK_NAME" --format '{{.Internal}}')
  [[ $project_label == "$COMPOSE_PROJECT_NAME" && $network_label == ikun-new-api-network && $driver == bridge && $internal == false ]] || {
    echo "Docker network collision: $NETWORK_NAME is not this isolated bridge" >&2
    exit 1
  }
  members=$(docker network inspect "$NETWORK_NAME" --format '{{range $id, $container := .Containers}}{{println $container.Name}}{{end}}')
  while IFS= read -r member; do
    [[ -z $member ]] && continue
    [[ $member == "$CONTAINER" || $member == "$POSTGRES_CONTAINER" || $member == "$REDIS_CONTAINER" ]] || {
      echo "Docker network $NETWORK_NAME contains an unexpected container: $member" >&2
      exit 1
    }
  done <<< "$members"
}

host_port_bound() {
  local port=$1
  ss -ltnH | awk -v wanted=":$port" '$4 ~ (wanted "$") {found=1} END {exit !found}'
}

assert_port_contract() {
  local host_port=$1
  local container=$2
  local container_port=$3
  local binding
  if ! host_port_bound "$host_port"; then
    return 0
  fi
  docker inspect "$container" >/dev/null 2>&1 || {
    echo "Host port collision: port $host_port is already bound" >&2
    exit 1
  }
  binding=$(docker port "$container" "$container_port/tcp" 2>/dev/null || true)
  grep -Fq "127.0.0.1:$host_port" <<< "$binding" || {
    echo "Host port collision: port $host_port is not owned by $container" >&2
    exit 1
  }
}

assert_container_contract "$CONTAINER" new-api "$IMAGE_TAG"
assert_container_contract "$POSTGRES_CONTAINER" postgres "$POSTGRES_IMAGE"
assert_container_contract "$REDIS_CONTAINER" redis "$REDIS_IMAGE"
assert_volume_contract "$POSTGRES_VOLUME" postgres-data
assert_volume_contract "$REDIS_VOLUME" redis-data
assert_network_contract
assert_port_contract "$APP_PORT" "$CONTAINER" 3000
assert_port_contract "$METRICS_HOST_PORT" "$CONTAINER" "$METRICS_PORT"

compose_services=$(compose config --services)
for required_service in new-api postgres redis; do
  grep -Fxq "$required_service" <<< "$compose_services" || {
    echo "Compose file is missing required service: $required_service" >&2
    exit 1
  }
done
[[ $(printf '%s\n' "$compose_services" | awk 'NF {count++} END {print count + 0}') -eq 3 ]] || {
  echo 'Compose file contains unexpected services' >&2
  exit 1
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

# A retry may reuse this stack's own database, but an unexpected user database
# means the volume is not a fresh, dedicated PostgreSQL instance. Fail before
# changing roles or grants so an unrelated workload cannot be modified.
unexpected_databases=$(pg_admin "$POSTGRES_ADMIN_DB" -Atqc \
  "SELECT datname FROM pg_database WHERE datistemplate = false AND datname NOT IN ('$POSTGRES_ADMIN_DB', '$POSTGRES_DB')")
[[ -z $unexpected_databases ]] || {
  echo 'PostgreSQL volume contains an unexpected database; refusing first-install reuse' >&2
  exit 1
}

role_flags=$(pg_admin "$POSTGRES_ADMIN_DB" -Atqc \
  "SELECT rolsuper || '|' || rolcreatedb || '|' || rolcreaterole || '|' || rolreplication || '|' || rolbypassrls || '|' || rolinherit || '|' || rolcanlogin FROM pg_roles WHERE rolname = '$POSTGRES_APP_USER'")
if [[ -z $role_flags ]]; then
  pg_admin "$POSTGRES_ADMIN_DB" -c \
    "CREATE ROLE \"$POSTGRES_APP_USER\" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS NOINHERIT PASSWORD '$POSTGRES_APP_PASSWORD'" >/dev/null
else
  [[ $role_flags == 'f|f|f|f|f|f|t' || $role_flags == 'false|false|false|false|false|false|true' ]] || {
    echo "Existing application role has unsafe flags: $role_flags" >&2
    exit 1
  }
  pg_admin "$POSTGRES_ADMIN_DB" -c \
    "ALTER ROLE \"$POSTGRES_APP_USER\" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS NOINHERIT PASSWORD '$POSTGRES_APP_PASSWORD'" >/dev/null
fi

membership_count=$(pg_admin "$POSTGRES_ADMIN_DB" -Atqc \
  "SELECT count(*) FROM pg_auth_members WHERE member = (SELECT oid FROM pg_roles WHERE rolname = '$POSTGRES_APP_USER')")
[[ $membership_count == 0 ]] || {
  echo 'Application role must not be a member of another PostgreSQL role' >&2
  exit 1
}

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

# PostgreSQL grants CONNECT to PUBLIC by default. Remove that inherited access
# from the bootstrap/template databases so the application role can connect
# only to its dedicated database; keep the administrator's access explicit.
pg_admin "$POSTGRES_ADMIN_DB" -c \
  "REVOKE CONNECT ON DATABASE \"$POSTGRES_ADMIN_DB\" FROM PUBLIC, \"$POSTGRES_APP_USER\"; GRANT CONNECT ON DATABASE \"$POSTGRES_ADMIN_DB\" TO \"$POSTGRES_ADMIN_USER\"; REVOKE CONNECT ON DATABASE template0 FROM PUBLIC, \"$POSTGRES_APP_USER\"; REVOKE CONNECT ON DATABASE template1 FROM PUBLIC, \"$POSTGRES_APP_USER\"; GRANT CONNECT ON DATABASE template1 TO \"$POSTGRES_ADMIN_USER\"" >/dev/null

app_probe=$(docker exec -e "PGPASSWORD=$POSTGRES_APP_PASSWORD" "$POSTGRES_CONTAINER" \
  psql -U "$POSTGRES_APP_USER" -d "$POSTGRES_DB" -Atqc \
  "SELECT current_database() || '|' || current_user")
[[ $app_probe == "$POSTGRES_DB|$POSTGRES_APP_USER" ]] || {
  echo 'Application PostgreSQL probe did not use the isolated role/database' >&2
  exit 1
}

if docker exec -e "PGPASSWORD=$POSTGRES_APP_PASSWORD" "$POSTGRES_CONTAINER" \
  psql -U "$POSTGRES_APP_USER" -d "$POSTGRES_ADMIN_DB" -v ON_ERROR_STOP=1 -Atqc 'SELECT 1' >/dev/null 2>&1; then
  echo 'Application role can connect to the bootstrap database' >&2
  exit 1
fi
if docker exec -e "PGPASSWORD=$POSTGRES_APP_PASSWORD" "$POSTGRES_CONTAINER" \
  psql -U "$POSTGRES_APP_USER" -d template1 -v ON_ERROR_STOP=1 -Atqc 'SELECT 1' >/dev/null 2>&1; then
  echo 'Application role can connect to template1' >&2
  exit 1
fi
echo "database_access=$POSTGRES_APP_USER:$POSTGRES_DB-only"

redis_ping=$(docker exec -e "REDISCLI_AUTH=$REDIS_PASSWORD" "$REDIS_CONTAINER" \
  redis-cli --no-auth-warning -n "$REDIS_DB" PING)
[[ $redis_ping == PONG ]] || { echo 'Dedicated Redis readiness check failed' >&2; exit 1; }

printf 'database=%s\n' "$POSTGRES_DB"
printf 'role=%s superuser=false createdb=false createrole=false replication=false bypassrls=false inherit=false memberships=0\n' "$POSTGRES_APP_USER"
printf 'redis_database=%s\n' "$REDIS_DB"
printf 'env_mode='; stat -c '%a' "$ENV_PATH"
printf 'env_sha256=%s\n' "$actual_env_sha"
echo 'status=bootstrapped'
