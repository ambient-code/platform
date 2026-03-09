#!/bin/bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
FRONTEND_DIR="$REPO_ROOT/components/frontend"

echo "▶ Running unit tests with coverage..."
cd "$FRONTEND_DIR"
npx vitest run --coverage

echo ""
echo "✅ Unit test coverage report above"
echo "   For E2E integration tests: cd e2e && npx cypress run --spec cypress/e2e/sessions.cy.ts"
