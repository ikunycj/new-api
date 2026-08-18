# ikun.love deployment materials

This directory is the target-specific deployment contract for the new-api
release on `ssh ikun.love`, served publicly at `https://ikun.love`.

It is intentionally separate from the previous `alltokenapi` deployment:

| Setting | ikun.love target |
| --- | --- |
| SSH alias | `ikun.love` |
| Release ref | `origin/ikun.love` |
| Public origin | `https://ikun.love` |
| Public status | `https://ikun.love/api/status` |
| Local status | `http://127.0.0.1:3000/api/status` |
| Default app directory | `/opt/new-api` (verify before use) |

## Files

- `.env.example`: application environment template with no credentials.
- `deployment.env.example`: non-secret operator settings for this target.
- `docker-compose.1panel.yml`: app-only Compose file; it reuses the existing
  1Panel PostgreSQL, Redis, and `1panel-network`.
- `baseline.sh`: read-only target inventory; it never prints `.env` values.
- `verify.sh`: local and public status checks after a release.

The real server `.env` must be created and maintained on `ikun.love` with mode
`600`. Preserve its existing `SESSION_SECRET`, `CRYPTO_SECRET`, database
credentials, Redis credentials, `data`, and `logs`; never copy them from the
workstation or commit them here.

## Preflight

This task only prepares materials. A deployment still requires explicit
production authorization in the current conversation. Before any remote write:

1. Verify the SSH alias and live DNS without exposing secrets:

   ```powershell
   ssh -G ikun.love | Select-String '^(hostname|user|port|identityfile) '
   Resolve-DnsName ikun.love
   ```

2. Capture a read-only baseline on the target. Verify the actual deployment
   directory, Compose file, app container, PostgreSQL container, Redis
   container, Docker network, and existing `.env` mode. Do not print `.env`.
   After uploading `baseline.sh`, run it with the two audited data-container
   names:

   ```bash
   POSTGRES_CONTAINER=<target-postgres-container> \
   REDIS_CONTAINER=<target-redis-container> \
   ./baseline.sh
   ```
3. Confirm OpenResty/TLS already routes `https://ikun.love` to the app's local
   port. This repository does not overwrite the target's certificate or proxy
   configuration.
4. Fill the target values in a private copy of `.env.example` and set mode 600.
   The `REPLACE_WITH_...` markers must not remain in the server file.

## Build the ikun.love release

Build the exact remote branch, not the current mutable directory. The generic
build script now accepts a release ref:

```powershell
$repoRoot = git rev-parse --show-toplevel
$releaseCommit = (git ls-remote origin refs/heads/ikun.love).Split()[0]
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" `
  -ReleaseRef ikun.love `
  -Commit $releaseCommit
```

Inspect the emitted binary, manifest, and archive SHA-256 values before upload.
Upload only the archive and the generic `deploy-binary.sh` script to the
staging area on `ikun.love`. Do not upload `.env.example` as `.env`.

## Install or update the app-only Compose contract

Only after the target baseline and authorization are complete, copy the
Compose file into the verified deployment directory. Do not use `--build`,
`docker compose down`, or recreate PostgreSQL/Redis:

```powershell
scp deploy/ikun.love/docker-compose.1panel.yml ikun.love:/opt/new-api/
scp .codex/skills/deploy-new-api/scripts/deploy-binary.sh ikun.love:/opt/new-api/
```

The exact target path and the PostgreSQL/Redis container names must come from
the read-only baseline, not from this example. The existing target `.env` must
already contain the values consumed by the Compose file.

## Switch the app image

Run the uploaded script with the exact artifact hashes and target-specific
parameters. Replace only the placeholder values obtained from the baseline:

```bash
cd /opt/new-api
bash ./deploy-binary.sh \
  --archive ./new-api-<full-commit>-linux-amd64.tar.gz \
  --archive-sha <archive-sha256> \
  --binary-sha <binary-sha256> \
  --commit <full-commit-sha> \
  --release <short-sha>-v1 \
  --deploy-dir /opt/new-api \
  --compose-file docker-compose.1panel.yml \
  --env-file .env \
  --postgres <target-postgres-container> \
  --redis <target-redis-container> \
  --local-url http://127.0.0.1:3000/api/status \
  --public-url https://ikun.love/api/status
```

The script verifies archive and binary hashes, preserves a rollback image,
recreates only the app service, verifies the runtime binary hash, and writes
`.deploy-commit` and `.deploy-release` only after local health succeeds.

## Verify and rollback

After the local checks pass, run `verify.sh` from the target deployment
directory (or use equivalent commands with the target values):

```bash
chmod 0755 verify.sh
./verify.sh
docker compose --env-file .env -f docker-compose.1panel.yml ps
```

Also verify the public homepage and the changed route through the actual
`ikun.love` OpenResty/TLS path. Keep the rollback image until the release is
known stable. If public checks fail while local health is good, inspect DNS,
TLS, and OpenResty routing before rebuilding or rolling back.
