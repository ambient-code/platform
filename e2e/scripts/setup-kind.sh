#!/bin/bash
set -euo pipefail

echo "======================================"
echo "Setting up kind cluster for Ambient"
echo "======================================"

# Detect container runtime (prefer explicit CONTAINER_ENGINE, then Docker, then Podman)
CONTAINER_ENGINE="${CONTAINER_ENGINE:-}"

if [ -z "$CONTAINER_ENGINE" ]; then
  if command -v docker &> /dev/null && docker ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &> /dev/null; then
    CONTAINER_ENGINE="podman"
  else
    echo "❌ Error: Neither Docker nor Podman found or running"
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
  echo "   ℹ️  Set KIND_EXPERIMENTAL_PROVIDER=podman"
  
  # Verify Podman is running
  if ! podman ps &> /dev/null; then
    echo "❌ Podman is installed but not running"
    echo "   Start it with: podman machine start"
    exit 1
  fi
fi

# Check if kind cluster already exists
if kind get clusters 2>/dev/null | grep -q "^ambient-local$"; then
  echo "⚠️  Kind cluster 'ambient-local' already exists"
  echo "   Run './scripts/cleanup.sh' first to remove it"
  exit 1
fi

echo ""
echo "Creating kind cluster..."

# Use higher ports for Podman rootless compatibility (ports >= 1024)
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  HTTP_PORT=8080
  HTTPS_PORT=8443
  echo "   ℹ️  Using ports 8080/8443 (Podman rootless compatibility)"
else
  HTTP_PORT=80
  HTTPS_PORT=443
  echo "   ℹ️  Using ports 80/443 (Docker standard ports)"
fi

cat <<EOF | kind create cluster --name ambient-local --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  # Use more stable Kubernetes version for Podman compatibility
  image: kindest/node:v1.28.0@sha256:b7a4cad12c197af3ba43202d3efe03246b3f0793f162afb40a33c923952d5b31
  extraPortMappings:
  - containerPort: 30080
    hostPort: ${HTTP_PORT}
    protocol: TCP
  - containerPort: 30443
    hostPort: ${HTTPS_PORT}
    protocol: TCP
EOF

echo ""
echo "✅ Kind cluster ready!"
echo "   Cluster: ambient-local"
echo "   Kubernetes: v1.28.0"
echo "   NodePort: 30080 → host port ${HTTP_PORT}"
echo ""
echo "📝 Next steps:"
echo "   1. Deploy the platform: make kind-up (continues deployment)"
echo "   2. Access services: make kind-port-forward (in another terminal)"
echo "   3. Frontend: http://localhost:${HTTP_PORT}"

