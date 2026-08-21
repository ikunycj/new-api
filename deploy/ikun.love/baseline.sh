#!/usr/bin/env bash
set -Eeuo pipefail

DEPLOY_DIR=${DEPLOY_DIR:-/opt/new-api}
APP_CONTAINER=${CONTAINER:-ikun-new-api}
IMAGE_TAG=${IMAGE_TAG:-new-api:ikun}
POSTGRES_CONTAINER=${POSTGRES_CONTAINER:-ikun-new-api-postgres}
REDIS_CONTAINER=${REDIS_CONTAINER:-ikun-new-api-redis}
NETWORK_NAME=${NETWORK_NAME:-ikun-new-api-network}

if docker info >/dev/null 2>&1; then
  DOCKER=(docker)
elif sudo -n docker info >/dev/null 2>&1; then
  DOCKER=(sudo -n docker)
else
  echo 'docker_access=unavailable'
  exit 1
fi

printf 'deploy_dir=%s\n' "$DEPLOY_DIR"
if [[ -d $DEPLOY_DIR ]]; then
  cd "$DEPLOY_DIR"
  printf 'env_mode='; stat -c '%a' .env 2>/dev/null || echo missing
  printf 'commit='; cat .deploy-commit 2>/dev/null || true
  printf 'release='; cat .deploy-release 2>/dev/null || true
else
  echo 'deploy_state=missing'
fi

inspect_container() {
  local label=$1
  local container=$2
  if "${DOCKER[@]}" container inspect "$container" >/dev/null 2>&1; then
    "${DOCKER[@]}" container inspect "$container" --format "$label={{.Name}} state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restarts={{.RestartCount}} image={{.Config.Image}}"
  else
    echo "$label=missing"
  fi
}

inspect_container app "$APP_CONTAINER"
inspect_container postgres "$POSTGRES_CONTAINER"
inspect_container redis "$REDIS_CONTAINER"
if "${DOCKER[@]}" image inspect "$IMAGE_TAG" >/dev/null 2>&1; then
  "${DOCKER[@]}" image inspect "$IMAGE_TAG" --format 'image={{.Id}} revision={{index .Config.Labels "org.opencontainers.image.revision"}} release={{index .Config.Labels "com.new-api.release"}}'
else
  echo 'image=missing'
fi

if "${DOCKER[@]}" network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  network_containers=$("${DOCKER[@]}" network inspect "$NETWORK_NAME" --format '{{range $id, $container := .Containers}}{{printf "%s " $container.Name}}{{end}}')
  printf 'network=%s members=%s\n' "$NETWORK_NAME" "${network_containers:-none}"
else
  echo "network=$NETWORK_NAME missing"
fi

# This is a fresh host. The old Sub2API service is on the separate
# `ikun.love-sub2api` SSH target and is checked there before/after deployment.
# Keep the new host free of the legacy port and report the current public
# endpoint as an informational DNS/routing observation.
printf 'legacy_sub2api_host=ikun.love-sub2api\n'
printf 'new_host_port8080='; ss -lnt 2>/dev/null | awk '$4 ~ /:8080$/ {found=1} END {print found ? "bound" : "unbound"}'
printf 'public_origin_status='; curl -sS -o /dev/null -w '%{http_code}\n' --max-time 15 https://ikun.love/api/status 2>/dev/null || echo 000
printf 'new_api_local_status='; curl -sS -o /dev/null -w '%{http_code}\n' --max-time 10 http://127.0.0.1:3000/api/status 2>/dev/null || echo 000
