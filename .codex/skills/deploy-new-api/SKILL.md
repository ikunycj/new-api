---
name: deploy-new-api
description: Build, deploy, verify, diagnose, and roll back QuantumNous/new-api from the Windows workstation to the selected SSH deployment host (`aliyun` for development or `alltokenapi` for production), using a locally cross-compiled Linux binary and the existing Docker image. Use for new-api releases, `/opt/new-api`, 1Panel PostgreSQL/Redis coexistence, artifact upload, failed remote Docker builds, production health checks, or rollback work.
---

# Deploy new-api

Deploy the latest `origin/master` commit without compiling on the target server. Reuse the existing runtime image and 1Panel data services, then prove that the running binary matches the local artifact.

## Select the deployment host first

Before every deployment, ask the user explicitly: `部署到 aliyun（开发）还是 alltokenapi（生产）？` Do not infer the target from the current branch, historical notes, DNS, or the last deployment. Set the selected SSH alias as `$deployHost` and use it consistently for all remote commands.

- `aliyun`: development server. Follow the development-node constraints in `AGENTS.md`, including `NODE_TYPE=slave` and isolated PostgreSQL/Redis.
- `alltokenapi`: production server. This is the current production deployment host and must use the production authorization gate, live DNS verification, and rollback checks below.

## Require the latest master

Deploy only the latest commit on `origin/master`. Before building or uploading anything, fetch `origin master` and require all three SHAs to be identical:

- local `refs/heads/master`
- fetched `refs/remotes/origin/master`
- live `refs/heads/master` returned by `git ls-remote origin`

Stop if local `master` is missing, any SHA differs, the remote query fails, or the worktree contains uncommitted changes that are intended for the release. Never deploy the current feature branch, another branch, a tag, an arbitrary commit, or a locally ahead/behind `master`. Build the verified master SHA from a detached clean worktree; do not require switching the user's current worktree to `master`.

## Load the right context

- Read [references/deployment-playbook.md](references/deployment-playbook.md) before preparing or deploying a release.
- Read [references/production-topology.md](references/production-topology.md) before any server command. Treat its snapshot values as hints and verify live state.
- Read [references/incident-recovery.md](references/incident-recovery.md) when Docker returns EOF, the host is memory-constrained, the app restarts unexpectedly, or dependencies are not ready.

## Enforce the safety contract

1. Require the selected deployment host in the current conversation before any deployment. For `alltokenapi`, also require explicit authorization before a remote write, image retag, container recreation, or production restart. Read-only inspection and local builds are allowed after the host is selected.
2. Inspect repository instructions and `git status` first. Preserve unrelated dirty files; build only the verified latest `origin/master` commit from a detached clean worktree.
3. Never print, upload, replace, or commit `.env` values. Never overwrite `/opt/new-api/.env`, `data`, or `logs`.
4. Never add or recreate PostgreSQL or Redis. The production app must reuse the existing 1Panel containers and external `1panel-network`.
5. Never run a source or multi-stage Docker build on the production host. Its memory is insufficient and a failed build can OOM-kill Docker.
6. Keep a rollback image until the new release passes local and public checks.
7. Verify every destructive cleanup target is inside the expected temporary directory before removal.

## Execute the release

1. Fetch `origin master`, verify local `master`, `origin/master`, and live remote `master` resolve to the same SHA, and use only that SHA as the release commit. Stop on any mismatch.
2. Use `scripts/build-release.ps1` to create a same-drive detached worktree, install the exact locked frontend dependencies in that worktree, build the default frontend, create the classic placeholder, cross-compile the Linux binary, and emit a SHA-256 manifest plus archive.
3. Inspect the artifact's ELF magic, `go version -m` metadata, commit, and SHA. Upload only the artifact archive and remote deploy script.
4. Verify the uploaded archive SHA before extraction. Run the binary with `--help` before wrapping it into an image.
5. Use `scripts/deploy-binary.sh` on the host. It preserves the current image as a rollback tag, replaces `/new-api` in a stopped temporary container, commits a candidate image, smoke-tests it, recreates only the `new-api` service, and rolls back if local health fails.
6. Verify the image revision label, runtime binary SHA, container health/restart count, PostgreSQL health, Redis state, local `/api/status`, public `/api/status`, and the changed public route.
7. Update only `.deploy-commit`, `.deploy-release`, and any deliberately maintained source snapshot after success. Remove temporary archives/worktrees only after verification; retain the rollback image.

## Use the scripts

Run the local preflight without creating anything:

```powershell
$repoRoot = git rev-parse --show-toplevel
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" -ValidateOnly
```

Build an exact revision:

```powershell
$repoRoot = git rev-parse --show-toplevel
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" -Commit <verified-master-sha>
```

Display remote deploy arguments:

```bash
bash deploy-binary.sh --help
```

Prefer uploading the script as a file. If a multiline remote command is unavoidable, base64-encode it locally and decode it remotely to avoid PowerShell, SSH, and Bash quoting corruption.
