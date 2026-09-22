---
name: ikun.love-rebase
description: Rebase new-api's ikun.love branch onto a user-selected remote base branch, keeping exactly one ikun.love branding UI commit at the tip. Use for updating this branding branch or repeating the rebase originally based on commit 1e073c613. The base branch must be selected for each rebase; publishing or updating this skill is a separate operation.
---

# ikun.love Rebase

## Required Result

The checked-out branch must be `ikun.love`, with this history:

```text
<new SHA> feat(web): apply ikun.love branding   (HEAD -> ikun.love)
<fetched SHA>                                 (origin/<selected-base-branch>)
```

`1e073c613ce6e2cbd9f5af715f416fc618d5c572` identifies the original branding commit. Its SHA changes after rebasing. Reuse its current rebased version at the tip of `ikun.love`, preserving previously reviewed conflict resolutions. Do not repeatedly cherry-pick the original SHA or select a same-named commit from arbitrary history.

Use the freshly fetched **remote branch selected for this invocation** as the base. The base is not a skill default: choose it from the current release plan every time, record the exact remote SHA, and pass the branch explicitly to the script. A local feature branch may have divergent history or contain old branding; do not assume its tip is the desired base. Leave its branch pointer intact during this rebase.

Maintain this skill in `.codex/skills/ikun.love-rebase/` on the selected base branch or on the shared branch from which that base is maintained. It is inherited by `ikun.love` through the base history, so it does not add a second branch-specific commit. Keep the explicitly requested name `ikun.love-rebase`, including the dot.

Publishing this skill is a separate operation: integrate the latest remote base, commit the skill on the selected maintenance branch, and push that branch. Do not run this rebase or push `ikun.love` merely because the user is committing or publishing the skill.

## Run

Work in the user's new-api checkout. Invocation of the rebase authorizes the local branch switch and rebase. Inspect branch status, then run from the repository root:

```bash
# Choose the base for this rebase explicitly; this example is not a default.
base_branch=feature/new-common-api
bash -c "$(cat .codex/skills/ikun.love-rebase/scripts/rebase.sh)" \
  ikun.love-rebase "$PWD" "$base_branch"
```

Read the whole script into Bash before it switches branches: an older `ikun.love` may not yet contain this skill. The command substitution reads trusted repository code; do not substitute untrusted text. When the skill was loaded from a different checkout, use that loaded skill's absolute script path and pass the intended checkout as the final argument.

The script fetches `origin/<base-branch>`, checks that the current branding tip is a single UI commit based on a known feature revision, switches branches, creates a backup ref when needed, and runs:

```bash
git rebase --no-rebase-merges --reapply-cherry-picks --empty=stop \
  --onto "$base" "$branding_parent"
```

The explicit parent boundary selects exactly one commit, even when the upstream history was rewritten. Repeating the operation with an unchanged base is a no-op. The script prints the selected base branch, pinned base SHA, and backup ref before rebasing. Omitting the base branch is an error; the script never chooses one implicitly.

Do not auto-stash a dirty working tree, overwrite `.env`, reset the local feature branch, or change running services. For a failed preflight, report the specific condition. If additional commits or an unknown parent are found, inspect their contents before deciding how to retain that work; never silently discard them to force the one-commit shape.

## Conflicts

Continue the authorized rebase after inspecting and resolving its conflicts. Do not rerun the script while a rebase is active. Keep the backup ref and use the pinned base printed by the script, rather than fetching another base mid-rebase.

- Preserve upstream features together with the branding delta. Do not resolve an entire text file with `--ours` or `--theirs`: during rebase, ours is the upstream tree and theirs is the replayed commit's tree.
- In `web/default/scripts/add-missing-keys.mjs`, retain all upstream translation entries and the branding additions. The original delta adds `Connected to ikun.love` for all supported locales and retires `Connected to AllTokenAPI`.
- If locale JSON also conflicts, first confirm that its intended delta is only those two keys. Use the pinned upstream locale snapshots as the regeneration input, apply the resolved `add-missing-keys.mjs`, and run `bun run i18n:sync`. Compare parsed dictionaries against upstream to verify that no unrelated key/value was changed or lost; do not transplant the old locale files.
- Preserve upstream CSS variables and rules added since the branding commit, while retaining its logo, theme palette, public button styles, dashboard highlight, docs base URL, and client-preview branding.
- Read the repository's current frontend conventions and applicable UI/i18n skills before editing those files. Stage only resolved files and use `GIT_EDITOR=true git rebase --continue`. Resolution belongs in the same branding commit, not a follow-up commit.
- If the commit becomes empty, inspect whether upstream already contains the branding. Report that condition instead of manufacturing an empty commit or skipping it and claiming there is one UI delta.

For a conflict that cannot be resolved without a product decision, retain the rebase and backup ref and describe the unresolved choice. Do not silently abort or force a resolution.

## Verify And Report

Use the explicitly selected and pinned base SHA for these checks, including after manually continuing a rebase:

```bash
git branch --show-current
git rev-parse HEAD^
git rev-list --count "$base..HEAD"
git log -1 --format=%s
git diff --stat "$base" HEAD
git diff --check "$base" HEAD
git status --short
```

Require `ikun.love`, parent equal to the base SHA, count `1`, exact subject `feat(web): apply ikun.love branding`, a nonempty branding-only frontend delta, and a clean worktree. Inspect the final delta against the source commit; a one-commit count alone does not prove its contents are correct. A changed patch ID may be legitimate after adapting upstream conflicts.

For a clean replay, verify Git invariants and diff integrity. For manual conflict edits, run the relevant syntax, locale, type, lint, or build checks required by the repository. Do not start services just to validate a Git operation.

Report the base SHA, new branding SHA, one-commit result, and whether anything was pushed. Rebase does not imply a push. When the current request explicitly targets pushing the branding branch, inspect the current remote `ikun.love` first, reconcile any new remote work, and push only `ikun.love` with an explicit `--force-with-lease=refs/heads/ikun.love:<verified-remote-SHA>`. If the lease fails, inspect again; never retry with an unconditional force push.
