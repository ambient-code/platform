#!/usr/bin/env bash
set -euo pipefail

COMMAND="${1:-}"
INSTANCE_ID="${2:-}"

CONFIG_NAMESPACE="ambient-code--config"
ARGOCD_NAMESPACE="${ARGOCD_NAMESPACE:-ambient-code--argocd}"
MAX_S0X_INSTANCES="${MAX_S0X_INSTANCES:-5}"
READY_TIMEOUT="${READY_TIMEOUT:-60}"
DELETE_TIMEOUT="${DELETE_TIMEOUT:-120}"

usage() {
  echo "Usage: $0 <create|destroy> <instance-id>"
  echo "  instance-id: e.g. pr-123-feat-xyz"
  echo ""
  echo "Environment variables:"
  echo "  MAX_S0X_INSTANCES  Maximum concurrent S0.x instances (default: 5)"
  echo "  READY_TIMEOUT      Seconds to wait for namespace Active (default: 60)"
  echo "  DELETE_TIMEOUT     Seconds to wait for namespace deletion (default: 120)"
  exit 1
}

[[ -z "$COMMAND" || -z "$INSTANCE_ID" ]] && usage
[[ "$COMMAND" != "create" && "$COMMAND" != "destroy" ]] && usage

NAMESPACE="ambient-code--${INSTANCE_ID}"

create() {
  echo "==> Reserving slot via ConfigMap lock..."
  LOCK_NAME="pr-test-slot-${INSTANCE_ID}"
  if ! oc create configmap "$LOCK_NAME" -n "$CONFIG_NAMESPACE" \
    --from-literal=instance="$INSTANCE_ID" \
    --from-literal=created="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    2>/dev/null; then
    echo "ERROR: Slot already reserved for instance $INSTANCE_ID (lock $LOCK_NAME exists)"
    exit 1
  fi
  echo "    Slot reserved: $LOCK_NAME"

  echo "==> Checking S0.x instance capacity..."
  ACTIVE=$(oc get tenantnamespace -n "$CONFIG_NAMESPACE" \
    -l ambient-code/instance-type=s0x --no-headers 2>/dev/null | wc -l | tr -d ' ')

  if [ "$ACTIVE" -ge "$MAX_S0X_INSTANCES" ]; then
    echo "ERROR: At capacity — $ACTIVE/$MAX_S0X_INSTANCES S0.x instances active."
    echo "Active instances:"
    oc get tenantnamespace -n "$CONFIG_NAMESPACE" \
      -l ambient-code/instance-type=s0x -o name
    oc delete configmap "$LOCK_NAME" -n "$CONFIG_NAMESPACE" --ignore-not-found=true
    exit 1
  fi
  echo "    Capacity OK: $ACTIVE/$MAX_S0X_INSTANCES"

  echo "==> Applying TenantNamespace CR: $INSTANCE_ID"
  cat <<EOF | oc apply -f -
apiVersion: tenant.paas.redhat.com/v1alpha1
kind: TenantNamespace
metadata:
  labels:
    tenant.paas.redhat.com/namespace-type: runtime
    tenant.paas.redhat.com/tenant: ambient-code
    ambient-code/instance-type: s0x
  name: ${INSTANCE_ID}
  namespace: ${CONFIG_NAMESPACE}
spec:
  network:
    security-zone: internal
  type: runtime
EOF

  echo "==> Waiting for namespace ${NAMESPACE} to become Active (timeout: ${READY_TIMEOUT}s)..."
  DEADLINE=$((SECONDS + READY_TIMEOUT))
  while [ $SECONDS -lt $DEADLINE ]; do
    NS_STATUS=$(oc get namespace "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || true)
    TN_READY=$(oc get tenantnamespace "$INSTANCE_ID" -n "$CONFIG_NAMESPACE" \
      -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)
    TN_RECONCILED=$(oc get tenantnamespace "$INSTANCE_ID" -n "$CONFIG_NAMESPACE" \
      -o jsonpath='{.status.lastSuccessfulReconciliationTimestamp}' 2>/dev/null || true)
    if [ "$NS_STATUS" == "Active" ] && { [ "$TN_READY" == "True" ] || [ -n "$TN_RECONCILED" ]; }; then
      echo "    Namespace ${NAMESPACE} is Active and TenantNamespace is Ready."
      echo "$NAMESPACE"
      exit 0
    fi
    echo "    ns=${NS_STATUS:-NotFound} tn-ready=${TN_READY:-unknown}, retrying..."
    sleep 3
  done

  echo "ERROR: Namespace ${NAMESPACE} did not become Active+Ready within ${READY_TIMEOUT}s."
  oc describe tenantnamespace "$INSTANCE_ID" -n "$CONFIG_NAMESPACE" || true
  exit 1
}

destroy() {
  APP_NAME="pr-test-${INSTANCE_ID}"
  echo "==> Deleting ArgoCD Application: $APP_NAME"
  oc delete application "$APP_NAME" -n "$ARGOCD_NAMESPACE" \
    --ignore-not-found=true 2>/dev/null || true

  echo "==> Deleting TenantNamespace CR: $INSTANCE_ID"
  oc delete tenantnamespace "$INSTANCE_ID" -n "$CONFIG_NAMESPACE" \
    --ignore-not-found=true

  LOCK_NAME="pr-test-slot-${INSTANCE_ID}"
  oc delete configmap "$LOCK_NAME" -n "$CONFIG_NAMESPACE" --ignore-not-found=true 2>/dev/null || true

  echo "==> Waiting for namespace ${NAMESPACE} to be deleted (timeout: ${DELETE_TIMEOUT}s)..."
  DEADLINE=$((SECONDS + DELETE_TIMEOUT))
  while [ $SECONDS -lt $DEADLINE ]; do
    NS_CHECK=$(oc get namespace "$NAMESPACE" 2>&1 || true)
    if echo "$NS_CHECK" | grep -q '(NotFound)\|not found'; then
      echo "    Namespace ${NAMESPACE} deleted."
      exit 0
    elif [ -z "$(oc get namespace "$NAMESPACE" -o name 2>/dev/null || true)" ]; then
      echo "    Namespace ${NAMESPACE} deleted."
      exit 0
    fi
    echo "    Namespace still exists, waiting..."
    sleep 5
  done

  echo "WARNING: Namespace ${NAMESPACE} still exists after ${DELETE_TIMEOUT}s. May need manual cleanup."
  exit 1
}

case "$COMMAND" in
  create)  create ;;
  destroy) destroy ;;
esac
