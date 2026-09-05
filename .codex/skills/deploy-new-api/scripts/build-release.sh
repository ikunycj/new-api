#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: build-release.sh [options]

Options:
  --repository PATH       Repository root (default: current repository)
  --branch NAME           Release branch (default: master)
  --commit SHA            Exact commit; defaults to the verified branch tip
  --output-directory PATH Artifact directory (default: ../new-api-releases)
  --keep-worktree         Keep the isolated build worktree
  --force-rebuild         Remove this commit's stale worktree/artifact first
  --validate-only         Run preflight without creating a worktree/artifact
  -h, --help              Show this help
EOF
}

REPOSITORY=''
BRANCH='master'
COMMIT=''
OUTPUT_DIRECTORY=''
KEEP_WORKTREE=0
FORCE_REBUILD=0
VALIDATE_ONLY=0

while (($#)); do
  case "$1" in
    --repository) REPOSITORY=${2:?missing value}; shift 2 ;;
    --branch) BRANCH=${2:?missing value}; shift 2 ;;
    --commit) COMMIT=${2:?missing value}; shift 2 ;;
    --output-directory) OUTPUT_DIRECTORY=${2:?missing value}; shift 2 ;;
    --keep-worktree) KEEP_WORKTREE=1; shift ;;
    --force-rebuild) FORCE_REBUILD=1; shift ;;
    --validate-only) VALIDATE_ONLY=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

for command_name in git bun go tar; do
  command -v "$command_name" >/dev/null || {
    echo "Required tool is not available: $command_name" >&2
    exit 1
  }
done

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print tolower($1)}'
  else
    shasum -a 256 "$1" | awk '{print tolower($1)}'
  fi
}

repo_input=${REPOSITORY:-.}
repo_root=$(cd "$repo_input" && pwd -P)
[[ -f "$repo_root/go.mod" ]] || {
  echo "Not a new-api repository: $repo_root" >&2
  exit 1
}

if [[ ! $BRANCH =~ ^[A-Za-z0-9][A-Za-z0-9._/-]*$ ||
      $BRANCH == *..* || $BRANCH == *//* || $BRANCH == */ ||
      $BRANCH == *. || $BRANCH == *.lock || $BRANCH == *@\{* ]]; then
  echo "Invalid release branch name: $BRANCH" >&2
  exit 2
fi

git -C "$repo_root" fetch origin "+refs/heads/$BRANCH:refs/remotes/origin/$BRANCH"
local_branch=$(git -C "$repo_root" rev-parse --verify "refs/heads/$BRANCH")
fetched_branch=$(git -C "$repo_root" rev-parse --verify "refs/remotes/origin/$BRANCH")
remote_branch_line=$(git -C "$repo_root" ls-remote origin "refs/heads/$BRANCH")
remote_branch=$(printf '%s\n' "$remote_branch_line" | awk -v ref="refs/heads/$BRANCH" '$2 == ref { print $1; exit }')
[[ $remote_branch =~ ^[0-9a-f]{40}$ ]] || {
  echo "Could not verify live origin/$BRANCH: $remote_branch_line" >&2
  exit 1
}
if [[ $local_branch != "$fetched_branch" || $local_branch != "$remote_branch" ]]; then
  echo "$BRANCH mismatch: local=$local_branch fetched=$fetched_branch remote=$remote_branch" >&2
  exit 1
fi

commit_spec=${COMMIT:-$remote_branch}
commit_sha=$(git -C "$repo_root" rev-parse --verify "${commit_spec}^{commit}")
[[ $commit_sha =~ ^[0-9a-f]{40}$ ]] || {
  echo "Could not resolve a single commit: $commit_sha" >&2
  exit 1
}
if [[ $commit_sha != "$remote_branch" ]]; then
  echo "Only the latest origin/$BRANCH may be built: requested=$commit_sha branch=$remote_branch" >&2
  exit 1
fi

dockerfile=$(git -C "$repo_root" show "$commit_sha:Dockerfile")
dockerfile_blob=$(git -C "$repo_root" rev-parse "$commit_sha:Dockerfile")
frontend_lock_blob=$(git -C "$repo_root" rev-parse "$commit_sha:web/bun.lock")
expected_go_version=$(printf '%s\n' "$dockerfile" | sed -nE 's/^FROM[[:space:]]+golang:([0-9.]+)-.*/\1/p' | head -n 1)
[[ $expected_go_version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo 'Could not determine the required Go version from Dockerfile.' >&2
  exit 1
}
go_experiment=$(printf '%s\n' "$dockerfile" | sed -nE 's/^ENV[[:space:]]+GOEXPERIMENT=([A-Za-z0-9,]+)[[:space:]]*$/\1/p' | head -n 1)
bun_version=$(bun --version)
[[ $bun_version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo "Could not parse Bun version: $bun_version" >&2
  exit 1
}

# GOTOOLCHAIN downloads the exact Dockerfile toolchain when the workstation has
# a different Go version. GOPROXY is configurable for restricted networks.
go_proxy=${GOPROXY:-https://goproxy.cn,direct}
go_version_output=$(GOTOOLCHAIN="go${expected_go_version}+auto" GOPROXY="$go_proxy" go version)
local_go_version=$(printf '%s\n' "$go_version_output" | sed -nE 's/^go version go([^ ]+).*/\1/p')
if [[ $local_go_version != "$expected_go_version" ]]; then
  echo "Go version mismatch: Dockerfile requires $expected_go_version, local tool is $go_version_output" >&2
  exit 1
fi

build_script_sha=$(sha256_file "$repo_root/.codex/skills/deploy-new-api/scripts/build-release.sh")
short_sha=${commit_sha:0:12}
release="${short_sha}-v1"
build_time=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
repo_parent=$(dirname "$repo_root")
worktree="$repo_parent/new-api-build-$commit_sha"
if [[ -n $OUTPUT_DIRECTORY ]]; then
  if [[ $OUTPUT_DIRECTORY = /* ]]; then
    output_root=$(cd "$(dirname "$OUTPUT_DIRECTORY")" && pwd -P)/$(basename "$OUTPUT_DIRECTORY")
  else
    output_root="$repo_root/$OUTPUT_DIRECTORY"
  fi
else
  output_root="$repo_parent/new-api-releases"
fi
artifact_directory="$output_root/$commit_sha"
binary_path="$artifact_directory/new-api"
manifest_path="$artifact_directory/manifest.json"
archive_path="$output_root/new-api-$commit_sha-linux-amd64.tar.gz"
partial_archive_path="$archive_path.partial-$$"

available_kb=$(df -Pk "$repo_root" | awk 'NR == 2 { print $4 }')
[[ $available_kb =~ ^[0-9]+$ ]] && ((available_kb >= 4 * 1024 * 1024)) || {
  echo "The build volume needs at least 4 GiB free; available: ${available_kb:-unknown} KiB" >&2
  exit 1
}

worktree_exists=0
[[ -e "$worktree" ]] && worktree_exists=1
artifact_exists=0
[[ -e "$artifact_directory" || -e "$archive_path" ]] && artifact_exists=1

if ((VALIDATE_ONLY)); then
  printf 'Repository=%s\nBranch=%s\nCommit=%s\nWorktree=%s\nWorktreeExists=%s\nDependencyMode=isolated bun install --frozen-lockfile\nOutputDirectory=%s\nArtifactExists=%s\nFreeGiB=%.2f\nGoVersion=%s\nGoExperiment=%s\nBunVersion=%s\nFrontendLockBlob=%s\nBuildScriptSha256=%s\nStatus=preflight-passed\n' \
    "$repo_root" "$BRANCH" "$commit_sha" "$worktree" "$worktree_exists" "$output_root" "$artifact_exists" \
    "$(awk -v kb="$available_kb" 'BEGIN { printf "%.2f", kb / 1024 / 1024 }')" "$local_go_version" "$go_experiment" "$bun_version" "$frontend_lock_blob" "$build_script_sha"
  exit 0
fi

case "$artifact_directory" in
  "$output_root"/*) ;;
  *) echo "Artifact path escapes output directory: $artifact_directory" >&2; exit 1 ;;
esac
case "$archive_path" in
  "$output_root"/*) ;;
  *) echo "Archive path escapes output directory: $archive_path" >&2; exit 1 ;;
esac

remove_stale_worktree() {
  [[ "$worktree" == "$repo_parent/new-api-build-$commit_sha" ]] || {
    echo "Refusing to clean unexpected worktree path: $worktree" >&2
    exit 1
  }
  if git -C "$repo_root" worktree list --porcelain | awk -v path="$worktree" '$1 == "worktree" && $2 == path { found = 1 } END { exit !found }'; then
    git -C "$repo_root" worktree remove --force "$worktree"
  elif [[ -e "$worktree" ]]; then
    rm -rf -- "$worktree"
  fi
  git -C "$repo_root" worktree prune
}

if ((worktree_exists)); then
  if ((FORCE_REBUILD)); then
    remove_stale_worktree
  else
    echo "Build worktree already exists: $worktree. Pass --force-rebuild to remove this exact commit's stale worktree." >&2
    exit 1
  fi
fi
if ((artifact_exists)) && (( ! FORCE_REBUILD )); then
  echo "Artifact already exists for $commit_sha. Pass --force-rebuild to remove it before rebuilding." >&2
  exit 1
fi
if ((FORCE_REBUILD)); then
  [[ "$artifact_directory" == "$output_root/$commit_sha" ]] || { echo 'Refusing unexpected artifact path' >&2; exit 1; }
  [[ "$archive_path" == "$output_root/new-api-$commit_sha-linux-amd64.tar.gz" ]] || { echo 'Refusing unexpected archive path' >&2; exit 1; }
  [[ ! -L "$artifact_directory" ]] && [[ -d "$artifact_directory" ]] && rm -rf -- "$artifact_directory"
  [[ ! -L "$archive_path" ]] && [[ -f "$archive_path" ]] && rm -f -- "$archive_path"
fi

mkdir -p "$output_root"
worktree_created=0
cleanup() {
  rm -f -- "$partial_archive_path" 2>/dev/null || true
  if ((worktree_created)) && (( ! KEEP_WORKTREE )); then
    rm -rf -- "$worktree/web/node_modules" 2>/dev/null || true
    git -C "$repo_root" worktree remove --force "$worktree" >/dev/null 2>&1 || true
    git -C "$repo_root" worktree prune >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

git -C "$repo_root" worktree add --detach "$worktree" "$commit_sha"
worktree_created=1
bun install --frozen-lockfile --cwd "$worktree/web"

version=''
if [[ -f "$worktree/VERSION" ]]; then
  version=$(sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' "$worktree/VERSION")
fi

(cd "$worktree/web/default" && DISABLE_ESLINT_PLUGIN=true VITE_REACT_APP_VERSION="$version" bun run build)

mkdir -p "$artifact_directory"
ldflags="-s -w -X github.com/QuantumNous/new-api/common.Version=$version -X github.com/QuantumNous/new-api/common.BuildCommit=$commit_sha -X github.com/QuantumNous/new-api/common.BuildRelease=$release -X github.com/QuantumNous/new-api/common.BuildTime=$build_time"
(cd "$worktree" && \
  GOTOOLCHAIN="go${expected_go_version}+auto" GOPROXY="$go_proxy" \
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOEXPERIMENT="$go_experiment" \
  go build -buildvcs=false -trimpath -ldflags "$ldflags" -o "$binary_path" .)

elf_magic=$(od -An -tx1 -N4 "$binary_path" | tr -d ' \n')
[[ $elf_magic == 7f454c46 ]] || { echo "Built artifact is not an ELF binary: $elf_magic" >&2; exit 1; }
GOTOOLCHAIN="go${expected_go_version}+auto" GOPROXY="$go_proxy" go version -m "$binary_path"
binary_sha=$(sha256_file "$binary_path")
version_json=$(printf '%s' "$version" | sed 's/\\/\\\\/g; s/"/\\"/g')
go_experiment_json=$(printf '%s' "$go_experiment" | sed 's/\\/\\\\/g; s/"/\\"/g')
build_time_json=$(printf '%s' "$build_time" | sed 's/\\/\\\\/g; s/"/\\"/g')
cat > "$manifest_path" <<EOF
{
  "commit": "$commit_sha",
  "short_commit": "$short_sha",
  "version": "$version_json",
  "release": "$release",
  "build_time": "$build_time_json",
  "target": "linux/amd64",
  "go_version": "$local_go_version",
  "goexperiment": "$go_experiment_json",
  "bun_version": "$bun_version",
  "frontend_lock_blob": "$frontend_lock_blob",
  "dockerfile_blob": "$dockerfile_blob",
  "build_script_sha256": "$build_script_sha",
  "binary_sha256": "$binary_sha",
  "built_at_utc": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
}
EOF

tar -czf "$partial_archive_path" -C "$artifact_directory" new-api manifest.json
mv -f -- "$partial_archive_path" "$archive_path"
archive_sha=$(sha256_file "$archive_path")
printf 'Branch=%s\nCommit=%s\nBinary=%s\nBinarySha256=%s\nArchive=%s\nArchiveSha256=%s\nManifest=%s\nBunVersion=%s\nBuildScriptSha256=%s\n' \
  "$BRANCH" "$commit_sha" "$binary_path" "$binary_sha" "$archive_path" "$archive_sha" "$manifest_path" "$bun_version" "$build_script_sha"
