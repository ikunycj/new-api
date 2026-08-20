# Deployment Artifacts

Deployment state is scoped by SSH alias. Each target keeps its private
environment and release outputs under `deploy/<ssh-alias>/`:

- `deploy/<ssh-alias>/.env` is the target's protected environment source of
  truth and is ignored by Git. Generate it locally, require administrator
  SHA-256 approval, and never print or diff its values.
- `deploy/<ssh-alias>/artifacts/` contains the binary, manifest, and archive
  for that target's verified commit. These files are ignored by Git.
- Keep `SESSION_SECRET` and `CRYPTO_SECRET` stable for the lifetime of that
  target; never copy an environment file between aliases.
- For Docker targets, the container connects to the target's dedicated
  PostgreSQL and Redis service names rather than a host-loopback service.
- `deploy/ikun.love/` is an isolated contract and must not reuse the
  `alltokenapi` URL, environment, database, Redis, or container snapshot.

Use the alias-aware release script from the repository root:

```powershell
& ".codex/skills/deploy-new-api/scripts/build-release.ps1" `
  -DeploymentAlias <ssh-alias> -ReleaseRef <release-ref> -Commit <verified-sha>
```

Before upload, run `prepare-deployment-env.ps1 -DeploymentAlias <ssh-alias>`.
If it creates the target `.env`, stop for administrator review; approve its
exact SHA explicitly before staging. Never use the legacy shared `deploy/.env`
for a deployment.
