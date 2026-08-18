#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: bootstrap-image.sh [--base-image IMAGE] [--image-tag TAG]

Seeds the target-only new-api image tag from the official runtime image.
The release deploy script then replaces /new-api with the locally built,
SHA-verified binary. This script does not start or restart any container.
EOF
}

BASE_IMAGE='calciumion/new-api:latest'
IMAGE_TAG='new-api:ikun'

while (($#)); do
  case "$1" in
    --base-image) BASE_IMAGE=${2:?missing value}; shift 2 ;;
    --image-tag) IMAGE_TAG=${2:?missing value}; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if docker info >/dev/null 2>&1; then
  DOCKER=(docker)
elif sudo -n docker info >/dev/null 2>&1; then
  DOCKER=(sudo -n docker)
else
  echo 'Docker is unavailable for the current user and passwordless sudo.' >&2
  exit 1
fi

if "${DOCKER[@]}" image inspect "$IMAGE_TAG" >/dev/null 2>&1; then
  echo "image_tag=$IMAGE_TAG"
  echo 'status=already-present'
  exit 0
fi

"${DOCKER[@]}" pull "$BASE_IMAGE"
base_id=$("${DOCKER[@]}" image inspect "$BASE_IMAGE" --format '{{.Id}}')
"${DOCKER[@]}" tag "$BASE_IMAGE" "$IMAGE_TAG"
tag_id=$("${DOCKER[@]}" image inspect "$IMAGE_TAG" --format '{{.Id}}')
[[ $base_id == "$tag_id" ]] || { echo 'Seed image ID mismatch' >&2; exit 1; }

echo "base_image=$BASE_IMAGE"
echo "base_image_id=$base_id"
echo "image_tag=$IMAGE_TAG"
echo 'status=seeded'
