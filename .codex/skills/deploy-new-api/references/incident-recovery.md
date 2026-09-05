# Incident recovery

## Remote Docker build ends with EOF

Treat `rpc error`, BuildKit EOF, or a vanished Docker connection as a host incident, not a source-code build error, until proven otherwise.

The recorded failure transferred only about 21 MB of context but spent roughly five minutes doing so. The build then ended in EOF. `dmesg` showed the OOM killer terminated `dockerd`, whose RSS was about 1.1 GiB, on a host with roughly 1.8 GiB RAM.

Collect evidence before changing anything:

```bash
date -Is
free -h
swapon --show
docker ps -a
systemctl status docker --no-pager
journalctl -u docker --since '-30 min' --no-pager | tail -n 200
dmesg -T | grep -Ei 'out of memory|oom|killed process' | tail -n 50
```

Do not retry the remote build. Move to the local binary build path.

## Docker restarted and the app failed first

An OOM-triggered Docker restart can briefly restart the app, PostgreSQL, and Redis together. The app may fail because PostgreSQL is not ready yet; this does not prove the old image is bad.

Recover in dependency order:

1. Confirm Docker is stable and not still restarting.
2. Wait for the selected target's PostgreSQL container to become healthy (`1Panel-postgresql-2LOJ` for `alltokenapi`, `ikun-new-api-postgres` for `ikun.love`).
3. Confirm the selected target's Redis container is running (`1Panel-redis-pDR8` for `alltokenapi`, `ikun-new-api-redis` for `ikun.love`).
4. Start or recreate only `new-api` if it did not recover automatically.
5. Verify local `/api/status`, then public status.

Do not recreate either data service and do not run `docker compose down`.

## Candidate fails local health

The deploy script automatically retags the preserved rollback image as the selected live image tag (`new-api:local` for the default stack, `new-api:ikun` for `ikun.love`) and recreates only the app service. After rollback, verify:

```bash
docker inspect new-api --format 'image={{.Image}} status={{.State.Status}} health={{.State.Health.Status}} restarts={{.RestartCount}}'
curl -fsS http://127.0.0.1:3000/api/status
docker logs --tail 200 new-api
```

Keep the failed candidate image for inspection until the cause is understood. Compare its revision label and binary SHA with the intended artifact.

## Local frontend build emits malformed drive paths

Paths such as `F:Project...` usually indicate a cross-drive Windows junction or symlink resolution problem, not a missing font. Use the default build script, which keeps the detached worktree on the repository drive and installs its own frozen dependencies. Do not link `node_modules` across drives.

## Public check fails while local health passes

Do not immediately rebuild or roll back. Inspect the reverse-proxy path separately:

```bash
curl -v http://127.0.0.1:3000/api/status
curl -v https://<selected-domain>/api/status
```

Check OpenResty routing, TLS, and upstream connectivity. A healthy local app with a failed public check is usually a proxy/domain problem, not an artifact problem.
