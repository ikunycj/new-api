---
name: deploy-new-api
description: Build, deploy, verify, diagnose, and roll back QuantumNous/new-api from the Windows workstation to the selected SSH deployment host (`aliyun`, `alltokenapi`, or `ikun.love`), using a locally cross-compiled Linux binary and Docker runtime/data services. Use for new-api releases, `/opt/new-api`, 1Panel PostgreSQL/Redis coexistence, the complete isolated `ikun.love` Docker stack, first-install bootstrap, artifact upload, failed remote Docker builds, production health checks, or rollback work.
---

# Deploy new-api

Deploy the latest verified commit for the selected release ref without compiling on the target server. Reuse the existing runtime/data services for the legacy targets; for `ikun.love`, bootstrap its own PostgreSQL and Redis containers before switching the application image, then prove that the running binary matches the local artifact.

## Select the deployment host first

Before every deployment, ask the user explicitly: `部署到 aliyun（开发）、alltokenapi（生产）还是 ikun.love（生产）？` Do not infer the target from the current branch, historical notes, DNS, or the last deployment. Set the selected SSH alias as `$deployHost` and use it consistently for all remote commands.

- `aliyun`: development/test server. Follow the test-topology constraints in `AGENTS.md`: the remote service and the local development process intentionally use `NODE_TYPE=master` and share the Aliyun test PostgreSQL/Redis, which remain isolated from production.
- `alltokenapi`: production server. This is the current production deployment host and must use the production authorization gate, live DNS verification, and rollback checks below.
- `ikun.love`: production server at SSH alias `ikun.love`, with public origin `https://ikun.love`. Use the target materials in `deploy/ikun.love/`, release ref `ikun.love`, live DNS verification, and the production authorization gate. This target is a fresh host with a complete isolated Docker project containing `ikun-new-api`, `ikun-new-api-postgres`, and `ikun-new-api-redis` on a private network with dedicated volumes. The legacy Sub2API service remains on SSH alias `ikun.love-sub2api`; verify it read-only before and after, and never stop or reconfigure it, its 1Panel containers, database, Redis db0, OpenResty, or DNS. Do not reuse `alltokenapi`'s public URL, `.env`, database/Redis names, OpenResty configuration, or deployment snapshot.

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
3. Keep each target's protected source env at `deploy/<ssh-alias>/.env`. Never print, diff, or commit it. Generate it with `scripts/prepare-deployment-env.ps1`, require an administrator-approved SHA-256, and upload it only for that target's first install. Never overwrite a different `/opt/new-api/.env`, `data`, or `logs`.
4. For `aliyun` and `alltokenapi`, never add or recreate PostgreSQL or Redis containers: reuse their verified existing services and network. For `ikun.love`, create only the dedicated PostgreSQL/Redis containers, network, and volumes defined in `deploy/ikun.love/docker-compose.1panel.yml`; never attach that stack to `1panel-network` or touch the existing 1Panel services.
5. Never run a source or multi-stage Docker build on the production host. Its memory is insufficient and a failed build can OOM-kill Docker.
6. Keep a rollback image until the new release passes local checks and the target's existing public service has been proven unaffected. A loopback-only `ikun.love` install intentionally has no new-api public check.
7. Verify every destructive cleanup target is inside the expected temporary directory before removal.

## Execute the release

1. Fetch the selected release ref, verify the local ref, fetched tracking ref, and live remote ref resolve to the same SHA, and use only that SHA as the release commit. For `ikun.love`, select `ikun.love` and the isolated target contract in `deploy/ikun.love/`; do not change the existing public root route. Stop on any mismatch.
2. Use `scripts/build-release.ps1 -DeploymentAlias <ssh-alias>` to create a same-drive detached worktree, install the exact locked frontend dependencies, build the frontend, and cross-compile the Linux binary. Keep the manifest, binary, and archive under `deploy/<ssh-alias>/artifacts/`; never use a shared release directory across targets.
3. Inspect the artifact's ELF magic, `go version -m` metadata, commit, and SHA. Upload only the artifact archive and remote deploy script.
4. Before a first install, run `prepare-deployment-env.ps1`; if it creates `deploy/<ssh-alias>/.env`, stop for administrator review. Approve the exact SHA with `-ApproveSha256`, rerun, and require `ReadyForUpload=True`. Stage that reviewed file in a mode-700 directory, install it as `/opt/new-api/.env` mode 600 only when the destination is absent, and compare local/remote SHA. For `ikun.love`, run `bootstrap-config.sh --expected-env-sha <approved-sha>` to start the dedicated PostgreSQL/Redis services and create `new_api` owned by restricted role `new_api_app`; then seed the runtime image. Never generate secrets on the server.
5. Use `scripts/deploy-binary.sh` on the host. It preserves the current target image as a rollback tag, replaces `/new-api` in a stopped temporary container, commits a candidate image, smoke-tests it, recreates only the `new-api` service, and rolls back if local health fails.
6. Verify the image revision label, runtime binary SHA, approved runtime env SHA, application/dependency health, dedicated network/volumes, restricted PostgreSQL role/database, and local `/api/status`. For `ikun.love`, verify the legacy service/8080/public health separately through `ssh ikun.love-sub2api`; do not require a new-api public URL for the isolated install.
7. Update only `.deploy-commit`, `.deploy-release`, and any deliberately maintained source snapshot after success. Remove temporary archives/worktrees only after verification; retain the rollback image.

## Use the scripts

Run the local preflight without creating anything:

```powershell
$repoRoot = git rev-parse --show-toplevel
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" `
  -DeploymentAlias <ssh-alias> -ReleaseRef master -ValidateOnly
```

Build an exact revision:

```powershell
$repoRoot = git rev-parse --show-toplevel
& "$repoRoot\.codex\skills\deploy-new-api\scripts\build-release.ps1" `
  -DeploymentAlias <ssh-alias> `
  -ReleaseRef <release-ref> `
  -Commit <verified-release-sha>
```

Prepare and approve the target env without displaying values:

```powershell
$envState = & "$repoRoot\.codex\skills\deploy-new-api\scripts\prepare-deployment-env.ps1" `
  -DeploymentAlias <ssh-alias>
# Administrator reviews deploy/<ssh-alias>/.env, then explicitly approves its SHA:
& "$repoRoot\.codex\skills\deploy-new-api\scripts\prepare-deployment-env.ps1" `
  -DeploymentAlias <ssh-alias> -ApproveSha256 $envState.Sha256
```

Display remote deploy arguments:

```bash
bash deploy-binary.sh --help
```

Prefer uploading the script as a file. If a multiline remote command is unavoidable, base64-encode it locally and decode it remotely to avoid PowerShell, SSH, and Bash quoting corruption.
