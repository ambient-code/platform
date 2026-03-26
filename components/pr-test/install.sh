#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${1:-}"
IMAGE_TAG="${2:-}"

SOURCE_NAMESPACE="${SOURCE_NAMESPACE:-ambient-code--runtime-int}"
CONFIG_NAMESPACE="${CONFIG_NAMESPACE:-ambient-code--config}"
ARGOCD_TOKEN_SECRET="${ARGOCD_TOKEN_SECRET:-tenantaccess-argocd-account-token}"

usage() {
  echo "Usage: $0 <namespace> <image-tag>"
  echo "  namespace:  e.g. ambient-code--pr-42"
  echo "  image-tag:  e.g. pr-42-amd64"
  echo ""
  echo "Optional environment variables:"
  echo "  SOURCE_NAMESPACE     Namespace to copy secrets from (default: ambient-code--runtime-int)"
  echo "  CONFIG_NAMESPACE     Namespace containing ArgoCD token (default: ambient-code--config)"
  echo "  ARGOCD_TOKEN_SECRET  Secret name for ArgoCD SA token (default: tenantaccess-argocd-account-token)"
  exit 1
}

[[ -z "$NAMESPACE" || -z "$IMAGE_TAG" ]] && usage

PR_ID=$(echo "$NAMESPACE" | grep -oE 'pr-[0-9]+')
CLUSTER_DOMAIN=$(oc get route frontend-route -n "$SOURCE_NAMESPACE" \
  -o jsonpath='{.spec.host}' 2>/dev/null | sed 's/^[^.]*\.//' \
  || echo "apps.dev-osd-east-1.mxty.p1.openshiftapps.com")

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MANIFESTS_DIR="$REPO_ROOT/components/manifests"

copy_secret() {
  local name="$1"
  echo "    Copying secret: $name"
  oc get secret "$name" -n "$SOURCE_NAMESPACE" -o json \
    | jq "del(.metadata.namespace, .metadata.resourceVersion, .metadata.uid, .metadata.creationTimestamp, .metadata.ownerReferences, .metadata.annotations[\"kubectl.kubernetes.io/last-applied-configuration\"])" \
    | oc apply -n "$NAMESPACE" -f -
}

echo "==> Installing Ambient into $NAMESPACE with images tagged $IMAGE_TAG"
echo "    Cluster domain: $CLUSTER_DOMAIN"

echo "==> Step 1: Verifying cluster-scoped resources exist (CRDs, ClusterRoles)"
FAILED=0
for crd_resource in agenticsessions projectsettings; do
  if oc get "$crd_resource" -n "$NAMESPACE" &>/dev/null 2>&1; then
    echo "    CRD OK: $crd_resource"
  else
    echo "ERROR: CRD missing: $crd_resource — run: oc apply -k components/manifests/base/crds/"
    FAILED=1
  fi
done
for cr in agentic-operator ambient-frontend-auth ambient-project-admin ambient-project-edit ambient-project-view backend-api; do
  if oc get clusterrole "$cr" &>/dev/null 2>&1; then
    echo "    ClusterRole OK: $cr"
  else
    echo "ERROR: ClusterRole missing: $cr — run: oc apply -k components/manifests/base/rbac/"
    FAILED=1
  fi
done
[[ $FAILED -eq 1 ]] && exit 1

echo "==> Step 2: Copying secrets from $SOURCE_NAMESPACE"
copy_secret ambient-vertex
copy_secret ambient-api-server

echo "==> Step 3: Fetching ArgoCD SA token from $CONFIG_NAMESPACE"
ARGOCD_TOKEN=$(oc get secret "$ARGOCD_TOKEN_SECRET" -n "$CONFIG_NAMESPACE" \
  -o jsonpath='{.data.token}' | base64 -d)

echo "==> Step 4: Deploying production overlay with image tag $IMAGE_TAG"
TMPDIR=$(mktemp -d)
cp -r "$MANIFESTS_DIR/." "$TMPDIR/"
trap "rm -rf $TMPDIR" EXIT

TMPOVERLAY="$TMPDIR/overlays/production"
pushd "$TMPOVERLAY" > /dev/null

kustomize edit set namespace "$NAMESPACE"
kustomize edit set image \
  "quay.io/ambient_code/vteam_frontend:latest=quay.io/ambient_code/vteam_frontend:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_backend:latest=quay.io/ambient_code/vteam_backend:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_operator:latest=quay.io/ambient_code/vteam_operator:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_claude_runner:latest=quay.io/ambient_code/vteam_claude_runner:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_state_sync:latest=quay.io/ambient_code/vteam_state_sync:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_api_server:latest=quay.io/ambient_code/vteam_api_server:${IMAGE_TAG}" \
  "quay.io/ambient_code/vteam_public_api:latest=quay.io/ambient_code/vteam_public_api:${IMAGE_TAG}"

FILTER_SCRIPT="$TMPDIR/filter.py"
cat > "$FILTER_SCRIPT" << 'PYEOF'
import sys, re, os

namespace = os.environ['NAMESPACE']
pr_id = os.environ['PR_ID']
cluster_domain = os.environ['CLUSTER_DOMAIN']

SKIP_KINDS = {'Namespace'}

ROUTE_HOSTS = {
    'ambient-api-server-grpc': f'api-grpc-{pr_id}.{cluster_domain}',
    'ambient-api-server':      f'api-{pr_id}.{cluster_domain}',
    'frontend-route':          f'frontend-{pr_id}.{cluster_domain}',
    'backend-route':           f'backend-{pr_id}.{cluster_domain}',
    'public-api-route':        f'pubapi-{pr_id}.{cluster_domain}',
    'unleash-route':           f'unleash-{pr_id}.{cluster_domain}',
}

CRB_NS_RE = re.compile(r'(  namespace:\s*)ambient-code(\s*$)', re.MULTILINE)

for doc in sys.stdin.read().split('\n---\n'):
    doc = doc.strip()
    if not doc:
        continue
    kind_m = re.search(r'^kind:\s*(\S+)', doc, re.MULTILINE)
    if not kind_m:
        continue
    kind = kind_m.group(1)
    if kind in SKIP_KINDS:
        continue
    name_m = re.search(r'^  name:\s*(\S+)', doc, re.MULTILINE)
    name = name_m.group(1) if name_m else ''
    if kind == 'ClusterRoleBinding':
        doc = CRB_NS_RE.sub(r'\g<1>' + namespace + r'\g<2>', doc)
    if kind == 'PersistentVolumeClaim':
        if 'annotations:' not in doc:
            doc = re.sub(r'(metadata:)', r'\1\n  annotations:', doc, count=1)
        if 'kubernetes.io/reclaimPolicy' not in doc:
            doc = re.sub(r'(  annotations:)', r'\1\n    kubernetes.io/reclaimPolicy: Delete', doc, count=1)
        if 'labels:' not in doc:
            doc = re.sub(r'(metadata:)', r'\1\n  labels:', doc, count=1)
        if 'paas.redhat.com/appcode' not in doc:
            doc = re.sub(r'(  labels:)', r'\1\n    paas.redhat.com/appcode: AMBC-001', doc, count=1)
        if 'storageClassName' not in doc:
            doc = re.sub(r'(spec:)', r'\1\n  storageClassName: aws-ebs', doc, count=1)
    print('---')
    print(doc)
PYEOF

kustomize build . \
  | NAMESPACE="$NAMESPACE" PR_ID="$PR_ID" CLUSTER_DOMAIN="$CLUSTER_DOMAIN" \
    python3 "$FILTER_SCRIPT" \
  | oc apply --token="$ARGOCD_TOKEN" -n "$NAMESPACE" -f -

popd > /dev/null

echo "==> Step 5: Patching operator ConfigMap with PR image tags"
SOURCE_OPERATOR_CONFIG=$(oc get configmap operator-config -n "$SOURCE_NAMESPACE" -o json \
  | jq -r '.data | to_entries | map(select(.key | test("VERTEX|CLOUD_ML|ANTHROPIC|GOOGLE"))) | from_entries' \
  2>/dev/null || echo '{}')

VERTEX_PATCH=$(echo "$SOURCE_OPERATOR_CONFIG" | jq -c \
  --arg runner "quay.io/ambient_code/vteam_claude_runner:${IMAGE_TAG}" \
  --arg sync "quay.io/ambient_code/vteam_state_sync:${IMAGE_TAG}" \
  '. + {"AMBIENT_CODE_RUNNER_IMAGE": $runner, "STATE_SYNC_IMAGE": $sync}')

oc patch configmap operator-config -n "$NAMESPACE" --type=merge \
  -p "{\"data\": $VERTEX_PATCH}"

echo "==> Step 6: Patching agent registry ConfigMap with PR image tags"
REGISTRY=$(oc get configmap ambient-agent-registry -n "$NAMESPACE" \
  -o jsonpath='{.data.agent-registry\.json}' 2>/dev/null || echo "{}")

REGISTRY=$(echo "$REGISTRY" | sed \
  "s|quay.io/ambient_code/vteam_claude_runner[@:][^\"]*|quay.io/ambient_code/vteam_claude_runner:${IMAGE_TAG}|g")
REGISTRY=$(echo "$REGISTRY" | sed \
  "s|quay.io/ambient_code/vteam_state_sync[@:][^\"]*|quay.io/ambient_code/vteam_state_sync:${IMAGE_TAG}|g")

oc patch configmap ambient-agent-registry -n "$NAMESPACE" --type=merge \
  -p "{\"data\":{\"agent-registry.json\":$(echo "$REGISTRY" | jq -Rs .)}}"

echo "==> Step 7: Waiting for rollouts"
for deploy in backend-api frontend agentic-operator postgresql minio unleash public-api; do
  echo "    Waiting for $deploy..."
  oc rollout status deployment/$deploy -n "$NAMESPACE" --timeout=300s
done

echo "    Waiting for ambient-api-server-db..."
oc rollout status deployment/ambient-api-server-db -n "$NAMESPACE" --timeout=300s

echo "    Waiting for ambient-api-server..."
oc rollout status deployment/ambient-api-server -n "$NAMESPACE" --timeout=300s

echo "==> Step 8: Verifying health"
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
echo "    Frontend:  $FRONTEND_URL"
echo "    Image tag: $IMAGE_TAG"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "frontend_url=$FRONTEND_URL" >> "$GITHUB_OUTPUT"
  echo "namespace=$NAMESPACE" >> "$GITHUB_OUTPUT"
fi
