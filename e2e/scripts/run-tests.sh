#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "======================================"
echo "Running Ambient E2E Tests"
echo "======================================"

# Load test token and base URL from .env.test if it exists
# Environment variables take precedence over .env.test
if [ -f .env.test ]; then
  # Only load if not already set in environment
  if [ -z "${TEST_TOKEN:-}" ]; then
    source .env.test
  else
    echo "Using TEST_TOKEN from environment (ignoring .env.test)"
  fi
fi

# Check for required config
if [ -z "${TEST_TOKEN:-}" ]; then
  echo "❌ Error: TEST_TOKEN not set"
  echo ""
  echo "Options:"
  echo "  1. For kind: Run 'make kind-up' first (creates .env.test)"
  echo "  2. For manual testing: Set TEST_TOKEN environment variable"
  echo "     Example: TEST_TOKEN=\$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d)"
  echo ""
  exit 1
fi

# Use CYPRESS_BASE_URL from env, .env.test, or default
CYPRESS_BASE_URL="${CYPRESS_BASE_URL:-http://localhost}"

# Load ANTHROPIC_API_KEY from .env or .env.local if available
if [ -f .env.local ]; then
  source .env.local
elif [ -f .env ]; then
  source .env
fi

echo ""
echo "Test token loaded ✓"
echo "Base URL: $CYPRESS_BASE_URL"
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  echo "API Key: ✓ Found in .env (agent tests will run)"
else
  echo "API Key: ✗ Not found (agent tests will FAIL)"
  echo "   Add ANTHROPIC_API_KEY to e2e/.env to run full test suite"
fi
echo ""

# Check if npm packages are installed
if [ ! -d node_modules ]; then
  echo "Installing npm dependencies..."
  npm install
  echo ""
fi

# Run Cypress tests
echo "Starting Cypress tests..."
echo ""

# Cypress will load .env/.env.local via cypress.config.ts
# Pass test token, base URL, and API key (if available)
CYPRESS_TEST_TOKEN="$TEST_TOKEN" \
  CYPRESS_BASE_URL="$CYPRESS_BASE_URL" \
  ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}" \
  npm test

exit_code=$?

echo ""
if [ $exit_code -eq 0 ]; then
  echo "✅ All tests passed!"
else
  echo "❌ Some tests failed (exit code: $exit_code)"
  echo ""
  echo "Debugging tips:"
  echo "  - Check pod logs: kubectl logs -n ambient-code -l app=frontend"
  echo "  - Check services: kubectl get svc -n ambient-code"
  echo "  - Test NodePort: curl http://localhost:8080 (podman) or http://localhost (docker)"
  echo "  - Port-forward: kubectl port-forward -n ambient-code svc/frontend-service 8080:3000"
fi

exit $exit_code

