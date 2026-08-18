# Isolated ikun.love Docker deployment

This directory installs the ikun.love branch as a complete, independent
Docker stack on ssh ikun.love. The stack contains new-api, PostgreSQL, and
Redis. It does not replace or connect to the running Sub2API service.

| Setting | Isolated target contract |
| --- | --- |
| SSH alias | ikun.love |
| Release ref | ikun.love |
| Deployment directory | /opt/new-api |
| Compose services | new-api, postgres, redis |
| App container | ikun-new-api |
| PostgreSQL container | ikun-new-api-postgres |
| Redis container | ikun-new-api-redis |
| Docker network | ikun-new-api-network |
| PostgreSQL volume | ikun-new-api-postgres-data |
| Redis volume | ikun-new-api-redis-data |
| App status | http://127.0.0.1:3000/api/status |
| Existing Sub2API port | 127.0.0.1:8080 |
| Existing public health | https://ikun.love/api/v1/settings/public |

## Isolation guarantees

- PostgreSQL and Redis are created by this Compose file. They are not the
  existing 1Panel containers and are not attached to 1panel-network.
- The stack uses the target's verified `postgres:18.4-alpine` and
  `redis:8.8.0` image tags; neither dependency uses a floating `latest` tag.
- PostgreSQL has a private bootstrap administrator and a separate
  new_api_app application role. The app receives only the latter's DSN; the
  application role is NOSUPERUSER, NOCREATEDB, and NOCREATEROLE.
- Redis uses its own container, persistent AOF volume, password, and database
  index 0. No Sub2API Redis keys are visible to it.
- The app binds only to host loopback on port 3000 (metrics use loopback
  8006). The public ikun.love root remains on Sub2API until a separate
  routing request is authorized.
- No command in these materials runs docker compose down, edits OpenResty, or
  stops/restarts sub2api.service.

## Files

- docker-compose.1panel.yml: complete three-service stack. The filename is
  retained for compatibility with the shared deploy script; it does not use
  1Panel services.
- .env.example: non-secret configuration shape.
- deployment.env.example: target operator settings.
- bootstrap-config.sh: generates/validates .env, starts only this stack's data
  services, and creates the least-privilege PostgreSQL role/database.
- bootstrap-image.sh: seeds the runtime image tag without starting any service.
- baseline.sh: read-only inventory and Sub2API preservation baseline.
- verify.sh: stack health, database/Redis isolation, and Sub2API checks.

The real /opt/new-api/.env is generated with mode 600. Never copy the Sub2API
environment into it, commit it, or print its values.

## Preflight

Confirm the target, DNS, and the existing service before any remote write:

~~~powershell
ssh -G ikun.love | Select-String '^(hostname|user|port|identityfile) '
Resolve-DnsName ikun.love
ssh ikun.love 'systemctl is-active sub2api.service; ss -lnt | grep -E ":8080\\b"'
~~~

Stage the materials and run the read-only baseline as admin:

~~~bash
sudo bash baseline.sh
~~~

An absent /opt/new-api directory, stack containers, network, and volumes is
expected on first install. The baseline must show the existing Sub2API service
and its local/public health route available.

## Build and stage

Build the exact latest ikun.love ref from a detached clean worktree, then
record the manifest's binary and archive SHA-256 values:

~~~powershell
$repoRoot = git rev-parse --show-toplevel
$releaseCommit = (git ls-remote origin refs/heads/ikun.love).Split()[0]
& "$repoRoot\\.codex\\skills\\deploy-new-api\\scripts\\build-release.ps1" -ReleaseRef ikun.love -Commit $releaseCommit
~~~

Stage all deployment files in the admin home:

~~~powershell
ssh ikun.love 'mkdir -p ~/ikun-new-api-stage'
scp deploy/ikun.love/docker-compose.1panel.yml ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/bootstrap-config.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/bootstrap-image.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/baseline.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/verify.sh ikun.love:~/ikun-new-api-stage/
scp .codex/skills/deploy-new-api/scripts/deploy-binary.sh ikun.love:~/ikun-new-api-stage/
scp <release-archive> ikun.love:~/ikun-new-api-stage/
ssh ikun.love 'sudo install -d -m 750 /opt/new-api /opt/new-api/data /opt/new-api/logs; sudo install -m 755 ~/ikun-new-api-stage/*.sh /opt/new-api/; sudo install -m 644 ~/ikun-new-api-stage/docker-compose.1panel.yml /opt/new-api/; sudo install -m 640 ~/ikun-new-api-stage/<archive-name> /opt/new-api/<archive-name>'
~~~

Verify the archive hash on both machines before extraction:

~~~powershell
Get-FileHash -Algorithm SHA256 <release-archive>
ssh ikun.love 'sha256sum /opt/new-api/<archive-name>'
~~~

## Bootstrap the complete stack

Run bootstrap-config.sh once. It creates the .env file, starts only
ikun-new-api-postgres and ikun-new-api-redis, waits for both health checks,
creates new_api owned by new_api_app, and verifies app-role connectivity. It is
safe to retry after an interrupted first attempt:

~~~bash
cd /opt/new-api
sudo bash ./bootstrap-config.sh
sudo docker compose --project-name ikun-new-api --env-file .env \
  -f docker-compose.1panel.yml config --services
~~~

The last command must list new-api, postgres, and redis. Seed the application
runtime image before the binary switch:

~~~bash
sudo bash ./bootstrap-image.sh \
  --base-image calciumion/new-api:latest \
  --image-tag new-api:ikun
~~~

## Deploy the app release

The release script recreates only ikun-new-api; --no-deps leaves the dedicated
PostgreSQL and Redis containers running:

~~~bash
cd /opt/new-api
sudo bash ./deploy-binary.sh \
  --archive ./new-api-<full-commit>-linux-amd64.tar.gz \
  --archive-sha <archive-sha256> \
  --binary-sha <binary-sha256> \
  --commit <full-commit-sha> \
  --release <short-sha>-v1 \
  --deploy-dir /opt/new-api \
  --compose-file docker-compose.1panel.yml \
  --project-name ikun-new-api \
  --env-file .env \
  --image-tag new-api:ikun \
  --container ikun-new-api \
  --postgres ikun-new-api-postgres \
  --redis ikun-new-api-redis \
  --timeout 300 \
  --local-url http://127.0.0.1:3000/api/status
~~~

The script preserves /opt/new-api/data, /opt/new-api/logs, both named data
volumes, and a new-api:rollback-<release> image. It never recreates or
reconfigures Sub2API.

## Verify and rollback

~~~bash
cd /opt/new-api
sudo env EXPECTED_BINARY_SHA=<binary-sha256> bash ./verify.sh
sudo docker compose --project-name ikun-new-api --env-file .env \
  -f docker-compose.1panel.yml ps
~~~

Verification must prove:

- all three dedicated containers are running and healthy;
- the app is loopback-bound and its local status is successful;
- the database owner is new_api_app, its role flags are non-superuser, and the
  app probe uses new_api/new_api_app;
- Redis authentication works in the dedicated container;
- sub2api.service, port 8080, and both existing local/public health checks
  remain available;
- no dedicated network member is a Sub2API or 1Panel container.

If the app candidate fails local health, deploy-binary.sh automatically restores
new-api:rollback-<release> and recreates only ikun-new-api. Keep the failed
candidate image for diagnosis. A binary rollback does not undo database
migrations already applied by the candidate, so take a PostgreSQL backup before
subsequent upgrades and retain the named volumes.

New-api public routing is intentionally skipped for this parallel install;
https://ikun.love continues to serve Sub2API. A fresh database has no admin
account yet. Use an SSH tunnel to open `http://127.0.0.1:3000/setup`, complete
the first-run setup, and confirm `GET /api/setup` reports database type
`postgres` (not SQLite) before using the instance.
