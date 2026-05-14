#!/usr/bin/env bash
set -euo pipefail

IMAGE="${1:?Usage: run-conformance.sh <image-ref>}"
CONTAINER_NAME="conformance-$$"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-30}"
HEALTH_PORT="${HEALTH_PORT:-8001}"
PASS=0
FAIL=0
RESULTS=()

# Detect container runtime
if command -v docker &>/dev/null; then
  RUNTIME=docker
elif command -v podman &>/dev/null; then
  RUNTIME=podman
else
  echo "ERROR: neither docker nor podman found in PATH" >&2
  exit 1
fi

cleanup() {
  "$RUNTIME" rm -f "$CONTAINER_NAME" &>/dev/null || true
}
trap cleanup EXIT

log_pass() {
  PASS=$((PASS + 1))
  RESULTS+=("PASS: $1")
  echo "  PASS: $1"
}

log_fail() {
  FAIL=$((FAIL + 1))
  RESULTS+=("FAIL: $1 -- $2")
  echo "  FAIL: $1 -- $2"
}

echo "=== Runner Conformance Test Suite ==="
echo "Image: $IMAGE"
echo "Runtime: $RUNTIME"
echo ""

# --- 1. Non-root user ---
echo "[1/6] Checking non-root user..."
UID_OUTPUT=$("$RUNTIME" run --rm --entrypoint id "$IMAGE" -u 2>/dev/null || echo "")
if [ -n "$UID_OUTPUT" ] && [ "$UID_OUTPUT" != "0" ]; then
  log_pass "runs as non-root (uid=$UID_OUTPUT)"
else
  log_fail "non-root user" "container runs as root (uid=${UID_OUTPUT:-unknown})"
fi

# --- 2. Required filesystem paths ---
echo "[2/6] Checking required filesystem paths..."
REQUIRED_PATHS=("/workspace" "/home/user" "/tmp")
for p in "${REQUIRED_PATHS[@]}"; do
  if "$RUNTIME" run --rm --entrypoint test "$IMAGE" -d "$p" 2>/dev/null; then
    log_pass "directory exists: $p"
  else
    log_fail "directory exists: $p" "missing or not a directory"
  fi
done

# Check /workspace is writable by non-root user
if "$RUNTIME" run --rm --entrypoint sh "$IMAGE" -c "touch /workspace/.conformance-test && rm /workspace/.conformance-test" 2>/dev/null; then
  log_pass "/workspace is writable"
else
  log_fail "/workspace writable" "/workspace is not writable by the container user"
fi

# --- 3. AG-UI health endpoint ---
echo "[3/6] Starting container and checking AG-UI endpoints..."
"$RUNTIME" run -d --name "$CONTAINER_NAME" \
  -e ANTHROPIC_API_KEY=sk-test-conformance \
  -e BACKEND_API_URL=http://localhost:9999 \
  -e RUNNER_TYPE=claude-agent-sdk \
  -e SESSION_NAME=conformance-test \
  -e NAMESPACE=conformance \
  "$IMAGE" >/dev/null 2>&1

HEALTHY=false
for i in $(seq 1 "$HEALTH_TIMEOUT"); do
  if "$RUNTIME" exec "$CONTAINER_NAME" curl -sf "http://localhost:${HEALTH_PORT}/health" >/dev/null 2>&1; then
    HEALTHY=true
    log_pass "AG-UI /health responds within ${i}s"
    break
  fi
  sleep 1
done

if [ "$HEALTHY" = false ]; then
  log_fail "AG-UI /health" "did not respond within ${HEALTH_TIMEOUT}s"
fi

# Check /capabilities endpoint
if [ "$HEALTHY" = true ]; then
  CAPS=$("$RUNTIME" exec "$CONTAINER_NAME" curl -sf "http://localhost:${HEALTH_PORT}/capabilities" 2>/dev/null || echo "")
  if [ -n "$CAPS" ]; then
    log_pass "AG-UI /capabilities responds"
  else
    log_fail "AG-UI /capabilities" "no response"
  fi

  ROOT_STATUS=$("$RUNTIME" exec "$CONTAINER_NAME" curl -s -o /dev/null -w '%{http_code}' "http://localhost:${HEALTH_PORT}/" 2>/dev/null || echo "000")
  if [ "$ROOT_STATUS" != "000" ]; then
    log_pass "AG-UI / reachable (HTTP $ROOT_STATUS)"
  else
    log_fail "AG-UI /" "not reachable"
  fi
fi

"$RUNTIME" rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true

# --- 4. CP-injected env vars not baked in ---
echo "[4/6] Checking CP-injected env vars are not baked into image..."
CP_VARS=("SESSION_NAME" "NAMESPACE" "BACKEND_API_URL" "GRPC_SERVER_URL")
for v in "${CP_VARS[@]}"; do
  BAKED=$("$RUNTIME" run --rm --entrypoint printenv "$IMAGE" "$v" 2>/dev/null || echo "")
  if [ -z "$BAKED" ]; then
    log_pass "env $v not baked into image"
  else
    log_fail "env $v" "baked into image with value '$BAKED'"
  fi
done

# --- 5. Contract version label ---
echo "[5/6] Checking OCI contract version label..."
LABEL=$("$RUNTIME" inspect --format='{{index .Config.Labels "dev.ambient.runner.contract-version"}}' "$IMAGE" 2>/dev/null || echo "")
if [ -n "$LABEL" ] && [ "$LABEL" != "<no value>" ]; then
  log_pass "contract version label present: $LABEL"
else
  log_fail "contract version label" "dev.ambient.runner.contract-version label missing"
fi

# --- 6. SUID/SGID binaries (advisory) ---
echo "[6/6] Checking for SUID/SGID binaries..."
SUID_COUNT=$(
  set +o pipefail
  "$RUNTIME" run --rm --entrypoint sh "$IMAGE" -c \
    "timeout 30 find / -xdev -perm /6000 -type f 2>/dev/null | wc -l | tr -d '[:space:]'"
) || SUID_COUNT="0"
if [ "$SUID_COUNT" -eq 0 ] 2>/dev/null; then
  log_pass "no SUID/SGID binaries found"
else
  log_pass "SUID/SGID advisory: $SUID_COUNT binaries (SecurityContext prevents escalation)"
fi

# --- Summary ---
echo ""
echo "=== Results ==="
for r in "${RESULTS[@]}"; do
  echo "  $r"
done
echo ""
echo "Total: $((PASS + FAIL)) checks, $PASS passed, $FAIL failed"

if [ "$FAIL" -gt 0 ]; then
  echo "CONFORMANCE: FAIL"
  exit 1
else
  echo "CONFORMANCE: PASS"
  exit 0
fi
