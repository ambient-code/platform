#!/usr/bin/env bash
set -euo pipefail

PR_INPUT="${1:-}"
CLI="${OC:-oc}"

usage() {
  echo "Usage: $0 <pr-url-or-number>"
  echo "  Tears down the PR test environment created by install-standard.sh"
  exit 1
}

[[ -z "$PR_INPUT" ]] && usage

PR_NUMBER=$(echo "$PR_INPUT" | grep -oE '[0-9]+$')
if [[ -z "$PR_NUMBER" ]]; then
  echo "ERROR: Could not extract PR number from: $PR_INPUT"
  exit 1
fi

NAMESPACE="pr-${PR_NUMBER}"
CR_NAME="ambient-control-plane-${NAMESPACE}"

echo "==> Tearing down PR test environment: $NAMESPACE"

echo "    Deleting ClusterRoleBinding ${CR_NAME}..."
$CLI delete clusterrolebinding "$CR_NAME" --ignore-not-found 2>/dev/null || true

echo "    Deleting ClusterRole ${CR_NAME}..."
$CLI delete clusterrole "$CR_NAME" --ignore-not-found 2>/dev/null || true

echo "    Deleting namespace ${NAMESPACE}..."
$CLI delete namespace "$NAMESPACE" --ignore-not-found --wait=false 2>/dev/null || true

echo "==> Teardown complete for $NAMESPACE"
