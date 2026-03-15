#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "======================================"
echo "Deploying Ambient to kind cluster"
echo "======================================"

# Cluster name (override via env var for multi-worktree support)
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-ambient-local}"

# Load .env file if it exists (for ANTHROPIC_API_KEY)
if [ -f ".env" ]; then
  echo "Loading configuration from .env..."
  # Source the .env file, handling quotes properly
  set -a
  source .env
  set +a
  echo "   Loaded .env"
fi

# Detect container runtime (same logic as setup-kind.sh)
CONTAINER_ENGINE="${CONTAINER_ENGINE:-}"

if [ -z "$CONTAINER_ENGINE" ]; then
  if command -v docker &> /dev/null && docker ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &> /dev/null && podman ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="podman"
  fi
fi

# Set KIND_EXPERIMENTAL_PROVIDER if using Podman
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  export KIND_EXPERIMENTAL_PROVIDER=podman
fi

# Check if kind cluster exists
if ! kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
  echo "Kind cluster '${KIND_CLUSTER_NAME}' not found"
  echo "   Run './scripts/setup-kind.sh' first"
  exit 1
fi

echo ""
echo "Applying manifests with kustomize..."
echo "   Using overlay: kind"

# Check for image overrides in .env (already sourced above)
if [ -f ".env" ]; then
  # Log image overrides
  if [ -n "${IMAGE_BACKEND:-}${IMAGE_FRONTEND:-}${IMAGE_OPERATOR:-}${IMAGE_RUNNER:-}${IMAGE_STATE_SYNC:-}${IMAGE_PUBLIC_API:-}" ]; then
    echo "   Image overrides from .env:"
    [ -n "${IMAGE_BACKEND:-}" ] && echo "      Backend: ${IMAGE_BACKEND}"
    [ -n "${IMAGE_FRONTEND:-}" ] && echo "      Frontend: ${IMAGE_FRONTEND}"
    [ -n "${IMAGE_OPERATOR:-}" ] && echo "      Operator: ${IMAGE_OPERATOR}"
    [ -n "${IMAGE_RUNNER:-}" ] && echo "      Runner: ${IMAGE_RUNNER}"
    [ -n "${IMAGE_STATE_SYNC:-}" ] && echo "      State-sync: ${IMAGE_STATE_SYNC}"
    [ -n "${IMAGE_PUBLIC_API:-}" ] && echo "      Public-API: ${IMAGE_PUBLIC_API}"
  fi
fi

# Build manifests and apply with image substitution (if IMAGE_* vars set)
# Use --validate=false for remote Podman API server compatibility
# When IMAGE_* overrides are set, also switch imagePullPolicy to IfNotPresent
# since the images are pre-loaded into kind via `kind load docker-image`.
kubectl kustomize ../components/manifests/overlays/kind/ | \
  sed "s|quay.io/ambient_code/vteam_backend:latest|${IMAGE_BACKEND:-quay.io/ambient_code/vteam_backend:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_frontend:latest|${IMAGE_FRONTEND:-quay.io/ambient_code/vteam_frontend:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_operator:latest|${IMAGE_OPERATOR:-quay.io/ambient_code/vteam_operator:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_claude_runner:latest|${IMAGE_RUNNER:-quay.io/ambient_code/vteam_claude_runner:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_state_sync:latest|${IMAGE_STATE_SYNC:-quay.io/ambient_code/vteam_state_sync:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_public_api:latest|${IMAGE_PUBLIC_API:-quay.io/ambient_code/vteam_public_api:latest}|g" | \
  if [ -n "${IMAGE_BACKEND:-}${IMAGE_FRONTEND:-}${IMAGE_OPERATOR:-}${IMAGE_RUNNER:-}${IMAGE_STATE_SYNC:-}${IMAGE_PUBLIC_API:-}" ]; then
    sed "s|imagePullPolicy: Always|imagePullPolicy: IfNotPresent|g"
  else
    cat
  fi | \
  kubectl apply --validate=false -f -

# Inject runner secrets for agent testing
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  echo ""
  echo "Injecting ANTHROPIC_API_KEY into runner secrets..."
  kubectl patch secret ambient-runner-secrets -n ambient-code \
    --type='json' \
    -p="[{\"op\": \"replace\", \"path\": \"/stringData/ANTHROPIC_API_KEY\", \"value\": \"${ANTHROPIC_API_KEY}\"}]" 2>/dev/null || \
  kubectl create secret generic ambient-runner-secrets -n ambient-code \
    --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
    --dry-run=client -o yaml | kubectl apply --validate=false -f -
  echo "   ANTHROPIC_API_KEY injected (agent testing enabled)"
else
  echo ""
  echo "No ANTHROPIC_API_KEY — using mock SDK client (pre-recorded fixtures)"
  kubectl create secret generic ambient-runner-secrets -n ambient-code \
    --from-literal=ANTHROPIC_API_KEY=mock-replay-key \
    --dry-run=client -o yaml | kubectl apply --validate=false -f -
  echo "   Mock SDK configured (ANTHROPIC_API_KEY=mock-replay-key)"
fi

echo ""
echo "Waiting for deployments to be ready..."
./scripts/wait-for-ready.sh

echo ""
echo "Initializing MinIO storage..."
./scripts/init-minio.sh

echo ""
echo "Extracting test user token..."
KIND_CLUSTER_NAME="$KIND_CLUSTER_NAME" KIND_HTTP_PORT="${KIND_HTTP_PORT:-}" CONTAINER_ENGINE="${CONTAINER_ENGINE:-}" ./scripts/extract-token.sh

# Append ANTHROPIC_API_KEY to .env.test if set (for agent testing in Cypress)
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  echo "ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY" >> .env.test
  echo "   API Key saved (agent testing enabled)"
fi

echo ""
echo "Deployment complete!"
echo ""
echo "Check pod status:"
echo "   kubectl get pods -n ambient-code"
echo ""
echo "Run tests:"
echo "   ./scripts/run-tests.sh"
