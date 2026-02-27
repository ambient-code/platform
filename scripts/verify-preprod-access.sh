#!/usr/bin/env bash
set -euo pipefail

echo "=== Identity ==="
oc whoami

echo "=== Namespace-scoped permissions (ambient-code--runtime-int) ==="
ns=ambient-code--runtime-int
failed=0
for check in \
  "create deployments.apps" \
  "patch deployments.apps" \
  "create services" \
  "create configmaps" \
  "patch configmaps" \
  "create persistentvolumeclaims" \
  "create serviceaccounts" \
  "create routes.route.openshift.io" \
  "patch routes.route.openshift.io" \
; do
  if oc auth can-i $check -n "$ns" 2>/dev/null; then
    echo "  YES  $check"
  else
    echo "  NO   $check"
    failed=1
  fi
done

echo ""
if [ "$failed" -eq 1 ]; then
  echo "ERROR: one or more permissions are missing (see above)"
  exit 1
fi
echo "All permissions OK"

echo "=== Tool versions ==="
oc version --client
echo "=== oc kustomize (built-in, no standalone binary needed — see #768) ==="
oc kustomize --help | head -1
