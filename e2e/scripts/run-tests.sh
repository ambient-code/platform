#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "======================================"
echo "Running Ambient E2E Tests"
echo "======================================"

# Check if .env.test exists
if [ ! -f .env.test ]; then
  echo "❌ Error: .env.test not found"
  echo "   Run './scripts/deploy.sh' first to set up the environment"
  exit 1
fi

# Load test token and base URL if .env.test exists
if [ -f .env.test ]; then
  source .env.test
fi

# Check for required config
if [ -z "${TEST_TOKEN:-}" ]; then
  echo "❌ Error: TEST_TOKEN not set"
  echo "   For kind: Run kind-up or kind-dev first"
  echo "   For external cluster: Set TEST_TOKEN and CYPRESS_BASE_URL env vars"
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

