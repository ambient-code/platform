#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "Extracting test user token..."

# Cluster name (override via env var for multi-worktree support)
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-ambient-local}"

# Wait for the secret to be populated with a token (max 30 seconds)
TOKEN=""
for i in {1..15}; do
  TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' 2>/dev/null | base64 -d 2>/dev/null || echo "")
  if [ -n "$TOKEN" ]; then
    echo "   Token extracted successfully"
    break
  fi
  if [ $i -eq 15 ]; then
    echo "Failed to extract test token after 30 seconds"
    echo "   The secret may not be ready. Check with:"
    echo "   kubectl get secret test-user-token -n ambient-code"
    exit 1
  fi
  sleep 2
done

# Detect container engine for port detection
CONTAINER_ENGINE="${CONTAINER_ENGINE:-}"
if [ -z "$CONTAINER_ENGINE" ]; then
  if command -v docker &> /dev/null && docker ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &> /dev/null && podman ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="podman"
  fi
fi

# Detect which port to use based on container engine
# Respect KIND_HTTP_PORT env var if set, otherwise auto-detect
if [ -n "${KIND_HTTP_PORT:-}" ]; then
  HTTP_PORT="$KIND_HTTP_PORT"
elif [ "$CONTAINER_ENGINE" = "podman" ]; then
  HTTP_PORT=8080
else
  # Auto-detect if not explicitly set
  if podman ps --filter "name=${KIND_CLUSTER_NAME}-control-plane" 2>/dev/null | grep -q "$KIND_CLUSTER_NAME"; then
    HTTP_PORT=8080
  else
    HTTP_PORT=80
  fi
fi

# Use localhost instead of custom hostname
BASE_URL="http://localhost"
if [ "$HTTP_PORT" != "80" ]; then
  BASE_URL="http://localhost:${HTTP_PORT}"
fi

# Write .env.test
echo "TEST_TOKEN=$TOKEN" > .env.test
echo "CYPRESS_BASE_URL=$BASE_URL" >> .env.test

echo "   Token saved to .env.test"
echo "   Base URL: $BASE_URL"
echo ""
echo "To enable agent testing:"
echo "   Add ANTHROPIC_API_KEY to e2e/.env"
echo "   Then run: make test-e2e"
