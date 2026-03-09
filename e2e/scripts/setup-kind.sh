#!/bin/bash
set -euo pipefail

echo "======================================"
echo "Setting up kind cluster for Ambient"
echo "======================================"

# Cluster name (override via env var for multi-worktree support)
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-ambient-local}"

# Detect container runtime (prefer explicit CONTAINER_ENGINE, then Docker, then Podman)
CONTAINER_ENGINE="${CONTAINER_ENGINE:-}"

if [ -z "$CONTAINER_ENGINE" ]; then
  if command -v docker &> /dev/null && docker ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &> /dev/null; then
    CONTAINER_ENGINE="podman"
  else
    echo "Error: Neither Docker nor Podman found or running"
    echo "   Please install and start Docker or Podman"
    echo "   Docker: https://docs.docker.com/get-docker/"
    echo "   Podman: brew install podman && podman machine init && podman machine start"
    exit 1
  fi
fi

echo "Using container runtime: $CONTAINER_ENGINE"

# Configure kind to use Podman if selected
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  export KIND_EXPERIMENTAL_PROVIDER=podman
  echo "   Set KIND_EXPERIMENTAL_PROVIDER=podman"

  # Verify Podman is running
  if ! podman ps &> /dev/null; then
    echo "Podman is installed but not running"
    echo "   Start it with: podman machine start"
    exit 1
  fi
fi

# Check if kind cluster already exists
if kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
  echo "Kind cluster '${KIND_CLUSTER_NAME}' already exists"
  echo "   Run 'make kind-down' or delete manually: kind delete cluster --name ${KIND_CLUSTER_NAME}"
  exit 1
fi

echo ""
echo "Creating kind cluster '${KIND_CLUSTER_NAME}'..."

# Port defaults: use env vars if set, otherwise pick based on container engine
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  HTTP_PORT="${KIND_HTTP_PORT:-8080}"
  HTTPS_PORT="${KIND_HTTPS_PORT:-8443}"
  echo "   Using ports ${HTTP_PORT}/${HTTPS_PORT} (Podman rootless compatibility)"
else
  HTTP_PORT="${KIND_HTTP_PORT:-80}"
  HTTPS_PORT="${KIND_HTTPS_PORT:-443}"
  echo "   Using ports ${HTTP_PORT}/${HTTPS_PORT} (Docker standard ports)"
fi

cat <<EOF | kind create cluster --name "${KIND_CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  apiServerAddress: "${API_SERVER_ADDRESS}"
nodes:
- role: control-plane
${CERT_SAN_PATCH}
  # Kind v0.31.0 default node image
  image: kindest/node:v1.35.0@sha256:452d707d4862f52530247495d180205e029056831160e22870e37e3f6c1ac31f
  extraPortMappings:
  - containerPort: 30080
    hostPort: ${HTTP_PORT}
    protocol: TCP
  - containerPort: 30443
    hostPort: ${HTTPS_PORT}
    protocol: TCP
EOF

echo ""
echo "Kind cluster ready!"
echo "   Cluster: ${KIND_CLUSTER_NAME}"
echo "   Kubernetes: v1.35.0"
echo "   NodePort: 30080 -> host port ${HTTP_PORT}"
echo ""
echo "Next steps:"
echo "   1. Deploy the platform: make kind-up (continues deployment)"
echo "   2. Access services: make kind-port-forward (in another terminal)"
echo "   3. Frontend: http://localhost:${HTTP_PORT}"
