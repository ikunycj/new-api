#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: deploy-binary.sh \
  --archive PATH --archive-sha SHA256 --binary-sha SHA256 \
  --commit FULL_SHA --release NAME [options]

Options:
  --deploy-dir PATH       Deployment directory (default: /opt/new-api)
  --compose-file PATH     Compose file relative to deploy dir
                          (default: docker-compose.1panel.yml)
  --env-file PATH         Env file relative to deploy dir (default: .env)
  --image-tag TAG         Live image tag (default: new-api:local)
  --project-name NAME     Compose project name (optional)
  --network NAME          Expected app network (optional)
  --service NAME          Compose service (default: new-api)
  --container NAME        App container (default: new-api)
  --postgres NAME         Existing PostgreSQL container
  --redis NAME            Existing Redis container
  --local-url URL         Local health URL
  --public-url URL        Optional public status URL
  --timeout SECONDS       Local health timeout (default: 120)
  -h, --help              Show this help
EOF
}

ARCHIVE=''
ARCHIVE_SHA=''
BINARY_SHA=''
COMMIT=''
RELEASE=''
DEPLOY_DIR='/opt/new-api'
COMPOSE_FILE='docker-compose.1panel.yml'
ENV_FILE='.env'
IMAGE_TAG='new-api:local'
PROJECT_NAME=''
NETWORK=''
SERVICE='new-api'
CONTAINER='new-api'
POSTGRES='1Panel-postgresql-2LOJ'
REDIS='1Panel-redis-pDR8'
LOCAL_URL='http://127.0.0.1:3000/api/status'
PUBLIC_URL=''
TIMEOUT=120

while (($#)); do
  case "$1" in
    --archive) ARCHIVE=${2:?missing value}; shift 2 ;;
    --archive-sha) ARCHIVE_SHA=${2:?missing value}; shift 2 ;;
    --binary-sha) BINARY_SHA=${2:?missing value}; shift 2 ;;
    --commit) COMMIT=${2:?missing value}; shift 2 ;;
    --release) RELEASE=${2:?missing value}; shift 2 ;;
    --deploy-dir) DEPLOY_DIR=${2:?missing value}; shift 2 ;;
    --compose-file) COMPOSE_FILE=${2:?missing value}; shift 2 ;;
    --env-file) ENV_FILE=${2:?missing value}; shift 2 ;;
    --image-tag) IMAGE_TAG=${2:?missing value}; shift 2 ;;
    --project-name) PROJECT_NAME=${2:?missing value}; shift 2 ;;
    --network) NETWORK=${2:?missing value}; shift 2 ;;
    --service) SERVICE=${2:?missing value}; shift 2 ;;
    --container) CONTAINER=${2:?missing value}; shift 2 ;;
    --postgres) POSTGRES=${2:?missing value}; shift 2 ;;
    --redis) REDIS=${2:?missing value}; shift 2 ;;
    --local-url) LOCAL_URL=${2:?missing value}; shift 2 ;;
    --public-url) PUBLIC_URL=${2:?missing value}; shift 2 ;;
    --timeout) TIMEOUT=${2:?missing value}; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

for value_name in ARCHIVE ARCHIVE_SHA BINARY_SHA COMMIT RELEASE; do
  if [[ -z ${!value_name} ]]; then
    echo "Missing required argument: $value_name" >&2
    usage >&2
    exit 2
  fi
done

[[ $ARCHIVE_SHA =~ ^[0-9a-fA-F]{64}$ ]] || { echo 'Invalid archive SHA-256' >&2; exit 2; }
[[ $BINARY_SHA =~ ^[0-9a-fA-F]{64}$ ]] || { echo 'Invalid binary SHA-256' >&2; exit 2; }
[[ $COMMIT =~ ^[0-9a-fA-F]{40}$ ]] || { echo 'Commit must be a full 40-character SHA' >&2; exit 2; }
[[ $RELEASE =~ ^[A-Za-z0-9._-]+$ ]] || { echo 'Release contains unsafe characters' >&2; exit 2; }
[[ $TIMEOUT =~ ^[0-9]+$ ]] && ((TIMEOUT >= 10 && TIMEOUT <= 900)) || { echo 'Timeout must be 10..900 seconds' >&2; exit 2; }
[[ -z $PROJECT_NAME || $PROJECT_NAME =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]] || { echo 'Project name contains unsafe characters' >&2; exit 2; }
[[ -z $NETWORK || $NETWORK =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]] || { echo 'Network name contains unsafe characters' >&2; exit 2; }

for command_name in docker curl sha256sum stat tar; do
  command -v "$command_name" >/dev/null || { echo "Missing command: $command_name" >&2; exit 1; }
done
docker compose version >/dev/null

cd "$DEPLOY_DIR"
[[ -f $COMPOSE_FILE ]] || { echo "Missing compose file: $COMPOSE_FILE" >&2; exit 1; }
[[ -f $ENV_FILE ]] || { echo "Missing env file: $ENV_FILE" >&2; exit 1; }
[[ -f $ARCHIVE ]] || { echo "Missing archive: $ARCHIVE" >&2; exit 1; }
env_mode=$(stat -c '%a' "$ENV_FILE")
[[ $env_mode == 600 ]] || { echo "Refusing to deploy: $ENV_FILE mode is $env_mode, expected 600" >&2; exit 1; }

compose_args=(--env-file "$ENV_FILE" -f "$COMPOSE_FILE")
if [[ -n $PROJECT_NAME ]]; then
  compose_args+=(--project-name "$PROJECT_NAME")
fi
docker compose "${compose_args[@]}" config --quiet
compose_services=$(docker compose "${compose_args[@]}" config --services)
grep -Fxq "$SERVICE" <<<"$compose_services" || {
  echo "Compose service is not defined: $SERVICE" >&2
  exit 1
}
compose_images=$(docker compose "${compose_args[@]}" config --images)
grep -Fxq "$IMAGE_TAG" <<<"$compose_images" || {
  echo "Compose does not resolve the expected app image: $IMAGE_TAG" >&2
  exit 1
}
if [[ -n $PROJECT_NAME ]]; then
  current_project=$(docker inspect "$CONTAINER" --format '{{index .Config.Labels "com.docker.compose.project"}}' 2>/dev/null || true)
  [[ $current_project == "$PROJECT_NAME" ]] || {
    echo "Existing app container belongs to project ${current_project:-missing}, expected $PROJECT_NAME" >&2
    exit 1
  }
fi
current_image=$(docker inspect "$CONTAINER" --format '{{.Config.Image}}' 2>/dev/null || true)
[[ $current_image == "$IMAGE_TAG" ]] || {
  echo "Existing app container uses ${current_image:-missing}, expected $IMAGE_TAG" >&2
  exit 1
}
if [[ -n $NETWORK ]]; then
  docker network inspect "$NETWORK" >/dev/null || { echo "Missing expected network: $NETWORK" >&2; exit 1; }
  current_networks=$(docker inspect "$CONTAINER" --format '{{json .NetworkSettings.Networks}}' 2>/dev/null || true)
  grep -Fq "\"$NETWORK\"" <<<"$current_networks" || {
    echo "Existing app container is not attached to expected network: $NETWORK" >&2
    exit 1
  }
fi

actual_archive_sha=$(sha256sum "$ARCHIVE" | awk '{print tolower($1)}')
[[ $actual_archive_sha == ${ARCHIVE_SHA,,} ]] || { echo 'Archive SHA-256 mismatch' >&2; exit 1; }

postgres_state=$(docker inspect "$POSTGRES" --format '{{.State.Status}}')
postgres_health=$(docker inspect "$POSTGRES" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}')
[[ $postgres_state == running && $postgres_health == healthy ]] || {
  echo "PostgreSQL is not ready: state=$postgres_state health=$postgres_health" >&2
  exit 1
}
redis_state=$(docker inspect "$REDIS" --format '{{.State.Status}}')
[[ $redis_state == running ]] || { echo "Redis is not running: $redis_state" >&2; exit 1; }

TMP_DIR="$DEPLOY_DIR/.deploy-tmp-$RELEASE-$$"
TEMP_CONTAINER="new-api-wrap-$RELEASE-$$"
ROLLBACK_TAG="new-api:rollback-$RELEASE"
CANDIDATE_TAG="new-api:candidate-$RELEASE"

cleanup() {
  docker rm -f "$TEMP_CONTAINER" >/dev/null 2>&1 || true
  case "$TMP_DIR" in
    "$DEPLOY_DIR"/.deploy-tmp-*) rm -rf -- "$TMP_DIR" ;;
    *) echo "Refusing to clean unexpected path: $TMP_DIR" >&2 ;;
  esac
}
trap cleanup EXIT

mkdir -m 700 "$TMP_DIR"
tar -xzf "$ARCHIVE" -C "$TMP_DIR" new-api
NEW_BINARY="$TMP_DIR/new-api"
[[ -f $NEW_BINARY && ! -L $NEW_BINARY ]] || { echo 'Archive did not contain a regular new-api binary' >&2; exit 1; }
actual_binary_sha=$(sha256sum "$NEW_BINARY" | awk '{print tolower($1)}')
[[ $actual_binary_sha == ${BINARY_SHA,,} ]] || { echo 'Binary SHA-256 mismatch' >&2; exit 1; }
chmod 0755 "$NEW_BINARY"
"$NEW_BINARY" --help >/dev/null

docker image inspect "$IMAGE_TAG" >/dev/null
previous_image_id=$(docker image inspect "$IMAGE_TAG" --format '{{.Id}}')
docker tag "$IMAGE_TAG" "$ROLLBACK_TAG"

docker create --name "$TEMP_CONTAINER" "$IMAGE_TAG" >/dev/null
docker cp "$NEW_BINARY" "$TEMP_CONTAINER:/new-api"
docker commit \
  --change "LABEL org.opencontainers.image.revision=$COMMIT" \
  --change "LABEL com.new-api.release=$RELEASE" \
  "$TEMP_CONTAINER" "$CANDIDATE_TAG" >/dev/null
docker rm "$TEMP_CONTAINER" >/dev/null
docker run --rm --entrypoint /new-api "$CANDIDATE_TAG" --help >/dev/null

compose_recreate() {
  docker compose "${compose_args[@]}" \
    up -d --no-build --no-deps --force-recreate "$SERVICE"
}

assert_runtime_topology() {
  local actual_name actual_image actual_networks
  actual_name=$(docker inspect "$CONTAINER" --format '{{.Name}}' | sed 's#^/##')
  actual_image=$(docker inspect "$CONTAINER" --format '{{.Config.Image}}')
  [[ $actual_name == "$CONTAINER" ]] || { echo "Unexpected app container: $actual_name" >&2; return 1; }
  [[ $actual_image == "$IMAGE_TAG" ]] || { echo "Unexpected app image: $actual_image" >&2; return 1; }
  if [[ -n $PROJECT_NAME ]]; then
    [[ $(docker inspect "$CONTAINER" --format '{{index .Config.Labels "com.docker.compose.project"}}') == "$PROJECT_NAME" ]] || {
      echo "Unexpected Compose project for $CONTAINER" >&2
      return 1
    }
  fi
  if [[ -n $NETWORK ]]; then
    actual_networks=$(docker inspect "$CONTAINER" --format '{{json .NetworkSettings.Networks}}')
    grep -Fq "\"$NETWORK\"" <<<"$actual_networks" || { echo "Unexpected app network" >&2; return 1; }
  fi
}

wait_for_local_health() {
  local deadline=$((SECONDS + TIMEOUT))
  local state health
  while ((SECONDS < deadline)); do
    state=$(docker inspect "$CONTAINER" --format '{{.State.Status}}' 2>/dev/null || true)
    health=$(docker inspect "$CONTAINER" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' 2>/dev/null || true)
    if [[ $state == running && $health == healthy ]] && curl -fsS "$LOCAL_URL" >/dev/null; then
      return 0
    fi
    sleep 3
  done
  echo "Health timeout: state=${state:-missing} health=${health:-missing}" >&2
  return 1
}

rollback() {
  echo "Rolling back to $previous_image_id" >&2
  docker tag "$ROLLBACK_TAG" "$IMAGE_TAG"
  compose_recreate
  wait_for_local_health
}

docker tag "$CANDIDATE_TAG" "$IMAGE_TAG"
if ! compose_recreate; then
  rollback
  exit 1
fi
if ! assert_runtime_topology; then
  rollback
  exit 1
fi
if ! wait_for_local_health; then
  docker logs --tail 200 "$CONTAINER" >&2 || true
  rollback
  exit 1
fi

runtime_tmp="$TMP_DIR/new-api.runtime"
docker cp "$CONTAINER:/new-api" "$runtime_tmp"
runtime_sha=$(sha256sum "$runtime_tmp" | awk '{print tolower($1)}')
[[ $runtime_sha == ${BINARY_SHA,,} ]] || {
  echo "Runtime binary mismatch after switch: $runtime_sha" >&2
  rollback
  exit 1
}

if [[ -n $PUBLIC_URL ]]; then
  curl -fsS "$PUBLIC_URL" >/dev/null || {
    echo "Local deployment is healthy, but public check failed: $PUBLIC_URL" >&2
    exit 1
  }
fi

printf '%s\n' "$COMMIT" > .deploy-commit
printf '%s\n' "$RELEASE" > .deploy-release

new_image_id=$(docker image inspect "$IMAGE_TAG" --format '{{.Id}}')
echo "release=$RELEASE"
echo "commit=$COMMIT"
echo "image=$new_image_id"
echo "rollback=$ROLLBACK_TAG"
echo "binary_sha256=$runtime_sha"
