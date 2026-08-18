---
name: deploy-new-api
description: Build, deploy, verify, diagnose, and roll back QuantumNous/new-api from the Windows workstation to the selected SSH deployment host (`aliyun`, `alltokenapi`, or `ikun.love`), using a locally cross-compiled Linux binary and Docker runtime/data services. Use for new-api releases, `/opt/new-api`, 1Panel PostgreSQL/Redis coexistence, first-install bootstrap, artifact upload, failed remote Docker builds, production health checks, or rollback work.
---

# Deploy new-api

Deploy the latest verified commit for the selected release ref without compiling on the target server. Reuse the existing runtime image and 1Panel data services, then prove that the running binary matches the local artifact.

## Select the deployment host first

Before every deployment, ask the user explicitly: `部署到 aliyun（开发）、alltokenapi（生产）还是 ikun.love（生产）？` Do not infer the target from the current branch, historical notes, DNS, or the last deployment. Set the selected SSH alias as `$deployHost` and use it consistently for all remote commands.

- `aliyun`: development/test server. Follow the test-topology constraints in `AGENTS.md`: the remote service and the local development process intentionally use `NODE_TYPE=master` and share the Aliyun test PostgreSQL/Redis, which remain isolated from production.
- `alltokenapi`: production server. This is the current production deployment host and must use the production authorization gate, live DNS verification, and rollback checks below.
- `ikun.love`: production server at SSH alias `ikun.love`, with public origin `https://ikun.love`. Use the target materials in `deploy/ikun.love/`, release ref `ikun.love`, live DNS verification, and the production authorization gate. This target is an isolated parallel install: reuse only the existing PostgreSQL and Redis *containers*, create the `new_api` database/`new_api_app` role and Redis db1, and never stop or reconfigure Sub2API, its database, Redis db0, or OpenResty. Do not reuse `alltokenapi`'s public URL, `.env`, database/Redis names, OpenResty configuration, or deployment snapshot.

## Require the latest target ref

For `aliyun` and `alltokenapi`, deploy only the latest commit on `origin/master`. For `ikun.love`, deploy only the latest commit on `origin/ikun.love`. Before building or uploading anything, fetch the selected branch and require all three SHAs to be identical:

- local `refs/heads/<release-ref>`
- fetched `refs/remotes/origin/<release-ref>`
- live `refs/heads/<release-ref>` returned by `git ls-remote origin`

Stop if the selected ref is missing, any SHA differs, the remote query fails, or the worktree contains uncommitted changes that are intended for the release. Never deploy a tag, an arbitrary commit, or a locally ahead/behind release ref. Build the verified SHA from a detached clean worktree; do not require switching the user's current worktree to the release branch.

## Load the right context

- Read [references/deployment-playbook.md](references/deployment-playbook.md) before preparing or deploying a release.
- Read [references/production-topology.md](references/production-topology.md) before any server command. Treat its snapshot values as hints and verify live state.
- Read [references/incident-recovery.md](references/incident-recovery.md) when Docker returns EOF, the host is memory-constrained, the app restarts unexpectedly, or dependencies are not ready.

## Enforce the safety contract

1. Require the selected deployment host in the current conversation before any deployment. For `alltokenapi` and `ikun.love`, also require explicit authorization before a remote write, image retag, container recreation, or production restart. Read-only inspection and local builds are allowed after the host is selected.
2. Inspect repository instructions and `git status` first. Preserve unrelated dirty files; build only the verified latest commit for the selected release ref from a detached clean worktree.
3. Never print, upload, replace, or commit `.env` values. Never overwrite `/opt/new-api/.env`, `data`, or `logs`.
4. Never add or recreate PostgreSQL or Redis containers. The production app must reuse the existing 1Panel containers and external `1panel-network`; for `ikun.love`, only the new database/role and Redis logical db1 may be provisioned.
5. Never run a source or multi-stage Docker build on the production host. Its memory is insufficient and a failed build can OOM-kill Docker.
6. Keep a rollback image until the new release passes local checks and the target's existing public service has been proven unaffected. A loopback-only `ikun.love` install intentionally has no new-api public check.
7. Verify every destructive cleanup target is inside the expected temporary directory before removal.

## Execute the release

1. Fetch the selected release ref, verify the local ref, fetched tracking ref, and live remote ref resolve to the same SHA, and use only that SHA as the release commit. For `ikun.love`, select `ikun.love` and the isolated target contract in `deploy/ikun.love/`; do not change the existing public root route. Stop on any mismatch.
2. Use `scripts/build-release.ps1` to create a same-drive detached worktree, install the exact locked frontend dependencies in that worktree, build the frontend, cross-compile the Linux binary, and emit a SHA-256 manifest plus archive.
3. Inspect the artifact's ELF magic, `go version -m` metadata, commit, and SHA. Upload only the artifact archive and remote deploy script.
4. For a first `ikun.love` install, run `bootstrap-config.sh` to create only the new PostgreSQL database/role, verify Redis db1 is empty, generate new-api-only secrets, and write `.env`; run `bootstrap-image.sh` to seed the target runtime tag. Verify the uploaded archive SHA before extraction and run the binary with `--help`.
5. Use `scripts/deploy-binary.sh` on the host. It preserves the current target image as a rollback tag, replaces `/new-api` in a stopped temporary container, commits a candidate image, smoke-tests it, recreates only the `new-api` service, and rolls back if local health fails.
6. Verify the image revision label, runtime binary SHA, container health/restart count, the isolated PostgreSQL and Redis containers, local `/api/status`, and that Sub2API's service/8080/public health remain unchanged. Do not require a new-api public URL for the isolated `ikun.love` install.
7. Update only `.deploy-commit`, `.deploy-release`, and any deliberately maintained source snapshot after success. Remove temporary archives/worktrees only after verification; retain the rollback image.

## Use the scripts

Run the local preflight without creating anything:

```powershell
$repoRoot = git rev-parse --show-toplevel
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" -ReleaseRef master -ValidateOnly
```

Build an exact revision:

```powershell
$repoRoot = git rev-parse --show-toplevel
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" `
  -ReleaseRef <release-ref> `
  -Commit <verified-release-sha>
```

Display remote deploy arguments:

```bash
bash deploy-binary.sh --help
```

Prefer uploading the script as a file. If a multiline remote command is unavoidable, base64-encode it locally and decode it remotely to avoid PowerShell, SSH, and Bash quoting corruption.
