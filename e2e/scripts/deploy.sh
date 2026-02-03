#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "======================================"
echo "Deploying Ambient to kind cluster"
echo "======================================"

# Load .env file if it exists (for ANTHROPIC_API_KEY)
if [ -f ".env" ]; then
  echo "Loading configuration from .env..."
  # Source the .env file, handling quotes properly
  set -a
  source .env
  set +a
  echo "   ✓ Loaded .env"
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
if ! kind get clusters 2>/dev/null | grep -q "^ambient-local$"; then
  echo "❌ Kind cluster 'ambient-local' not found"
  echo "   Run './scripts/setup-kind.sh' first"
  exit 1
fi

echo ""
echo "Applying manifests with kustomize..."
echo "   Using overlay: kind"

# Check for image overrides in .env
if [ -f ".env" ]; then
  source .env
  
  # Log image overrides
  if [ -n "${IMAGE_BACKEND:-}${IMAGE_FRONTEND:-}${IMAGE_OPERATOR:-}${IMAGE_RUNNER:-}${IMAGE_STATE_SYNC:-}" ]; then
    echo "   ℹ️  Image overrides from .env:"
    [ -n "${IMAGE_BACKEND:-}" ] && echo "      Backend: ${IMAGE_BACKEND}"
    [ -n "${IMAGE_FRONTEND:-}" ] && echo "      Frontend: ${IMAGE_FRONTEND}"
    [ -n "${IMAGE_OPERATOR:-}" ] && echo "      Operator: ${IMAGE_OPERATOR}"
    [ -n "${IMAGE_RUNNER:-}" ] && echo "      Runner: ${IMAGE_RUNNER}"
    [ -n "${IMAGE_STATE_SYNC:-}" ] && echo "      State-sync: ${IMAGE_STATE_SYNC}"
  fi
fi

# Build manifests and apply with image substitution (if IMAGE_* vars set)
# Use --validate=false for remote Podman API server compatibility
kubectl kustomize ../components/manifests/overlays/kind/ | \
  sed "s|quay.io/ambient_code/vteam_backend:latest|${IMAGE_BACKEND:-quay.io/ambient_code/vteam_backend:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_frontend:latest|${IMAGE_FRONTEND:-quay.io/ambient_code/vteam_frontend:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_operator:latest|${IMAGE_OPERATOR:-quay.io/ambient_code/vteam_operator:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_claude_runner:latest|${IMAGE_RUNNER:-quay.io/ambient_code/vteam_claude_runner:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_state_sync:latest|${IMAGE_STATE_SYNC:-quay.io/ambient_code/vteam_state_sync:latest}|g" | \
  kubectl apply --validate=false -f -

# Inject ANTHROPIC_API_KEY if set (for agent testing)
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  echo ""
  echo "Injecting ANTHROPIC_API_KEY into runner secrets..."
  kubectl patch secret ambient-runner-secrets -n ambient-code \
    --type='json' \
    -p="[{\"op\": \"replace\", \"path\": \"/stringData/ANTHROPIC_API_KEY\", \"value\": \"${ANTHROPIC_API_KEY}\"}]" 2>/dev/null || \
  kubectl create secret generic ambient-runner-secrets -n ambient-code \
    --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
    --dry-run=client -o yaml | kubectl apply --validate=false -f -
  echo "   ✓ ANTHROPIC_API_KEY injected (agent testing enabled)"
else
  echo ""
  echo "⚠️  No ANTHROPIC_API_KEY found - agent testing will be limited"
  echo "   To enable full agent testing, create e2e/.env with:"
  echo "   ANTHROPIC_API_KEY=your-api-key-here"
fi

echo ""
echo "Waiting for deployments to be ready..."
./scripts/wait-for-ready.sh

echo ""
echo "Initializing MinIO storage..."
./scripts/init-minio.sh

echo ""
echo "Extracting test user token..."
# Wait for the secret to be populated with a token (max 30 seconds)
TOKEN=""
for i in {1..15}; do
  TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' 2>/dev/null | base64 -d 2>/dev/null || echo "")
  if [ -n "$TOKEN" ]; then
    echo "   ✓ Token extracted successfully"
    break
  fi
  if [ $i -eq 15 ]; then
    echo "❌ Failed to extract test token after 30 seconds"
    echo "   The secret may not be ready. Check with:"
    echo "   kubectl get secret test-user-token -n ambient-code"
    exit 1
  fi
  sleep 2
done

# Detect which port to use based on container engine
# Podman uses port 8080 (rootless compatibility), Docker uses port 80
if [ "${CONTAINER_ENGINE:-}" = "podman" ]; then
  HTTP_PORT=8080
else
  # Auto-detect if not explicitly set
  if podman ps --filter "name=ambient-local-control-plane" 2>/dev/null | grep -q "ambient-local"; then
    HTTP_PORT=8080
  else
    HTTP_PORT=80
  fi
fi

# Use localhost instead of vteam.local to avoid needing /etc/hosts modification
BASE_URL="http://localhost"
if [ "$HTTP_PORT" != "80" ]; then
  BASE_URL="http://localhost:${HTTP_PORT}"
fi

echo "TEST_TOKEN=$TOKEN" > .env.test
echo "CYPRESS_BASE_URL=$BASE_URL" >> .env.test
# Save ANTHROPIC_API_KEY to .env.test if set (for agent testing in Cypress)
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  echo "ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY" >> .env.test
  echo "   ✓ Token saved to .env.test"
  echo "   ✓ Base URL: $BASE_URL"
  echo "   ✓ API Key saved (agent testing enabled)"
else
  echo "   ✓ Token saved to .env.test"
  echo "   ✓ Base URL: $BASE_URL"
  echo "   ⚠️  No API Key (agent testing will be skipped)"
fi

echo ""
echo "✅ Deployment complete!"
echo ""
echo "Access the application:"
echo "   Frontend: $BASE_URL"
echo "   Backend:  $BASE_URL/api/health"
echo ""
echo "Check pod status:"
echo "   kubectl get pods -n ambient-code"
echo ""
echo "Run tests:"
echo "   ./scripts/run-tests.sh"

