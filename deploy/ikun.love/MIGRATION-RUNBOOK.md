# Sub2API to new-api migration runbook

This runbook covers the isolated migration from `ikun.love-sub2api` to
`ikun.love`. The source service must remain online throughout the operation.
The source and target use separate PostgreSQL and Redis services.

## Tool and safety contract

- Tool: `migrate-sub2api-users.ps1` in this directory.
- The script reads a repeatable source snapshot, builds a deterministic plan,
  and writes each accepted user in a transaction.
- Reports contain masked email references and fingerprints only. Do not commit
  snapshots, reports, `.env` files, passwords, or API keys.
- `-FastApply` is allowed only after a clean full dry-run and verified database
  backups. It does not mean "skip validation"; the source snapshot and target
  preflight still run.

## Migration rules

1. Only live users with role `user` and status `active` are imported.
2. Empty usernames become the normalized email. Usernames are not treated as
   identity keys; normalized email is the identity field.
3. Group mapping is:

   | Source group | Target group |
   | --- | --- |
   | OpenAI官方转发 | ChatGPT官转 |
   | ChatGPT 生图专用 | 生图 |
   | ChatGPT Plus | ChatGPT Plus |
   | ChatGPT Pro | ChatGPT Pro |
   | Claude | Claude Kiro |
   | Claude Max 20x | Claude Max满血 |
   | ChatGPT 羊毛福利 | 羊毛福利 |

   Any unmapped or unavailable group falls back to `ChatGPT Plus`.
4. Wallet balance and active subscription remaining value are converted with
   `floor(USD / 5 * 500000)`. Negative wallet values are clamped to zero and
   recorded as an audit warning.
5. One leading lowercase `sk-` is removed from an API key before storage. The
   original case and body are otherwise preserved. Key quota and used quota are
   converted independently; key quota is never added to the user wallet.
6. Expired, inactive, exhausted, or policy-blocked keys are imported disabled.
   Source rate limits, usage windows, and blacklists that have no target
   enforcement equivalent are written to
   `sub2api_api_key_migration_policies` and the target key is disabled.
7. Existing target email accounts are reconciled or merged without adding the
   source wallet twice. Target passwords, history, and existing keys are
   preserved; the source mapping table is the idempotency ledger.

## Procedure

Run these commands from the repository worktree. Use a new report filename for
each run.

```powershell
$script = "F:\Project\My-Project\new-api-migration-bulk\deploy\ikun.love\migrate-sub2api-users.ps1"
$report = "F:\Project\My-Project\new-api-migration-bulk\deploy\ikun.love\migration-reports\preflight.json"
pwsh -NoProfile -File $script -All -ReportPath $report
```

The dry-run must have no blocker result. For a completed batch, the expected
idempotent result is `already_migrated` for every source row.

After backup review and operator approval:

```powershell
$report = "F:\Project\My-Project\new-api-migration-bulk\deploy\ikun.love\migration-reports\apply.json"
pwsh -NoProfile -File $script -All -Apply -FastApply -ReportPath $report
```

Run the full dry-run again after apply. Never use `FLUSHALL` for cache cleanup.
Only remove the application cache namespaces (`user:*` and `token:v2:*`) when
the target application is ready to repopulate them from PostgreSQL.

## Acceptance checks

- Source active ordinary-user count equals migration mapping count.
- Mapping source IDs and target IDs are both unique.
- Every source active-user key has exactly one normalized target key.
- Clean imports retain the source bcrypt hash; merge/reconcile modes preserve
  the existing target password by design.
- Target token quota, used quota, unlimited flag, expiry, and status match the
  conversion rules above.
- Every policy-blocked key has an audit row and a non-enabled target status.
- Target PostgreSQL and Redis are healthy, target `/api/status` is successful,
  and `sub2api.service` remains active on the source host.

The migration mapping and policy tables are audit data. Do not delete them as
part of routine cleanup; they prevent duplicate wallet grants on reruns.
