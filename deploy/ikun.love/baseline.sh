#!/usr/bin/env bash
set -Eeuo pipefail

DEPLOY_DIR=${DEPLOY_DIR:-/opt/new-api}
APP_CONTAINER=${CONTAINER:-ikun-new-api}
IMAGE_TAG=${IMAGE_TAG:-new-api:ikun}
POSTGRES_CONTAINER=${POSTGRES_CONTAINER:-1Panel-postgresql-8Kr6}
REDIS_CONTAINER=${REDIS_CONTAINER:-1Panel-redis-xsdn}

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

if "${DOCKER[@]}" container inspect "$APP_CONTAINER" >/dev/null 2>&1; then
  "${DOCKER[@]}" inspect "$APP_CONTAINER" --format 'app_state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restarts={{.RestartCount}} image={{.Config.Image}}'
else
  echo 'app_state=missing'
fi
if "${DOCKER[@]}" image inspect "$IMAGE_TAG" >/dev/null 2>&1; then
  "${DOCKER[@]}" image inspect "$IMAGE_TAG" --format 'image={{.Id}} revision={{index .Config.Labels "org.opencontainers.image.revision"}} release={{index .Config.Labels "com.new-api.release"}}'
else
  echo 'image_state=missing'
fi
if "${DOCKER[@]}" inspect "$POSTGRES_CONTAINER" >/dev/null 2>&1; then
  "${DOCKER[@]}" inspect "$POSTGRES_CONTAINER" --format 'postgres_state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
else
  echo 'postgres_state=missing'
fi
if "${DOCKER[@]}" inspect "$REDIS_CONTAINER" >/dev/null 2>&1; then
  "${DOCKER[@]}" inspect "$REDIS_CONTAINER" --format 'redis_state={{.State.Status}}'
else
  echo 'redis_state=missing'
fi

printf 'new_api_local_status='; curl -sS -o /dev/null -w '%{http_code}\n' --max-time 10 http://127.0.0.1:3000/api/status 2>/dev/null || echo 000
printf 'sub2api_service='; systemctl is-active sub2api.service 2>/dev/null || true
printf 'sub2api_local_status='; curl -sS -o /dev/null -w '%{http_code}\n' --max-time 10 http://127.0.0.1:8080/api/v1/settings/public 2>/dev/null || echo 000
printf 'sub2api_public_status='; curl -sS -o /dev/null -w '%{http_code}\n' --max-time 15 https://ikun.love/api/v1/settings/public 2>/dev/null || echo 000
