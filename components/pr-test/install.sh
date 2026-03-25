#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${1:-}"
IMAGE_TAG="${2:-}"

SOURCE_NAMESPACE="${SOURCE_NAMESPACE:-ambient-code--runtime-int}"

usage() {
  echo "Usage: $0 <namespace> <image-tag>"
  echo "  namespace:  e.g. ambient-code--pr-42-feat-xyz"
  echo "  image-tag:  e.g. pr-42-amd64"
  echo ""
  echo "Optional environment variables:"
  echo "  SOURCE_NAMESPACE  Namespace to copy secrets from (default: ambient-code--runtime-int)"
  exit 1
}

[[ -z "$NAMESPACE" || -z "$IMAGE_TAG" ]] && usage

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OVERLAY_DIR="$REPO_ROOT/components/manifests/overlays/production"

copy_secret() {
  local name="$1"
  echo "    Copying secret: $name"
  oc get secret "$name" -n "$SOURCE_NAMESPACE" -o json \
    | jq "del(.metadata.namespace, .metadata.resourceVersion, .metadata.uid, .metadata.creationTimestamp, .metadata.ownerReferences, .metadata.annotations[\"kubectl.kubernetes.io/last-applied-configuration\"])" \
    | oc apply -n "$NAMESPACE" -f -
}

echo "==> Installing Ambient into $NAMESPACE with images tagged $IMAGE_TAG"

echo "==> Step 1: Applying CRDs and RBAC (cluster-scoped, idempotent)"
oc apply -k "$REPO_ROOT/components/manifests/base/crds/"
oc apply -k "$REPO_ROOT/components/manifests/base/rbac/"

echo "==> Step 2: Copying secrets from $SOURCE_NAMESPACE"
copy_secret ambient-vertex
copy_secret ambient-api-server

echo "==> Step 3: Deploying production overlay with image tag $IMAGE_TAG"
TMPDIR=$(mktemp -d)
cp -r "$OVERLAY_DIR/." "$TMPDIR/"
trap "rm -rf $TMPDIR" EXIT

pushd "$TMPDIR" > /dev/null

kustomize edit set namespace "$NAMESPACE"

kustomize edit set image \
  "quay.io/ambient_code/vteam_frontend:latest=quay.io/ambient_code/vteam_frontend:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_backend:latest=quay.io/ambient_code/vteam_backend:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_operator:latest=quay.io/ambient_code/vteam_operator:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_claude_runner:latest=quay.io/ambient_code/vteam_claude_runner:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_state_sync:latest=quay.io/ambient_code/vteam_state_sync:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_api_server:latest=quay.io/ambient_code/vteam_api_server:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_public_api:latest=quay.io/ambient_code/vteam_public_api:${IMAGE_TAG}"

oc apply -k . -n "$NAMESPACE"
popd > /dev/null

echo "==> Step 4: Patching operator ConfigMap with PR image tags"
SOURCE_OPERATOR_CONFIG=$(oc get configmap operator-config -n "$SOURCE_NAMESPACE" -o json \
  | jq -r '.data | to_entries | map(select(.key | test("VERTEX|CLOUD_ML|ANTHROPIC|GOOGLE"))) | from_entries' \
  2>/dev/null || echo '{}')

VERTEX_PATCH=$(echo "$SOURCE_OPERATOR_CONFIG" | jq -c \
  --arg runner "quay.io/ambient_code/vteam_claude_runner:${IMAGE_TAG}" \
  --arg sync "quay.io/ambient_code/vteam_state_sync:${IMAGE_TAG}" \
  '. + {"AMBIENT_CODE_RUNNER_IMAGE": $runner, "STATE_SYNC_IMAGE": $sync}')

oc patch configmap operator-config -n "$NAMESPACE" --type=merge \
  -p "{\"data\": $VERTEX_PATCH}"

echo "==> Step 5: Patching agent registry ConfigMap with PR image tags"
REGISTRY=$(oc get configmap ambient-agent-registry -n "$NAMESPACE" \
  -o jsonpath='{.data.agent-registry\.json}' 2>/dev/null || echo "{}")

REGISTRY=$(echo "$REGISTRY" | sed \
  "s|quay.io/ambient_code/vteam_claude_runner[@:][^\"]*|quay.io/ambient_code/vteam_claude_runner:${IMAGE_TAG}|g")
REGISTRY=$(echo "$REGISTRY" | sed \
  "s|quay.io/ambient_code/vteam_state_sync[@:][^\"]*|quay.io/ambient_code/vteam_state_sync:${IMAGE_TAG}|g")

oc patch configmap ambient-agent-registry -n "$NAMESPACE" --type=merge \
  -p "{\"data\":{\"agent-registry.json\":$(echo "$REGISTRY" | jq -Rs .)}}"

echo "==> Step 6: Waiting for rollouts"
for deploy in backend-api frontend agentic-operator postgresql minio unleash public-api; do
  echo "    Waiting for $deploy..."
  oc rollout status deployment/$deploy -n "$NAMESPACE" --timeout=300s
done

echo "    Waiting for ambient-api-server-db..."
oc rollout status deployment/ambient-api-server-db -n "$NAMESPACE" --timeout=300s

echo "    Waiting for ambient-api-server..."
oc rollout status deployment/ambient-api-server -n "$NAMESPACE" --timeout=300s

echo "==> Step 7: Verifying health"
BACKEND_HOST=$(oc get route backend-route -n "$NAMESPACE" \
  -o jsonpath='{.spec.host}' 2>/dev/null || true)

if [[ -n "$BACKEND_HOST" ]]; then
  HEALTH=$(curl -s "https://${BACKEND_HOST}/health" || true)
  echo "    Backend health: $HEALTH"
fi

FRONTEND_URL=$(oc get route frontend-route -n "$NAMESPACE" \
  -o jsonpath='https://{.spec.host}' 2>/dev/null || true)

echo ""
echo "==> Ambient installed successfully in $NAMESPACE"
echo "    Frontend: $FRONTEND_URL"
echo "    Image tag: $IMAGE_TAG"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "frontend_url=$FRONTEND_URL" >> "$GITHUB_OUTPUT"
  echo "namespace=$NAMESPACE" >> "$GITHUB_OUTPUT"
fi
