#!/bin/bash
set -euo pipefail

CLAUDE_BIN="${CLAUDE_CLI_PATH:-/usr/local/bin/claude}"

if [[ "${OPENSHELL_ENABLED:-}" == "true" ]]; then
  exec /openshell-sandbox \
    --policy-rules "${OPENSHELL_POLICY_RULES:-/etc/openshell/policy.rego}" \
    --policy-data "${OPENSHELL_POLICY_DATA:-/etc/openshell/policy.yaml}" \
    --log-level "${OPENSHELL_LOG_LEVEL:-warn}" \
    -- "$CLAUDE_BIN" "$@"
else
  exec "$CLAUDE_BIN" "$@"
fi
