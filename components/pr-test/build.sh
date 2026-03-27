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

declare -A COMPONENTS=(
  [frontend]="context=components/frontend dockerfile=components/frontend/Dockerfile image=vteam_frontend"
  [backend]="context=components/backend dockerfile=components/backend/Dockerfile image=vteam_backend"
  [operator]="context=components/operator dockerfile=components/operator/Dockerfile image=vteam_operator"
  [ambient-runner]="context=components/runners dockerfile=components/runners/ambient-runner/Dockerfile image=vteam_claude_runner"
  [state-sync]="context=components/runners/state-sync dockerfile=components/runners/state-sync/Dockerfile image=vteam_state_sync"
  [public-api]="context=components/public-api dockerfile=components/public-api/Dockerfile image=vteam_public_api"
  [ambient-api-server]="context=components/ambient-api-server dockerfile=components/ambient-api-server/Dockerfile image=vteam_api_server"
)

COMPONENT_ORDER=(frontend backend operator ambient-runner state-sync public-api ambient-api-server)

echo "==> Building and pushing PR #${PR_NUMBER} images"
echo "    Tag:      ${IMAGE_TAG}"
echo "    Registry: ${REGISTRY}"
echo "    Platform: ${PLATFORM}"
echo ""

cd "$REPO_ROOT"

GIT_SHA=$(git rev-parse HEAD)

for name in "${COMPONENT_ORDER[@]}"; do
  eval "declare -A comp=(${COMPONENTS[$name]})"
  full_image="${REGISTRY}/${comp[image]}:${IMAGE_TAG}"

  echo "==> Building ${name} → ${full_image}"
  "$CONTAINER_ENGINE" build \
    --platform "$PLATFORM" \
    --build-arg "AMBIENT_VERSION=${GIT_SHA}" \
    -f "${comp[dockerfile]}" \
    -t "$full_image" \
    "${comp[context]}"

  echo "==> Pushing ${full_image}"
  "$CONTAINER_ENGINE" push "$full_image"

  echo ""
done

echo "==> All images pushed for PR #${PR_NUMBER}"
echo "    Image tag: ${IMAGE_TAG}"
echo "    Registry:  ${REGISTRY}"
