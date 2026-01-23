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

# Check if agent testing is enabled
AGENT_TESTING="${AGENT_TESTING_ENABLED:-false}"

echo ""
echo "Test token loaded ✓"
echo "Base URL: $CYPRESS_BASE_URL"
if [ "$AGENT_TESTING" = "true" ]; then
  echo "Agent testing: ENABLED (Running Session tests will execute)"
else
  echo "Agent testing: DISABLED (Running Session tests will be skipped)"
  echo "   To enable, add ANTHROPIC_API_KEY to e2e/.env and redeploy"
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

CYPRESS_TEST_TOKEN="$TEST_TOKEN" \
  CYPRESS_BASE_URL="$CYPRESS_BASE_URL" \
  CYPRESS_AGENT_TESTING_ENABLED="$AGENT_TESTING" \
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
  echo "  - Check ingress: kubectl get ingress -n ambient-code"
  echo "  - Test manually: curl http://localhost"
fi

exit $exit_code

