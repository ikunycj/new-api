# Isolated ikun.love deployment materials

This directory deploys the `ikun.love` branch as a new Dockerized `new-api`
instance on `ssh ikun.love`. It does not replace the running Sub2API service.

| Setting | Isolated target contract |
| --- | --- |
| SSH alias | `ikun.love` |
| Release ref | `origin/ikun.love` |
| New-api directory | `/opt/new-api` |
| New-api container | `ikun-new-api` |
| New-api local status | `http://127.0.0.1:3000/api/status` |
| Existing Sub2API port | `127.0.0.1:8080` |
| Existing public health | `https://ikun.love/api/v1/settings/public` |
| PostgreSQL container | `1Panel-postgresql-8Kr6` |
| New-api database/role | `new_api` / `new_api_app` |
| Redis logical database | `1` (Sub2API uses `0`) |

## Isolation guarantees

- Never stop, restart, reconfigure, or remove `sub2api.service`.
- Never edit `/opt/1panel/www/sites/ikun.love/proxy/root.conf` or TLS files
  during this deployment. The existing public root remains on Sub2API.
- New-api uses the existing PostgreSQL container only as a host service. The
  bootstrap creates a new `new_api` database and a `new_api_app` login role
  with `NOSUPERUSER`, `NOCREATEDB`, and `NOCREATEROLE`; no `sub2api` database
  credentials are used.
- New-api uses the existing Redis container only through logical database `1`.
  The bootstrap refuses to continue if that database is not empty. Redis db0
  remains untouched for Sub2API.
- New-api binds host port `3000` to loopback. A public cutover requires a
  separate explicit request and a routing plan with rollback.

## Files

- `.env.example`: non-secret runtime template.
- `deployment.env.example`: target operator settings.
- `docker-compose.1panel.yml`: app-only Compose contract with both existing
  data services and isolated credentials.
- `bootstrap-config.sh`: creates the database/role, checks Redis db1, generates
  new-api-only secrets, and writes `.env` without printing values.
- `bootstrap-image.sh`: seeds the target image tag from the official runtime
  image without starting or rebuilding Sub2API.
- `baseline.sh`: read-only inventory plus Sub2API preservation checks.
- `verify.sh`: new-api health and Sub2API non-regression checks.

The real `/opt/new-api/.env` is generated on the target with mode `600`.
Never copy the Sub2API environment into it, commit it, or print its values.

## Preflight

The deployment target and production authorization are selected in the current
task. Repeat the read-only checks before remote writes:

```powershell
ssh -G ikun.love | Select-String '^(hostname|user|port|identityfile) '
Resolve-DnsName ikun.love
ssh ikun.love 'systemctl is-active sub2api.service; ss -lnt | grep -E ":8080\\b"'
```

Run the baseline as `admin`; it automatically uses passwordless `sudo docker`:

```bash
sudo bash baseline.sh
```

The baseline must show Sub2API active and its local/public health route
available. A missing `/opt/new-api` directory and image are expected initially.

## Build and stage

Build the exact latest `ikun.love` ref from a detached clean worktree, then
inspect the manifest and both SHA-256 values:

```powershell
$repoRoot = git rev-parse --show-toplevel
$releaseCommit = (git ls-remote origin refs/heads/ikun.love).Split()[0]
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" `
  -ReleaseRef ikun.love `
  -Commit $releaseCommit
```

Stage files in the admin home because `admin` cannot write `/opt` directly:

```powershell
ssh ikun.love 'mkdir -p ~/ikun-new-api-stage'
scp deploy/ikun.love/docker-compose.1panel.yml ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/bootstrap-config.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/bootstrap-image.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/baseline.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/verify.sh ikun.love:~/ikun-new-api-stage/
scp .codex/skills/deploy-new-api/scripts/deploy-binary.sh ikun.love:~/ikun-new-api-stage/
scp <release-archive> ikun.love:~/ikun-new-api-stage/
ssh ikun.love 'sudo install -d -m 750 /opt/new-api /opt/new-api/data /opt/new-api/logs; sudo install -m 755 ~/ikun-new-api-stage/*.sh /opt/new-api/; sudo install -m 644 ~/ikun-new-api-stage/docker-compose.1panel.yml /opt/new-api/'
```

## Bootstrap data services and image

Run the configuration bootstrap as root. It reads the existing container
credentials internally, creates only the new database/role, checks that Redis
db1 is empty, generates new-api-only secrets, and records only `.env` mode and
SHA-256. It never displays a password or connection string:

```bash
cd /opt/new-api
sudo bash ./bootstrap-config.sh \
  --postgres-container 1Panel-postgresql-8Kr6 \
  --redis-container 1Panel-redis-xsdn \
  --database new_api \
  --role new_api_app \
  --redis-db 1
```

Seed the runtime image tag. This pulls/tags an image only; it does not start or
recreate any running service:

```bash
sudo bash ./bootstrap-image.sh \
  --base-image calciumion/new-api:latest \
  --image-tag new-api:ikun
```

## Deploy only the new-api container

Run the SHA-verified app switch as root. Pass both data-container names so the
script gates on their readiness; it does not recreate either one:

```bash
cd /opt/new-api
sudo bash ./deploy-binary.sh \
  --archive ./new-api-<full-commit>-linux-amd64.tar.gz \
  --archive-sha <archive-sha256> \
  --binary-sha <binary-sha256> \
  --commit <full-commit-sha> \
  --release <short-sha>-v1 \
  --deploy-dir /opt/new-api \
  --compose-file docker-compose.1panel.yml \
  --env-file .env \
  --image-tag new-api:ikun \
  --container ikun-new-api \
  --postgres 1Panel-postgresql-8Kr6 \
  --redis 1Panel-redis-xsdn \
  --local-url http://127.0.0.1:3000/api/status
```

The script replaces only the `new-api` service, preserves `/opt/new-api/data`
and `logs`, verifies the runtime binary SHA, and retains
`new-api:rollback-<release>`. It never runs `docker compose down`.

## Verify and rollback

```bash
cd /opt/new-api
sudo bash ./verify.sh
sudo docker compose --env-file .env -f docker-compose.1panel.yml ps
```

Verification must prove:

- `ikun-new-api` is running and healthy, and its runtime binary SHA matches the
  local manifest.
- PostgreSQL `1Panel-postgresql-8Kr6` is healthy and Redis
  `1Panel-redis-xsdn` is running; only `new_api`/`new_api_app` and Redis db1 are
  used by new-api.
- `http://127.0.0.1:3000/api/status` succeeds.
- `sub2api.service` remains active, port `8080` remains bound, and both its
  local and public health checks still succeed.
- New-api public routing is intentionally skipped until a separate cutover is
  authorized.

Keep the rollback image. If new-api fails local health, the deploy script
restores the previous `new-api:ikun` image without touching Sub2API.
