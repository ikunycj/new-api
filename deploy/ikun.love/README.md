# Isolated ikun.love Docker deployment

This directory installs the ikun.love branch as a complete, independent
Docker stack on ssh ikun.love. The stack contains new-api, PostgreSQL, and
Redis. The running Sub2API service is on ssh ikun.love-sub2api and is not
replaced or connected to.

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
| Legacy Sub2API SSH alias | ikun.love-sub2api |
| Legacy Sub2API port | 127.0.0.1:8080 on ikun.love-sub2api |
| Public health | https://ikun.love/api/status |

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
  8006). Public HTTPS for ikun.love is terminated by the target host's Nginx
  and reverse-proxied to the app on 127.0.0.1:3000.
- No command in these materials runs docker compose down, edits OpenResty, or
  stops/restarts sub2api.service.

## Files

- docker-compose.1panel.yml: complete three-service stack. The filename is
  retained for compatibility with the shared deploy script; it does not use
  1Panel services.
- .env.example: non-secret configuration shape.
- deployment.env.example: target operator settings.
- bootstrap-config.sh: validates the reviewed .env, starts only this stack's
  data services, and creates the least-privilege PostgreSQL role/database.
- bootstrap-image.sh: seeds the runtime image tag without starting any service.
- baseline.sh: read-only inventory for the fresh host and port/collision guards.
- verify.sh: stack health, database/Redis isolation, and fresh-host checks. The
  legacy service is verified separately through `ikun.love-sub2api`.

The source of truth for this target is `deploy/ikun.love/.env`. Generate it
locally with `.codex/skills/deploy-new-api/scripts/prepare-deployment-env.ps1`;
the script never overwrites an existing file and prints only metadata and
SHA-256. Never copy the Sub2API environment into it, commit it, or print its
values. An administrator must review the generated file and explicitly approve
its SHA before it can be uploaded. The installed `/opt/new-api/.env` must be
mode 600 and byte-for-byte identical to that approved file.

## Preflight

Confirm both SSH targets, DNS, and the old service before any remote write:

~~~powershell
ssh -G ikun.love | Select-String '^(hostname|user|port|identityfile) '
ssh -G ikun.love-sub2api | Select-String '^(hostname|user|port|identityfile) '
Resolve-DnsName ikun.love
Resolve-DnsName test.ikun.love
ssh ikun.love 'docker ps -a --format "{{.Names}}"; ss -lnt | grep -E ":3000\\b|:8006\\b" || true'
ssh ikun.love-sub2api 'systemctl is-active sub2api.service; ss -lnt | grep -E ":8080\\b"'
~~~

Stage the materials and run the read-only baseline as admin:

~~~bash
sudo bash baseline.sh
~~~

An absent /opt/new-api directory, stack containers, network, and volumes is
expected on the new host's first install. The separate legacy baseline on
`ikun.love-sub2api` must show Sub2API active and port 8080 bound, with its
public health URL available at `https://test.ikun.love/api/v1/settings/public`.
The new host's public health URL is `https://ikun.love/api/status`.

## Build and stage

Build the exact latest ikun.love ref from a detached clean worktree, then
record the manifest's binary and archive SHA-256 values:

~~~powershell
$repoRoot = git rev-parse --show-toplevel
$releaseCommit = (git ls-remote origin refs/heads/ikun.love).Split()[0]
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" `
  -DeploymentAlias ikun.love -ReleaseRef ikun.love -Commit $releaseCommit
~~~

The archive and manifest are written to
`deploy/ikun.love/artifacts/<full-commit>/` and
`deploy/ikun.love/artifacts/new-api-<full-commit>-linux-amd64.tar.gz`.

Prepare the target environment locally. If the file is missing, the first
command creates it and the process stops for administrator review. Do not
upload or approve it automatically:

~~~powershell
$envState = & "$repoRoot\.codex\skills\deploy-new-api\scripts\prepare-deployment-env.ps1" `
  -DeploymentAlias ikun.love
# Review deploy/ikun.love/.env without printing its values.
# After approval, pass the reviewed SHA-256 explicitly:
$approved = & "$repoRoot\.codex\skills\deploy-new-api\scripts\prepare-deployment-env.ps1" `
  -DeploymentAlias ikun.love -ApproveSha256 <reviewed-env-sha256>
if (-not $approved.ReadyForUpload) { throw 'Environment approval is required' }
~~~

Stage all deployment files in a mode-700 admin directory. Upload the approved
environment file only after the review gate above:

~~~powershell
ssh ikun.love 'install -d -m 700 ~/ikun-new-api-stage'
scp deploy/ikun.love/.env ikun.love:~/ikun-new-api-stage/.env
scp deploy/ikun.love/docker-compose.1panel.yml ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/bootstrap-config.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/bootstrap-image.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/baseline.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/verify.sh ikun.love:~/ikun-new-api-stage/
scp .codex/skills/deploy-new-api/scripts/deploy-binary.sh ikun.love:~/ikun-new-api-stage/
scp deploy/ikun.love/artifacts/new-api-<full-commit>-linux-amd64.tar.gz ikun.love:~/ikun-new-api-stage/
ssh ikun.love 'set -eu; chmod 600 ~/ikun-new-api-stage/.env; test ! -e /opt/new-api/.env && test ! -L /opt/new-api/.env; sudo install -d -m 750 /opt/new-api /opt/new-api/data /opt/new-api/logs; sudo install -m 600 ~/ikun-new-api-stage/.env /opt/new-api/.env; sudo install -m 755 ~/ikun-new-api-stage/*.sh /opt/new-api/; sudo install -m 644 ~/ikun-new-api-stage/docker-compose.1panel.yml /opt/new-api/; sudo install -m 640 ~/ikun-new-api-stage/<archive-name> /opt/new-api/<archive-name>'
~~~

The install command is fail-closed: if `/opt/new-api/.env` already exists
(including a symlink), it stops and does not overwrite it. Compare only
SHA-256 fingerprints on the local machine, staging path, and installed path;
never print or diff the file contents.

Verify the archive hash on both machines before extraction:

~~~powershell
Get-FileHash -Algorithm SHA256 <release-archive>
ssh ikun.love 'sha256sum /opt/new-api/<archive-name>'
~~~

## Bootstrap the complete stack

Run bootstrap-config.sh once with the administrator-approved SHA. It validates
the installed .env, starts only ikun-new-api-postgres and ikun-new-api-redis,
waits for both health checks,
creates new_api owned by new_api_app, and verifies app-role connectivity. It is
safe to retry after an interrupted first attempt:

~~~bash
cd /opt/new-api
sudo bash ./bootstrap-config.sh --expected-env-sha <approved-env-sha256>
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
sudo env EXPECTED_BINARY_SHA=<binary-sha256> EXPECTED_ENV_SHA=<approved-env-sha256> bash ./verify.sh
sudo docker compose --project-name ikun-new-api --env-file .env \
  -f docker-compose.1panel.yml ps
~~~

Verification must prove:

- all three dedicated containers are running and healthy;
- the app is loopback-bound and its local status is successful;
- the database owner is new_api_app, its role is non-superuser,
  non-createdb/non-createrole/non-replication/non-bypassrls/non-inherit with
  no memberships, and the app probe uses new_api/new_api_app only;
- Redis authentication works in the dedicated container;
- the old host `ssh ikun.love-sub2api` still has `sub2api.service` active,
  port 8080 bound, and its local/public health checks available;
- no dedicated network member is a Sub2API or 1Panel container.

If the app candidate fails local health, deploy-binary.sh automatically restores
new-api:rollback-<release> and recreates only ikun-new-api. Keep the failed
candidate image for diagnosis. A binary rollback does not undo database
migrations already applied by the candidate, so take a PostgreSQL backup before
subsequent upgrades and retain the named volumes.

The public route is served by the target host's Nginx configuration in
`deploy/ikun.love/nginx/ikun.love.conf`, using the certificate issued for
`ikun.love` and proxying to `127.0.0.1:3000`. The legacy Sub2API host is
available independently at `https://test.ikun.love`; its service remains
untouched by this stack. A fresh database has no admin account yet. Use an SSH
tunnel to open `http://127.0.0.1:3000/setup`, complete
the first-run setup, and confirm `GET /api/setup` reports database type
`postgres` (not SQLite) before using the instance.
