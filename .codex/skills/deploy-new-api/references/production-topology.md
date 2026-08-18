# Deployment topology

This is a last-known snapshot from 2026-07-21. Verify every mutable value on the host before using it.

## Deployment host selection

Ask before every deployment whether the target is `aliyun` (development), `alltokenapi` (production), or `ikun.love` (production). The host and public URL must come from the current conversation and a live read-only check.

| Environment | SSH alias | Host | Deployment scope |
| --- | --- | --- | --- |
| Development/test | `aliyun` | Verify from local SSH config | The remote test service and local development process both use `NODE_TYPE=master` and share the Aliyun test PostgreSQL/Redis; keep those services isolated from production. |
| Production | `alltokenapi` | `154.37.213.1` (verify live) | Production release and rollback workflow below. |
| Production | `ikun.love` | Verify from local SSH config | Dedicated `ikun.love` release ref and deployment materials; do not reuse the AllToken snapshot. |

## Service map

| Role | Last-known value | Required behavior |
| --- | --- | --- |
| SSH alias | `alltokenapi` | Use the selected alias from the host-selection gate; do not assume this value for development deployments. |
| Public site | `https://alltokenapi.com` | Verify status and the changed user route after local health passes. |
| Deploy directory | `/opt/new-api` | Preserve `.env`, `data`, and `logs`. |
| App container/service | `new-api` | Recreate only this service. |
| App image tag | `new-api:local` | Retag the current image before switching. |
| App port | `127.0.0.1:3000` | Check `/api/status` locally. |
| Compose file | `docker-compose.1panel.yml` | It should define only the app and reuse external services. |
| Runtime env | `.env`, mode `600` | Never display or replace secret values. |
| Docker network | `1panel-network` | Reuse it; do not create a second database network. |
| PostgreSQL | `1Panel-postgresql-2LOJ` | Must be healthy before switching the app. |
| Redis | `1Panel-redis-pDR8` | Must be running before switching the app. |

## ikun.love target contract

The following values define the target-specific release contract, not a live
server snapshot. Verify every mutable value during the read-only baseline:

| Role | Target-specific value | Required behavior |
| --- | --- | --- |
| SSH alias | `ikun.love` | Use this alias for every remote command. |
| Release ref | `origin/ikun.love` | Verify local, fetched, and live remote SHAs match. |
| Existing public site | `https://ikun.love` | Verify the existing Sub2API `/health` route; do not change the root proxy during the isolated install. |
| Deploy materials | `deploy/ikun.love/` | Use its Compose and target parameter templates. |
| Deploy directory | `/opt/new-api` (verify) | Preserve `.env`, `data`, and `logs`. |
| App service/container | `new-api` / `ikun-new-api` | Recreate only the new-api service; leave `sub2api.service` untouched. |
| App port | `127.0.0.1:3000` | Check local `/api/status`. |
| Compose file | `docker-compose.1panel.yml` | Complete three-service Compose with private network and named volumes. |
| Docker network | `ikun-new-api-network` | Dedicated network; never attach to `1panel-network`. |
| PostgreSQL | `ikun-new-api-postgres` | Dedicated `postgres:18.4-alpine` container and volume; app uses only restricted `new_api_app` in `new_api`. |
| Redis | `ikun-new-api-redis` | Dedicated `redis:8.8.0` container and AOF volume; use logical database `0`. |
| Existing 1Panel services | `1Panel-postgresql-8Kr6` / `1Panel-redis-xsdn` | Must remain running and untouched for Sub2API. |
| Runtime `.env` | `/opt/new-api/.env`, mode `600` | Preserve secrets; never print or replace them. |

The live `ikun.love` server currently runs `/opt/sub2api/sub2api` as
`sub2api.service` on port 8080, with OpenResty proxying the public root to that
port. The new-api install is parallel and loopback-only; do not stop or edit
that service, its database, Redis db0, or the OpenResty configuration.

Verify current host memory and disk before writes. Do not build the frontend,
Go binary, or Docker multi-stage image on the target host.

## Last successful release record

These values are historical evidence, not defaults for the next release:

- Branch: `feature/initial-alltokenapi-ui`
- Commit and deployment marker: `8180dddf27e925d69cf365caae6f8cccc72eeef9`
- Release marker: `8180dddf-v1`
- Runtime image ID: `sha256:855daccd5ad8a3ae5ec22bf794b4dc0b4102957c601d26ecb2e778e4d9030ae9`
- Preserved rollback image ID: `sha256:cf03f277a1c13ad27ad33b971ef3ba31e1aff3f6ce3f4fbec472eae60e9bea7c`
- Runtime `/new-api` SHA-256: `f8228896cacb4616046485b6a50a03cdee8883bc04eeee5bf845d90155336c04`

## Read-only baseline (legacy alltokenapi)

Run these before a deployment without expanding `.env`:

```bash
cd /opt/new-api
printf 'commit='; cat .deploy-commit 2>/dev/null || true
printf 'release='; cat .deploy-release 2>/dev/null || true
docker ps --filter name='^/new-api$' --format '{{.Names}} {{.Status}} {{.Image}}'
docker inspect new-api --format 'image={{.Image}} status={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restarts={{.RestartCount}}'
docker inspect 1Panel-postgresql-2LOJ --format 'status={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
docker inspect 1Panel-redis-pDR8 --format 'status={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
docker image inspect new-api:local --format 'id={{.Id}} revision={{index .Config.Labels "org.opencontainers.image.revision"}}'
curl -fsS http://127.0.0.1:3000/api/status
```

Do not use `docker compose down`. Do not restart Docker as part of a normal release.

For `ikun.love`, substitute the verified target directory, service/container,
PostgreSQL, and Redis names from the dedicated target contract and
`deploy/ikun.love/deployment.env.example`. Do not expand `.env` while checking
the baseline.
