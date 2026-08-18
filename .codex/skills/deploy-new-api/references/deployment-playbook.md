# Deployment playbook

## Contents

- [1. Gate and baseline](#1-gate-and-baseline)
- [2. Freeze and verify an exact release ref](#2-freeze-and-verify-an-exact-release-ref)
- [3. Build on the workstation](#3-build-on-the-workstation)
- [4. Upload and verify](#4-upload-and-verify)
- [5. Wrap and switch the image](#5-wrap-and-switch-the-image)
- [6. Prove the result](#6-prove-the-result)
- [7. Record and clean up](#7-record-and-clean-up)

## 1. Gate and baseline

First ask: `部署到 aliyun（开发）、alltokenapi（生产）还是 ikun.love（生产）？` Set `$deployHost` to the user's answer and do not run remote commands until the target is confirmed. For `alltokenapi` and `ikun.love`, confirm explicit permission for production writes. Then read repository instructions, inspect the dirty worktree, and capture a read-only server baseline from `production-topology.md`.

Target-specific defaults:

| SSH alias | Release ref | Public status URL | Materials |
| --- | --- | --- | --- |
| `aliyun` | `master` | verify live | repository defaults |
| `alltokenapi` | `master` | `https://alltokenapi.com/api/status` | legacy production topology |
| `ikun.love` | `ikun.love` | loopback-only new-api; existing `https://ikun.love` remains Sub2API | `deploy/ikun.love/` |

Check workstation tools:

```powershell
git --version
bun --version
go version
ssh -V
Get-Command scp
tar --version
```

Re-read `Dockerfile` before every release. Match its Go version and build experiment rather than trusting an old note. The successful 2026-07-21 build used Go 1.26.1, `GOEXPERIMENT=greenteagc`, Linux amd64, and `CGO_ENABLED=0`.

## 2. Verify the latest target release ref

Deploy only the latest commit on the selected release ref. Use `master` for `aliyun` and `alltokenapi`; use `ikun.love` for the `ikun.love` target. Do not package the current dirty directory or deploy an arbitrary commit. Preserve unrelated dirty files in the user's current worktree.

Fetch the remote tracking ref, then compare the local branch, fetched tracking ref, and a live remote query:

```powershell
git status --short --branch
$releaseRef = 'ikun.love' # use 'master' for aliyun/alltokenapi
git fetch origin $releaseRef
$localRelease = git rev-parse --verify "refs/heads/$releaseRef"
$fetchedRelease = git rev-parse --verify "refs/remotes/origin/$releaseRef"
$remoteReleaseLine = git ls-remote origin "refs/heads/$releaseRef"
if ($LASTEXITCODE -ne 0 -or -not $remoteReleaseLine) { throw "Cannot verify origin/$releaseRef" }
$remoteRelease = ($remoteReleaseLine -split '\s+')[0]
if ($localRelease -ne $fetchedRelease -or $localRelease -ne $remoteRelease) {
    throw "release mismatch: ref=$releaseRef local=$localRelease fetched=$fetchedRelease remote=$remoteRelease"
}
$releaseCommit = $localRelease
```

If the selected local ref is missing, ahead, or behind, stop. Update it to exactly match its remote tracking ref, then rerun the full check. Do not silently force-move a branch or substitute a different ref while reporting that local and remote match.

If GitHub HTTPS repeatedly times out, use the workstation's GitHub key over SSH port 443 without changing `origin`, then fetch and query the same selected ref:

```powershell
$env:GIT_SSH_COMMAND = 'ssh -i C:/Users/86139/.ssh/id_ed25519_github -o IdentitiesOnly=yes -o HostKeyAlias=github.com -p 443'
git fetch 'ssh://git@ssh.github.com:443/ikunycj/new-api.git' "${releaseRef}:refs/remotes/origin/${releaseRef}"
git ls-remote 'ssh://git@ssh.github.com:443/ikunycj/new-api.git' "refs/heads/$releaseRef"
Remove-Item Env:GIT_SSH_COMMAND
```

Use this only as a fallback and still require all three SHAs to match.

## 3. Build on the workstation

Run `scripts/build-release.ps1 -ReleaseRef $releaseRef -Commit $releaseCommit`. Its important invariants are:

- Create a detached worktree from the commit, not from working-tree files.
- Put the worktree on the same drive as the repository. A `C:` worktree linked to `F:` dependencies caused Rspack font paths such as `F:Project...` and failed the build.
- Run `bun install --frozen-lockfile` inside the worktree, then build `web/default` with Bun. Do not reuse the mutable repository `web/node_modules`; installed package versions can differ from the target commit's lockfile even when the build succeeds.
- Build the frontend from `web/default` with the frozen workspace lockfile.
- Cross-compile with `GOOS=linux`, `GOARCH=amd64`, `CGO_ENABLED=0`, and the current Dockerfile's `GOEXPERIMENT`.
- Set `common.Version` from `VERSION`. An empty `VERSION` is valid and matched the live deployment during the recorded release.
- Use `-buildvcs=false` because the worktree revision is recorded separately and deployment integrity comes from explicit hashes.

The script names the artifact directory and archive with the full commit SHA and fails if that artifact already exists. Use `-ForceRebuild` only when intentionally replacing the same commit's artifact; the old artifact is removed before the new build starts so a failed rebuild cannot leave a plausible stale release. The manifest records the Go and Bun versions, Dockerfile and frontend-lock Git blobs, build-script SHA, and binary SHA.

Treat the emitted archive SHA as the identity of that individual artifact. The manifest build time and gzip/tar timestamps mean archive bytes are not promised to reproduce across builds. The Linux binary should remain stable when the source, frozen dependencies, toolchain, and build script are unchanged. Review the binary, `manifest.json`, and `.tar.gz` output independently:

```powershell
Get-FileHash -Algorithm SHA256 <archive>
Get-FileHash -Algorithm SHA256 <binary>
go version -m <binary>
```

The binary must start with ELF magic `7f454c46`.

## 4. Upload and verify

For `alltokenapi`/`aliyun`, upload the archive and remote script to a staging
path, not over the live binary. For a first `ikun.love` install, stage the
target bootstrap scripts and Compose file in `~/ikun-new-api-stage`, then use
`sudo install` into the new `/opt/new-api` directory; never write the Sub2API
directory or its service files.

```powershell
scp <archive> ${deployHost}:/opt/new-api/
$repoRoot = git rev-parse --show-toplevel
scp "$repoRoot/.codex/skills/deploy-new-api/scripts/deploy-binary.sh" ${deployHost}:/opt/new-api/
```

Compare the local and remote archive hashes before extraction:

```powershell
Get-FileHash -Algorithm SHA256 <archive>
ssh $deployHost "sha256sum /opt/new-api/<archive-name>"
```

Do not print `.env` while checking the directory.

## 5. Wrap and switch the image

Invoke the uploaded script with full commit and hashes. Pass explicit
PostgreSQL/Redis container names for deployments that use them; omit neither
gate for the `ikun.love` full-stack install:

```bash
cd /opt/new-api
bash ./deploy-binary.sh \
  --archive ./<archive-name> \
  --archive-sha <archive-sha256> \
  --binary-sha <binary-sha256> \
  --commit <full-commit-sha> \
  --release <short-sha>-v1 \
  --postgres <target-postgres-container> \
  --redis <target-redis-container>
```

The script intentionally uses `docker create`, `docker cp`, and `docker commit` instead of `docker build`. It runs Compose with `--no-build --no-deps` so PostgreSQL and Redis are not recreated.

For remote multiline logic outside the script, encode the text locally as UTF-8/base64 and decode it remotely. Avoid nested PowerShell/SSH/Bash quoting.

## 6. Prove the result

Do not stop at `docker ps`. Verify all of the following:

```bash
docker inspect <target-app-container> --format 'image={{.Image}} status={{.State.Status}} health={{.State.Health.Status}} restarts={{.RestartCount}}'
docker image inspect <target-image-tag> --format 'id={{.Id}} revision={{index .Config.Labels "org.opencontainers.image.revision"}}'
docker cp new-api:/new-api /tmp/new-api.runtime
sha256sum /tmp/new-api.runtime
rm -f /tmp/new-api.runtime
docker inspect <target-postgres-container> --format '{{.State.Status}} {{.State.Health.Status}}'
docker inspect <target-redis-container> --format '{{.State.Status}}'
curl -fsS http://127.0.0.1:3000/api/status
curl -fsS <target-public-status-url> # omit for isolated ikun.love first install
curl -fsS -o /dev/null -w '%{http_code}\n' <target-public-route> # existing service guard for ikun.love
```

The runtime binary hash must equal the local artifact hash. Check the specific public route changed by the release, not only `/api/status`.

## 7. Record and clean up

The remote script writes `.deploy-commit` and `.deploy-release` only after local health passes. If the server keeps a source snapshot, sync only deliberate changed files after success; never overwrite `.env`, `data`, or `logs`, and never rebuild from that snapshot.

Remove upload archives and temporary worktrees only after all checks pass. Keep the rollback image tag until a later release is known stable.
