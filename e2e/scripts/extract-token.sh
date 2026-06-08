#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "Extracting test user token..."

# Cluster name (override via env var for multi-worktree support)
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-ambient-local}"

# Default: K8s SA token (works in both legacy and SSO mode via TokenReview fallback).
# Set E2E_USE_SSO=true to use Keycloak client_credentials instead.
TOKEN=""

if [ "${E2E_USE_SSO:-false}" = "true" ]; then
  KEYCLOAK_URL="http://keycloak-service.ambient-code.svc.cluster.local:8080"
  KEYCLOAK_REALM="ambient-code"
  E2E_CLIENT_ID="${E2E_CLIENT_ID:-ambient-e2e}"
  E2E_CLIENT_SECRET="${E2E_CLIENT_SECRET:-e2e-secret-do-not-use-in-prod}"

  echo "   Obtaining Keycloak token via client_credentials..."
  RESPONSE=$(kubectl run -n ambient-code e2e-token-fetch --rm -i --restart=Never --quiet \
    --image=curlimages/curl -- sh -c \
    "curl -sf -X POST ${KEYCLOAK_URL}/realms/${KEYCLOAK_REALM}/protocol/openid-connect/token \
      -d client_id=${E2E_CLIENT_ID} \
      -d client_secret=${E2E_CLIENT_SECRET} \
      -d grant_type=client_credentials \
      -d scope=openid" 2>/dev/null || echo "")
  TOKEN=$(echo "$RESPONSE" | jq -r '.access_token // empty' 2>/dev/null || echo "")
  if [ -n "$TOKEN" ]; then
    echo "   Token obtained from Keycloak (client_credentials)"
  else
    echo "   Keycloak token fetch failed, falling back to K8s SA token..."
  fi
fi

if [ -z "$TOKEN" ]; then
  for i in {1..15}; do
    TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' 2>/dev/null | base64 -d 2>/dev/null || echo "")
    if [ -n "$TOKEN" ]; then
      echo "   Token extracted from K8s SA"
      break
    fi
    if [ $i -eq 15 ]; then
      echo "Failed to extract test token after 30 seconds"
      exit 1
    fi
    sleep 2
  done
fi

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
