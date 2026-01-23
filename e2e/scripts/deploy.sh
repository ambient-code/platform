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
echo "Waiting for ingress admission webhook to be ready..."
# The admission webhook needs time to start even after the controller is ready
for i in {1..30}; do
  if kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io ingress-nginx-admission &>/dev/null; then
    # Give it a few more seconds to be fully ready
    sleep 3
    break
  fi
  if [ $i -eq 30 ]; then
    echo "⚠️  Warning: Admission webhook may not be ready, but continuing..."
    break
  fi
  sleep 2
done

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
kubectl kustomize ../components/manifests/overlays/kind/ | \
  sed "s|quay.io/ambient_code/vteam_backend:latest|${IMAGE_BACKEND:-quay.io/ambient_code/vteam_backend:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_frontend:latest|${IMAGE_FRONTEND:-quay.io/ambient_code/vteam_frontend:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_operator:latest|${IMAGE_OPERATOR:-quay.io/ambient_code/vteam_operator:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_claude_runner:latest|${IMAGE_RUNNER:-quay.io/ambient_code/vteam_claude_runner:latest}|g" | \
  sed "s|quay.io/ambient_code/vteam_state_sync:latest|${IMAGE_STATE_SYNC:-quay.io/ambient_code/vteam_state_sync:latest}|g" | \
  kubectl apply -f -

# Inject ANTHROPIC_API_KEY if set (for agent testing)
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  echo ""
  echo "Injecting ANTHROPIC_API_KEY into runner secrets..."
  kubectl patch secret ambient-runner-secrets -n ambient-code \
    --type='json' \
    -p="[{\"op\": \"replace\", \"path\": \"/stringData/ANTHROPIC_API_KEY\", \"value\": \"${ANTHROPIC_API_KEY}\"}]" 2>/dev/null || \
  kubectl create secret generic ambient-runner-secrets -n ambient-code \
    --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
    --dry-run=client -o yaml | kubectl apply -f -
  echo "   ✓ ANTHROPIC_API_KEY injected (agent testing enabled)"
  
  # Also create a default test project namespace with the secret for manual local dev
  # This allows users to immediately test without going through the API/UI
  echo "   Creating default 'test-project' namespace with API key..."
  kubectl create namespace test-project --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
  kubectl create secret generic ambient-runner-secrets -n test-project \
    --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
  echo "   ✓ test-project namespace ready for manual testing"
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

# Detect which port to use (check kind cluster config)
HTTP_PORT=80
if kind get clusters 2>/dev/null | grep -q "^ambient-local$"; then
  # Check if we're using non-standard ports (Podman)
  if docker ps --filter "name=ambient-local-control-plane" --format "{{.Ports}}" 2>/dev/null | grep -q "8080" || \
     podman ps --filter "name=ambient-local-control-plane" --format "{{.Ports}}" 2>/dev/null | grep -q "8080"; then
    HTTP_PORT=8080
  fi
fi

# Use localhost instead of vteam.local to avoid needing /etc/hosts modification
BASE_URL="http://localhost"
if [ "$HTTP_PORT" != "80" ]; then
  BASE_URL="http://localhost:${HTTP_PORT}"
fi

echo "TEST_TOKEN=$TOKEN" > .env.test
echo "CYPRESS_BASE_URL=$BASE_URL" >> .env.test
# Pass through ANTHROPIC_API_KEY availability for tests to know if agent tests can run
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  echo "AGENT_TESTING_ENABLED=true" >> .env.test
else
  echo "AGENT_TESTING_ENABLED=false" >> .env.test
fi
echo "   ✓ Token saved to .env.test"
echo "   ✓ Base URL: $BASE_URL"

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

