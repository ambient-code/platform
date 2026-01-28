#!/bin/bash
set -euo pipefail

echo "======================================"
echo "Loading images into kind cluster"
echo "======================================"

# Detect container runtime
CONTAINER_ENGINE="${CONTAINER_ENGINE:-}"

if [ -z "$CONTAINER_ENGINE" ]; then
  if command -v docker &> /dev/null && docker ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &> /dev/null && podman ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="podman"
  else
    echo "❌ No container engine found"
    exit 1
  fi
fi

echo "Using container runtime: $CONTAINER_ENGINE"

# Set KIND_EXPERIMENTAL_PROVIDER if using Podman
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  export KIND_EXPERIMENTAL_PROVIDER=podman
fi

# Check if kind cluster exists
if ! kind get clusters 2>/dev/null | grep -q "^ambient-local$"; then
  echo "❌ Kind cluster 'ambient-local' not found"
  echo "   Run './scripts/setup-kind.sh' first"
  exit 1
fi

# Detect expected architecture based on host
case "$(uname -m)" in
  arm64|aarch64) EXPECTED_ARCH="arm64" ;;
  x86_64|amd64)  EXPECTED_ARCH="amd64" ;;
  *) EXPECTED_ARCH="amd64" ;;
esac

# Images to load
IMAGES=(
  "vteam_backend:latest"
  "vteam_frontend:latest"
  "vteam_operator:latest"
  "vteam_claude_runner:latest"
  "vteam_state_sync:latest"
)

echo ""
echo "Loading ${#IMAGES[@]} images into kind cluster..."

for IMAGE in "${IMAGES[@]}"; do
  echo "   Loading $IMAGE..."

  # Verify image exists
  if ! $CONTAINER_ENGINE image inspect "$IMAGE" >/dev/null 2>&1; then
    echo "      ❌ Image not found. Run 'make build-all' first"
    exit 1
  fi

  # Warn if architecture mismatch (don't block)
  IMAGE_ARCH=$($CONTAINER_ENGINE image inspect "$IMAGE" --format '{{.Architecture}}' 2>/dev/null)
  if [ -n "$IMAGE_ARCH" ] && [ "$IMAGE_ARCH" != "$EXPECTED_ARCH" ]; then
    echo "      ⚠️  Image is $IMAGE_ARCH, host is $EXPECTED_ARCH (may be slow)"
  fi

  # Save as OCI archive
  $CONTAINER_ENGINE save --format oci-archive -o "/tmp/${IMAGE//://}.oci.tar" "$IMAGE"
  
  # Import into kind node with docker.io/library prefix so kubelet can find it
  cat "/tmp/${IMAGE//://}.oci.tar" | \
    $CONTAINER_ENGINE exec -i ambient-local-control-plane \
    ctr --namespace=k8s.io images import --no-unpack \
    --index-name "docker.io/library/$IMAGE" - 2>&1 | grep -q "saved" && \
    echo "      ✓ $IMAGE loaded" || \
    echo "      ⚠️  $IMAGE may have failed"
  
  # Cleanup temp file
  rm -f "/tmp/${IMAGE//://}.oci.tar"
done

echo ""
echo "✅ All images loaded into kind cluster!"
echo ""
echo "Verifying images in cluster..."
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  $CONTAINER_ENGINE exec ambient-local-control-plane crictl images | grep vteam_ | head -n 5
else
  docker exec ambient-local-control-plane crictl images | grep vteam_ | head -n 5
fi
