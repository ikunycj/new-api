---
name: deploy-new-api
description: Build, deploy, verify, diagnose, and roll back QuantumNous/new-api from the workstation to the selected SSH deployment host (`aliyun` for development, `alltokenapi` production, or `ikun.love` production), using a locally cross-compiled Linux binary and the existing Docker image. Use for new-api releases, `/opt/new-api`, PostgreSQL/Redis coexistence, artifact upload, failed remote Docker builds, production health checks, or rollback work.
---

# Deploy new-api

Deploy the latest commit on the branch assigned to the selected host without compiling on the target server. Reuse the existing runtime image and data services, then prove that the running binary matches the local artifact.

## Select the deployment host first

Before every deployment, confirm the target SSH alias and environment with the user. The supported choices are `aliyun` (development), `alltokenapi` (production), and `ikun.love` (production). Do not infer the target from the current branch, historical notes, DNS, or the last deployment. Set the selected SSH alias as `$deployHost` and use it consistently for all remote commands. The host table in [references/production-topology.md](references/production-topology.md) supplies the branch and remote overrides for the selected target.

- `aliyun`: development/test server. Follow the test-topology constraints in `AGENTS.md`: the remote service and the local development process intentionally use `NODE_TYPE=master` and share the Aliyun test PostgreSQL/Redis, which remain isolated from production.
- `alltokenapi`: production server. Deploy only the configured production branch (`master` unless the topology is deliberately changed), use the production authorization gate, verify live DNS, and run the rollback checks below.
- `ikun.love`: production server for `https://ikun.love`. Deploy only `origin/ikun.love`; use the `new-api:ikun` image tag and the `ikun-new-api*` Compose containers documented in the topology reference. This target also requires the production authorization gate, live DNS verification, and rollback checks below.

## Require the latest target branch

Deploy only the latest commit on the branch assigned to the selected host. Before building or uploading anything, fetch that branch and require all three SHAs to be identical:

- local `refs/heads/<branch>`
- fetched `refs/remotes/origin/<branch>`
- live `refs/heads/<branch>` returned by `git ls-remote origin`

Stop if the local branch is missing, any SHA differs, the remote query fails, or the worktree contains uncommitted changes that are intended for the release. Never deploy another branch, a tag, an arbitrary commit, or a locally ahead/behind target branch. Build the verified SHA from a detached clean worktree; do not require switching the user's current worktree to the release branch.

## Load the right context

- Read [references/deployment-playbook.md](references/deployment-playbook.md) before preparing or deploying a release.
- Read [references/production-topology.md](references/production-topology.md) before any server command. Treat its snapshot values as hints and verify live state.
- Read [references/incident-recovery.md](references/incident-recovery.md) when Docker returns EOF, the host is memory-constrained, the app restarts unexpectedly, or dependencies are not ready.

## Enforce the safety contract

1. Require the selected deployment host in the current conversation before any deployment. For `alltokenapi` and `ikun.love`, also require explicit authorization before a remote write, image retag, container recreation, or production restart. Read-only inspection and local builds are allowed after the host is selected.
2. Inspect repository instructions and `git status` first. Preserve unrelated dirty files; build only the verified latest commit on the selected target branch from a detached clean worktree.
3. Never print, upload, replace, or commit `.env` values. Never overwrite `/opt/new-api/.env`, `data`, or `logs`.
4. Never add or recreate PostgreSQL or Redis. The app must reuse the selected target's existing data containers and Docker network documented in the topology reference.
5. Never run a source or multi-stage Docker build on the production host. Its memory is insufficient and a failed build can OOM-kill Docker.
6. Keep a rollback image until the new release passes local and public checks.
7. Verify every destructive cleanup target is inside the expected temporary directory before removal.

## Execute the release

1. Fetch the selected branch, verify the local branch, `origin/<branch>`, and live remote branch resolve to the same SHA, and use only that SHA as the release commit. Stop on any mismatch.
2. Use `scripts/build-release.ps1` on Windows or `scripts/build-release.sh` on macOS/Linux to create a detached worktree, install the exact locked frontend dependencies in that worktree, build the frontend, cross-compile the Linux binary, and emit a SHA-256 manifest plus archive.
3. Inspect the artifact's ELF magic, `go version -m` metadata, commit, and SHA. Upload only the artifact archive and remote deploy script.
4. Verify the uploaded archive SHA before extraction. Run the binary with `--help` before wrapping it into an image.
5. Use `scripts/deploy-binary.sh` on the host. It preserves the current image as a rollback tag, replaces `/new-api` in a stopped temporary container, commits a candidate image, smoke-tests it, recreates only the `new-api` service, and rolls back if local health fails.
6. Verify the image revision label, runtime binary SHA, container health/restart count, PostgreSQL health, Redis state, local `/api/status`, public `/api/status`, and the changed public route.
7. Update only `.deploy-commit`, `.deploy-release`, and any deliberately maintained source snapshot after success. Remove temporary archives/worktrees only after verification; retain the rollback image.

## Use the scripts

Run the local preflight without creating anything (replace `master` with the branch assigned to the selected host):

```powershell
$repoRoot = git rev-parse --show-toplevel
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" -Branch master -ValidateOnly
```

Build an exact revision:

```powershell
$repoRoot = git rev-parse --show-toplevel
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" -Branch <branch> -Commit <verified-branch-sha>
```

On macOS/Linux, use the equivalent shell script:

```bash
bash .codex/skills/deploy-new-api/scripts/build-release.sh \
  --branch ikun.love --commit <verified-ikun.love-sha>
```

Display remote deploy arguments:

```bash
bash deploy-binary.sh --help
```

For `ikun.love`, pass the live topology explicitly:

```bash
bash deploy-binary.sh \
  --archive ./<archive-name> --archive-sha <archive-sha256> \
  --binary-sha <binary-sha256> --commit <full-commit-sha> \
  --release <short-sha>-v1 --image-tag new-api:ikun \
  --project-name ikun-new-api --network ikun-new-api-network \
  --container ikun-new-api --postgres ikun-new-api-postgres \
  --redis ikun-new-api-redis --public-url https://ikun.love/api/status
```

Prefer uploading the script as a file. If a multiline remote command is unavoidable, base64-encode it locally and decode it remotely to avoid PowerShell, SSH, and Bash quoting corruption.
