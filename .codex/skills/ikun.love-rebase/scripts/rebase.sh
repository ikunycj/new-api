#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

[[ $# -le 1 ]] || fail 'Usage: bash rebase.sh [new-api-checkout]'
cd "${1:-.}"
repo=$(git rev-parse --show-toplevel)
cd "$repo"

target='ikun.love'
upstream='origin/feature/new-channel-and-group'
subject='feat(web): apply ikun.love branding'

for operation in rebase-merge rebase-apply MERGE_HEAD CHERRY_PICK_HEAD REVERT_HEAD sequencer; do
  [[ ! -e "$(git rev-parse --git-path "$operation")" ]] ||
    fail "An unfinished Git operation exists: $operation"
done
[[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]] ||
  fail 'Working tree is not clean; preserve the pending changes before rebasing.'
source=$(git rev-parse --verify "refs/heads/$target^{commit}") ||
  fail 'Local ikun.love is missing; inspect origin/ikun.love before creating it.'
[[ "$(git show -s --format=%s "$source")" == "$subject" ]] ||
  fail 'ikun.love does not end in the branding commit; inspect additional work first.'
parent=$(git show -s --format=%P "$source")
[[ -n "$parent" && "$parent" != *' '* ]] ||
  fail 'The branding tip must have exactly one parent.'
git diff --quiet "$parent" "$source" && fail 'The branding commit has no UI changes.'

# Limit the automatic replay to the original branding surfaces.
while IFS= read -r -d '' file; do
  case "$file" in
    web/default/public/logo.png|web/default/public/logo-28.webp|web/default/public/logo-56.webp|\
    web/default/scripts/add-missing-keys.mjs|\
    web/default/src/components/layout/components/public-header.tsx|\
    web/default/src/features/dashboard/components/overview/overview-dashboard.tsx|\
    web/default/src/features/docs/hooks/use-docs-base-url.ts|\
    web/default/src/features/home/components/landing/ai-clients-section.tsx|\
    web/default/src/i18n/locales/*.json|web/default/src/styles/theme.css) ;;
    *) fail "The branding tip changes an unexpected path: $file" ;;
  esac
done < <(git diff --no-renames --name-only -z "$parent" "$source")

# Remember the previous upstream so a freshly rewritten remote is supported.
previous_base=$(git rev-parse --verify "refs/remotes/$upstream^{commit}" 2>/dev/null || true)
git -c http.lowSpeedLimit=1 -c http.lowSpeedTime=30 fetch origin \
  '+refs/heads/feature/new-channel-and-group:refs/remotes/origin/feature/new-channel-and-group'
base=$(git rev-parse --verify "refs/remotes/$upstream^{commit}")
if ! git merge-base --is-ancestor "$parent" "$base"; then
  if [[ -z "$previous_base" ]] || ! git merge-base --is-ancestor "$parent" "$previous_base"; then
    fail 'The branding parent is not a known upstream revision; inspect intervening commits before replaying.'
  fi
fi
[[ "$(git rev-parse "refs/heads/$target")" == "$source" ]] ||
  fail 'ikun.love changed during preflight; inspect the new branch state.'

git switch "$target"
printf 'Base: %s\nBranding source: %s\n' "$base" "$source"
if [[ "$parent" != "$base" ]]; then
  backup="refs/backup/ikun.love-rebase/$source"
  git update-ref "$backup" "$source"
  printf 'Backup: %s\n' "$backup"
  if ! GIT_EDITOR=true git -c rebase.autoStash=false -c rebase.updateRefs=false \
    rebase --no-rebase-merges --reapply-cherry-picks --empty=stop --onto "$base" "$parent"; then
    printf 'Rebase paused. Inspect conflicts or an empty commit and follow SKILL.md.\n' >&2
    exit 1
  fi
fi

[[ "$(git branch --show-current)" == "$target" ]] || fail 'Unexpected checked-out branch.'
[[ "$(git show -s --format=%P HEAD)" == "$base" ]] || fail 'Branding parent does not match the fetched base.'
[[ "$(git rev-list --count "$base..HEAD")" == 1 ]] || fail 'Expected exactly one commit above the base.'
[[ "$(git show -s --format=%s HEAD)" == "$subject" ]] || fail 'Unexpected tip commit subject.'
git diff --quiet "$base" HEAD && fail 'No branding delta remains.'
git diff --check "$base" HEAD
[[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]] || fail 'Working tree is not clean after rebase.'
printf 'Verified: %s + one branding commit %s\n' "$base" "$(git rev-parse HEAD)"
printf 'No push performed.\n'
