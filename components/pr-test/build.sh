#!/usr/bin/env bash
set -euo pipefail

PR_URL="${1:-}"
REGISTRY="${REGISTRY:-quay.io/ambient_code}"
PLATFORM="${PLATFORM:-linux/amd64}"
CONTAINER_ENGINE="${CONTAINER_ENGINE:-docker}"

usage() {
  echo "Usage: $0 <pr-url>"
  echo "  pr-url:  e.g. https://github.com/ambient-code/platform/pull/1005"
  echo ""
  echo "Optional environment variables:"
  echo "  REGISTRY          Registry prefix (default: quay.io/ambient_code)"
  echo "  PLATFORM          Build platform (default: linux/amd64)"
  echo "  CONTAINER_ENGINE  docker or podman (default: docker)"
  exit 1
}

[[ -z "$PR_URL" ]] && usage

PR_NUMBER=$(echo "$PR_URL" | grep -oE '[0-9]+$')
if [[ -z "$PR_NUMBER" ]]; then
  echo "ERROR: Could not extract PR number from URL: $PR_URL"
  exit 1
fi

IMAGE_TAG="pr-${PR_NUMBER}-amd64"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "==> Building and pushing PR #${PR_NUMBER} images (mpp-openshift target)"
echo "    Tag:      ${IMAGE_TAG}"
echo "    Registry: ${REGISTRY}"
echo "    Platform: ${PLATFORM}"
echo ""

cd "$REPO_ROOT"

GIT_SHA=$(git rev-parse HEAD)

build_push() {
  local name="$1" context="$2" dockerfile="$3" image="$4"
  local full_image="${REGISTRY}/${image}:${IMAGE_TAG}"
  echo "==> Building ${name} → ${full_image}"
  "$CONTAINER_ENGINE" build \
    --platform "$PLATFORM" \
    --build-arg "AMBIENT_VERSION=${GIT_SHA}" \
    -f "$dockerfile" \
    -t "$full_image" \
    "$context"
  echo "==> Pushing ${full_image}"
  "$CONTAINER_ENGINE" push "$full_image"
  echo ""
}

build_push ambient-api-server \
  components/ambient-api-server \
  components/ambient-api-server/Dockerfile \
  vteam_api_server

build_push ambient-control-plane \
  components \
  components/ambient-control-plane/Dockerfile \
  vteam_control_plane

build_push ambient-runner \
  components/runners \
  components/runners/ambient-runner/Dockerfile \
  vteam_claude_runner

echo "==> All images pushed for PR #${PR_NUMBER}"
echo "    Image tag: ${IMAGE_TAG}"
echo "    Registry:  ${REGISTRY}"
