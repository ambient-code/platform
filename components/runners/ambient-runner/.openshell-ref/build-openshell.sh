#!/usr/bin/env bash
set -euo pipefail

REGISTRY="${REGISTRY:-quay.io/ambient_code}"
IMAGE_TAG="${IMAGE_TAG:-openshell-v1}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"

echo "==> Building OpenShell-enabled images"
echo "    Registry: ${REGISTRY}"
echo "    Tag:      ${IMAGE_TAG}"
echo ""

echo "==> Step 1: Building runner image"
podman build \
  -t "${REGISTRY}/vteam_claude_runner:${IMAGE_TAG}" \
  -f "${REPO_ROOT}/components/runners/ambient-runner/Dockerfile" \
  "${REPO_ROOT}/components/runners/ambient-runner"

echo "==> Step 2: Building control-plane image"
podman build \
  -t "${REGISTRY}/vteam_control_plane:${IMAGE_TAG}" \
  -f "${REPO_ROOT}/components/ambient-control-plane/Dockerfile" \
  "${REPO_ROOT}/components/ambient-control-plane"

echo "==> Step 3: Pushing images"
podman push "${REGISTRY}/vteam_claude_runner:${IMAGE_TAG}"
podman push "${REGISTRY}/vteam_control_plane:${IMAGE_TAG}"

echo ""
echo "==> Images built and pushed:"
echo "    ${REGISTRY}/vteam_claude_runner:${IMAGE_TAG}"
echo "    ${REGISTRY}/vteam_control_plane:${IMAGE_TAG}"
echo ""
echo "    Note: api-server image is NOT rebuilt — use an existing tag"
echo "    or set IMAGE_TAG to an existing api-server tag."
