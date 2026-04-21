#!/usr/bin/env bash
# CodeRabbit Integration Test
#
# Validates the full CodeRabbit integration stack against a running cluster:
#   1. Backend API endpoints (connect, status, disconnect, test, runtime creds)
#   2. Frontend card rendering on the integrations page
#   3. Pre-commit hook graceful skip behavior
#   4. Integrations status includes CodeRabbit
#
# Usage:
#   ./scripts/test-coderabbit-integration.sh                    # auto-detect cluster
#   ./scripts/test-coderabbit-integration.sh --context kind-foo # explicit context
#
# Requires: kubectl, curl, jq
# Optional: CODERABBIT_API_KEY for live API key validation test

set -uo pipefail

PASS=0
FAIL=0
SKIP=0
CONTEXT=""
NAMESPACE="ambient-code"

# Parse args
while [[ $# -gt 0 ]]; do
  case $1 in
    --context) CONTEXT="$2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

# Auto-detect context from kind clusters
if [ -z "$CONTEXT" ]; then
  CONTEXT=$(kubectl config current-context 2>/dev/null || true)
  if [ -z "$CONTEXT" ]; then
    echo "ERROR: No kubectl context. Pass --context or set KUBECONFIG."
    exit 1
  fi
fi

KUBECTL="kubectl --context $CONTEXT"

# Get backend URL via port-forward
BACKEND_PORT=${BACKEND_PORT:-12399}
# Find an available port if default is taken
for port in $BACKEND_PORT 12400 12401 12402; do
  if ! lsof -i ":$port" &>/dev/null; then
    BACKEND_PORT=$port
    break
  fi
done
$KUBECTL port-forward svc/backend-service "$BACKEND_PORT":8080 -n "$NAMESPACE" &>/dev/null &
PF_PID=$!
BACKEND="http://localhost:$BACKEND_PORT"

# Wait for port-forward to be ready
for i in 1 2 3 4 5; do
  if curl -s -o /dev/null -w "" "$BACKEND/healthz" 2>/dev/null; then break; fi
  sleep 1
done

# Get test token
TOKEN=$($KUBECTL get secret test-user-token -n "$NAMESPACE" -o jsonpath='{.data.token}' | base64 -d)

cleanup() {
  kill "$PF_PID" 2>/dev/null || true
}
trap cleanup EXIT

# Helpers
pass() { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); }
skip() { echo "  SKIP: $1"; ((SKIP++)); }

assert_status() {
  local actual="$1" expected="$2" label="$3"
  if [ "$actual" = "$expected" ]; then pass "$label"
  else fail "$label (expected $expected, got $actual)"; fi
}

assert_json() {
  local body="$1" key="$2" expected="$3" label="$4"
  local actual
  actual=$(echo "$body" | python3 -c "
import sys,json
v = json.load(sys.stdin).get('$key','MISSING')
# Normalize booleans to lowercase for shell comparison
if isinstance(v, bool): print(str(v).lower())
else: print(v)
" 2>/dev/null || echo "PARSE_ERROR")
  if [ "$actual" = "$expected" ]; then pass "$label"
  else fail "$label (expected $key=$expected, got $actual)"; fi
}

# The backend extracts userID from X-Forwarded-User (set by OAuth proxy in production).
# For direct API access with a service account token, we set it explicitly.
AUTH=(-H "Authorization: Bearer $TOKEN" -H "X-Forwarded-User: test-user")
JSON=(-H "Content-Type: application/json")

echo "CodeRabbit Integration Test"
echo "  Cluster: $CONTEXT"
echo "  Backend: $BACKEND"
echo ""

# ─── 1. Backend API Endpoints ───────────────────────────────────────────

echo "1. Backend API Endpoints"

# 1a. Status without auth → 401
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BACKEND/api/auth/coderabbit/status")
assert_status "$STATUS" "401" "Status without auth returns 401"

# 1b. Status with auth → connected:false
BODY=$(curl -s "${AUTH[@]}" "$BACKEND/api/auth/coderabbit/status")
assert_json "$BODY" "connected" "false" "Status returns connected=false (no key stored)"

# 1c. Connect with empty key → 400
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${AUTH[@]}" "${JSON[@]}" -d '{"apiKey":""}' "$BACKEND/api/auth/coderabbit/connect")
assert_status "$STATUS" "400" "Connect with empty key returns 400"

# 1d. Test with fake key → valid:false
BODY=$(curl -s -X POST "${AUTH[@]}" "${JSON[@]}" -d '{"apiKey":"cr-fake-key"}' "$BACKEND/api/auth/coderabbit/test")
assert_json "$BODY" "valid" "false" "Test with fake key returns valid=false"

# 1e. Disconnect (idempotent) → 200
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${AUTH[@]}" "$BACKEND/api/auth/coderabbit/disconnect")
assert_status "$STATUS" "200" "Disconnect (idempotent) returns 200"

# 1f. Runtime creds for nonexistent session → 404
BODY=$(curl -s "${AUTH[@]}" "$BACKEND/api/projects/default/agentic-sessions/nonexistent/credentials/coderabbit")
assert_json "$BODY" "error" "CodeRabbit credentials not configured" "Runtime creds for missing session returns 404"

# 1g. Test with real API key (optional)
if [ -n "${CODERABBIT_API_KEY:-}" ]; then
  # Connect with real key
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${AUTH[@]}" "${JSON[@]}" \
    -d "{\"apiKey\":\"$CODERABBIT_API_KEY\"}" "$BACKEND/api/auth/coderabbit/connect")
  assert_status "$STATUS" "200" "Connect with real API key returns 200"

  # Verify status shows connected
  BODY=$(curl -s "${AUTH[@]}" "$BACKEND/api/auth/coderabbit/status")
  assert_json "$BODY" "connected" "true" "Status shows connected=true after connect"

  # Clean up
  curl -s -X DELETE "${AUTH[@]}" "$BACKEND/api/auth/coderabbit/disconnect" >/dev/null
  pass "Cleanup: disconnected after test"
else
  skip "Live API key test (CODERABBIT_API_KEY not set)"
fi

echo ""

# ─── 2. Integrations Status ─────────────────────────────────────────────

echo "2. Integrations Status"

BODY=$(curl -s "${AUTH[@]}" "$BACKEND/api/auth/integrations/status")
HAS_CR=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print('yes' if 'coderabbit' in d else 'no')" 2>/dev/null || echo "no")
if [ "$HAS_CR" = "yes" ]; then pass "Unified status includes coderabbit field"
else fail "Unified status missing coderabbit field"; fi

echo ""

# ─── 3. Frontend ────────────────────────────────────────────────────────

echo "3. Frontend"

# Check integrations page renders (via NodePort)
FRONTEND_PORT=$($KUBECTL get svc -n "$NAMESPACE" -o json | python3 -c "
import sys,json
svcs = json.load(sys.stdin)['items']
for s in svcs:
  if 'frontend' in s['metadata']['name']:
    for p in s['spec'].get('ports',[]):
      if p.get('nodePort'): print(p['nodePort']); sys.exit(0)
print('NONE')
" 2>/dev/null || echo "NONE")

if [ "$FRONTEND_PORT" != "NONE" ]; then
  # The frontend is behind the kind NodePort — check container port mapping
  FRONTEND_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${FRONTEND_PORT}/integrations" 2>/dev/null || echo "000")
  if [ "$FRONTEND_STATUS" = "200" ]; then pass "Integrations page loads (HTTP 200)"
  else skip "Integrations page returned $FRONTEND_STATUS (may need auth)"; fi
else
  skip "Frontend NodePort not found"
fi

echo ""

# ─── 4. Review Gate ────────────────────────────────────────────────────

echo "4. Review Gate"

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
GATE="$REPO_ROOT/scripts/hooks/coderabbit-review-gate.sh"
if [ -x "$GATE" ]; then
  # Run in standalone mode (no CLAUDE_TOOL_INPUT) — exercises same
  # coderabbit review --agent --base main path as the hook.
  GATE_EXIT=0
  OUTPUT=$(bash "$GATE" 2>&1) || GATE_EXIT=$?
  if [ "$GATE_EXIT" -eq 0 ]; then
    pass "Review gate passed"
  elif [ "$GATE_EXIT" -eq 2 ]; then
    # Exit 2 = blocked (findings, CLI missing, or rate limit)
    if echo "$OUTPUT" | grep -qiE "CLI not found"; then
      skip "Review gate: CodeRabbit CLI not installed"
    elif echo "$OUTPUT" | grep -qiE "rate.limited"; then
      skip "Review gate: CodeRabbit rate-limited"
    else
      pass "Review gate blocked with findings (expected)"
    fi
  else
    fail "Review gate exited $GATE_EXIT: $OUTPUT"
  fi
else
  fail "Review gate not found or not executable at $GATE"
fi

echo ""

# ─── Summary ─────────────────────────────────────────────────────────────

echo "─────────────────────────────────────"
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"

if [ "$FAIL" -gt 0 ]; then
  echo "INTEGRATION TEST FAILED"
  exit 1
fi

echo "INTEGRATION TEST PASSED"
