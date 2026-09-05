# Deployment topology

This is a last-known snapshot from 2026-09-05. Verify every mutable value on the host before using it. Values in this file are discovery hints, not permission to change a server.

## Deployment host selection

Confirm the target before every deployment. The release branch is part of the target contract and must not be inferred from the checked-out branch.

| Environment | SSH alias | Host | Deployment scope |
| --- | --- | --- | --- |
| Development/test | `aliyun` | Verify from local SSH config | Deploy the configured development branch (normally `master`). The remote test service and local development process both use `NODE_TYPE=master` and share the Aliyun test PostgreSQL/Redis; keep those services isolated from production. |
| Production | `alltokenapi` | `154.37.213.1` (verify live) | Deploy the configured `master` production branch using the existing 1Panel stack. |
| Production | `ikun.love` | `64.90.0.95` (verify live; SSH alias is authoritative) | Deploy only `origin/ikun.love` to `https://ikun.love` using the existing `ikun-new-api` Compose stack. |

## Service map

### `alltokenapi`

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

### `ikun.love`

The live inspection on 2026-09-05 found the following topology. Re-check it before every release because Compose names and image tags are mutable.

| Role | Last-known value | Required behavior |
| --- | --- | --- |
| SSH alias | `ikun.love` | Use this alias for all remote commands for this target. |
| Host | `64.90.0.95` | Verify DNS and SSH identity before a production write. |
| Release branch | `ikun.love` | Fetch and compare local, tracking, and live `origin/ikun.love` SHAs; never substitute `master` or a feature branch. |
| Public site | `https://ikun.love` | Verify `/api/status` and at least one changed public route after local health passes. |
| Deploy directory | `/opt/new-api` | Preserve `.env`, `data`, and `logs`. |
| Compose file | `docker-compose.1panel.yml` | Use `--no-build --no-deps`; recreate only the `new-api` service. |
| Compose project | `ikun-new-api` | Do not change the project name or recreate its data services. |
| App image tag | `new-api:ikun` | Preserve the current image under a rollback tag before switching. |
| App service | `new-api` | Recreate only this service. |
| App container | `ikun-new-api` | Check health and runtime binary after switching. |
| App port | `127.0.0.1:3000` | Check `/api/status` locally. |
| Runtime env | `.env`, mode `600` | Never display or replace secret values. |
| Docker network | `ikun-new-api-network` | Reuse it; do not create a second database network. |
| PostgreSQL | `ikun-new-api-postgres` | Must be running and healthy before switching the app. |
| Redis | `ikun-new-api-redis` | Must be running before switching the app. |

The `alltokenapi` host has roughly 1.8 GiB RAM plus swap. Do not build the frontend, Go binary, or Docker multi-stage image there. The `ikun.love` host is a separate dedicated stack; its observed capacity is not a deployment default, and the same no-remote-build policy still applies.

## Last successful release record

These values are historical evidence, not defaults for the next release:

- Branch: `feature/initial-alltokenapi-ui`
- Commit and deployment marker: `8180dddf27e925d69cf365caae6f8cccc72eeef9`
- Release marker: `8180dddf-v1`
- Runtime image ID: `sha256:855daccd5ad8a3ae5ec22bf794b4dc0b4102957c601d26ecb2e778e4d9030ae9`
- Preserved rollback image ID: `sha256:cf03f277a1c13ad27ad33b971ef3ba31e1aff3f6ce3f4fbec472eae60e9bea7c`
- Runtime `/new-api` SHA-256: `f8228896cacb4616046485b6a50a03cdee8883bc04eeee5bf845d90155336c04`

## Read-only baseline

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

For `ikun.love`, use the live names rather than the `alltokenapi` defaults:

```bash
cd /opt/new-api
printf 'commit='; cat .deploy-commit 2>/dev/null || true
printf 'release='; cat .deploy-release 2>/dev/null || true
docker inspect ikun-new-api --format 'image={{.Config.Image}} id={{.Image}} status={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restarts={{.RestartCount}}'
docker inspect ikun-new-api-postgres --format 'status={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
docker inspect ikun-new-api-redis --format 'status={{.State.Status}}'
docker image inspect new-api:ikun --format 'id={{.Id}} revision={{index .Config.Labels "org.opencontainers.image.revision"}} release={{index .Config.Labels "com.new-api.release"}}'
curl -fsS http://127.0.0.1:3000/api/status
```

Do not use `docker compose down`. Do not restart Docker as part of a normal release.
