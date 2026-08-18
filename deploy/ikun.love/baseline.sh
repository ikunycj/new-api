#!/usr/bin/env bash
set -Eeuo pipefail

DEPLOY_DIR=${DEPLOY_DIR:-/opt/new-api}
APP_CONTAINER=${CONTAINER:-new-api}
POSTGRES_CONTAINER=${POSTGRES_CONTAINER:?Set POSTGRES_CONTAINER from the read-only target audit}
REDIS_CONTAINER=${REDIS_CONTAINER:?Set REDIS_CONTAINER from the read-only target audit}

cd "$DEPLOY_DIR"

printf 'deploy_dir=%s\n' "$DEPLOY_DIR"
printf 'env_mode='; stat -c '%a' .env
printf 'commit='; cat .deploy-commit 2>/dev/null || true
printf 'release='; cat .deploy-release 2>/dev/null || true

docker ps --filter "name=^/${APP_CONTAINER}$" --format 'container={{.Names}} status={{.Status}} image={{.Image}}'
docker inspect "$APP_CONTAINER" --format 'app_state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restarts={{.RestartCount}}'
docker inspect "$POSTGRES_CONTAINER" --format 'postgres_state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
docker inspect "$REDIS_CONTAINER" --format 'redis_state={{.State.Status}}'
docker network inspect 1panel-network --format 'network={{.Name}} driver={{.Driver}} scope={{.Scope}}'
docker image inspect new-api:local --format 'image={{.Id}} revision={{index .Config.Labels "org.opencontainers.image.revision"}} release={{index .Config.Labels "com.new-api.release"}}'

if curl -fsS http://127.0.0.1:3000/api/status >/dev/null; then
  echo 'local_status=ok'
else
  echo 'local_status=failed'
fi
