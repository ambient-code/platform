#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

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

# Detect container engine for port detection
CONTAINER_ENGINE="${CONTAINER_ENGINE:-}"
if [ -z "$CONTAINER_ENGINE" ]; then
  if command -v docker &> /dev/null && docker ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &> /dev/null && podman ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="podman"
  fi
fi

# Detect which port to use
HTTP_PORT=80
if kind get clusters 2>/dev/null | grep -q "^ambient-local$"; then
  if docker ps --filter "name=ambient-local-control-plane" --format "{{.Ports}}" 2>/dev/null | grep -q "8080" || \
     podman ps --filter "name=ambient-local-control-plane" --format "{{.Ports}}" 2>/dev/null | grep -q "8080"; then
    HTTP_PORT=8080
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

# Check if ANTHROPIC_API_KEY was provided for agent testing
# Note: Agent testing also requires operator to properly copy secrets to project namespaces
# For now, default to false unless explicitly enabled
if [ "${AGENT_TESTING_ENABLED:-false}" = "true" ]; then
  echo "AGENT_TESTING_ENABLED=true" >> .env.test
else
  echo "AGENT_TESTING_ENABLED=false" >> .env.test
fi

echo "   ✓ Token saved to .env.test"
echo "   ✓ Base URL: $BASE_URL"
