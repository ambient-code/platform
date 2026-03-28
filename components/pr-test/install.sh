#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${1:-}"
IMAGE_TAG="${2:-}"

SOURCE_NAMESPACE="${SOURCE_NAMESPACE:-ambient-code--runtime-int}"
CONFIG_NAMESPACE="${CONFIG_NAMESPACE:-ambient-code--config}"

REQUIRED_SOURCE_SECRETS=(
  ambient-vertex
  ambient-api-server
  ambient-api-server-db
  tenantaccess-ambient-control-plane-token
)

usage() {
  echo "Usage: $0 <namespace> <image-tag>"
  echo "  namespace:  e.g. ambient-code--pr-42"
  echo "  image-tag:  e.g. pr-42"
  echo ""
  echo "Optional environment variables:"
  echo "  SOURCE_NAMESPACE   Namespace to copy secrets from (default: ambient-code--runtime-int)"
  echo "  CONFIG_NAMESPACE   Namespace containing lock ConfigMaps (default: ambient-code--config)"
  exit 1
}

[[ -z "$NAMESPACE" || -z "$IMAGE_TAG" ]] && usage

PR_ID=$(echo "$NAMESPACE" | grep -oE 'pr-[0-9]+')

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OVERLAY_DIR="$REPO_ROOT/components/manifests/overlays/mpp-openshift"

copy_secret() {
  local name="$1"
  echo "    Copying secret: $name"
  oc get secret "$name" -n "$SOURCE_NAMESPACE" -o json \
    | python3 -c "
import json, sys
s = json.load(sys.stdin)
del s['metadata']['namespace']
del s['metadata']['resourceVersion']
del s['metadata']['uid']
del s['metadata']['creationTimestamp']
s['metadata'].pop('ownerReferences', None)
s['metadata'].pop('annotations', None)
s.pop('status', None)
print(json.dumps(s))
" | oc apply -n "$NAMESPACE" -f -
}

echo "==> Installing Ambient into $NAMESPACE with images tagged $IMAGE_TAG"

echo "==> Step 1: Verifying required secrets exist in $SOURCE_NAMESPACE"
FAILED=0
for secret in "${REQUIRED_SOURCE_SECRETS[@]}"; do
  if oc get secret "$secret" -n "$SOURCE_NAMESPACE" &>/dev/null 2>&1; then
    echo "    Secret OK: $secret"
  else
    echo "ERROR: Required secret missing from $SOURCE_NAMESPACE: $secret"
    FAILED=1
  fi
done
[[ $FAILED -eq 1 ]] && exit 1

echo "==> Step 2: Copying secrets from $SOURCE_NAMESPACE"
for secret in "${REQUIRED_SOURCE_SECRETS[@]}"; do
  copy_secret "$secret"
done

echo "==> Step 3: Deploying mpp-openshift overlay with image tag $IMAGE_TAG"
TMPDIR=$(mktemp -d)
cp -r "$OVERLAY_DIR/." "$TMPDIR/"
trap "rm -rf $TMPDIR" EXIT

pushd "$TMPDIR" > /dev/null

python3 - "$NAMESPACE" "$IMAGE_TAG" << 'PYEOF'
import sys, re

namespace, tag = sys.argv[1], sys.argv[2]
kfile = "kustomization.yaml"
text = open(kfile).read()

text = re.sub(r'(^namespace:\s*).*', r'\g<1>' + namespace, text, flags=re.MULTILINE)
if not re.search(r'^namespace:', text, re.MULTILINE):
    text = "namespace: " + namespace + "\n" + text

for repo in ("vteam_api_server", "vteam_control_plane"):
    text = re.sub(
        r'(- name: quay\.io/ambient_code/' + repo + r':latest\n\s+newName:.*\n\s+newTag:\s*).*',
        r'\g<1>' + tag, text,
    )
    text = re.sub(
        r'(- name: quay\.io/ambient_code/' + repo + r'\n\s+newTag:\s*).*',
        r'\g<1>' + tag, text,
    )

open(kfile, 'w').write(text)
PYEOF

FILTER_SCRIPT="$TMPDIR/filter.py"
cat > "$FILTER_SCRIPT" << 'PYEOF'
import sys, re, os

namespace = os.environ['NAMESPACE']
pr_id = os.environ['PR_ID']

for doc in sys.stdin.read().split('\n---\n'):
    doc = doc.strip()
    if not doc:
        continue
    kind_m = re.search(r'^kind:\s*(\S+)', doc, re.MULTILINE)
    if not kind_m:
        continue
    kind = kind_m.group(1)
    if kind == 'Route':
        if 'labels:' not in doc:
            doc = re.sub(r'(metadata:)', r'\1\n  labels:', doc, count=1)
        if 'paas.redhat.com/appcode' not in doc:
            doc = re.sub(r'(  labels:)', r'\1\n    paas.redhat.com/appcode: AMBC-001', doc, count=1)
        doc = re.sub(
            r'(  host:\s*).*',
            lambda m: m.group(1) + f'ambient-api-server-{namespace}.internal-router-shard.mpp-w2-preprod.cfln.p1.openshiftapps.com',
            doc,
        )
    print('---')
    print(doc)
PYEOF

oc kustomize . \
  | NAMESPACE="$NAMESPACE" PR_ID="$PR_ID" \
    python3 "$FILTER_SCRIPT" \
  | oc apply -n "$NAMESPACE" -f -

popd > /dev/null

echo "==> Step 4: Patching control-plane service URLs and kubeconfig"
oc set env deployment/ambient-control-plane -n "$NAMESPACE" \
  AMBIENT_API_SERVER_URL="http://ambient-api-server.${NAMESPACE}.svc:8000" \
  AMBIENT_GRPC_SERVER_ADDR="ambient-api-server.${NAMESPACE}.svc:9000" \
  CP_RUNTIME_NAMESPACE="$NAMESPACE"

KUBE_HOST=$(oc whoami --show-server)
KUBE_CA=$(oc get secret tenantaccess-ambient-control-plane-token -n "$NAMESPACE" \
  -o jsonpath='{.data.ca\.crt}')

python3 - << PYEOF
import subprocess, base64, json, os

kube_host = os.environ.get('KUBE_HOST', '').strip() or """$KUBE_HOST""".strip()
kube_ca   = os.environ.get('KUBE_CA',   '').strip() or """$KUBE_CA""".strip()
namespace = """$NAMESPACE""".strip()

kubeconfig = (
    "apiVersion: v1\n"
    "kind: Config\n"
    "clusters:\n"
    "- name: cluster\n"
    "  cluster:\n"
    f"    server: {kube_host}\n"
    f"    certificate-authority-data: {kube_ca}\n"
    "users:\n"
    "- name: ambient-control-plane\n"
    "  user:\n"
    "    tokenFile: /var/run/secrets/project-kube/token\n"
    "contexts:\n"
    "- name: default\n"
    "  context:\n"
    "    cluster: cluster\n"
    "    user: ambient-control-plane\n"
    "current-context: default\n"
)

secret = {
    "apiVersion": "v1",
    "kind": "Secret",
    "metadata": {"name": "ambient-control-plane-kubeconfig", "namespace": namespace},
    "data": {"kubeconfig": base64.b64encode(kubeconfig.encode()).decode()},
}

import tempfile
with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
    json.dump(secret, f)
    fname = f.name

r = subprocess.run(["oc", "apply", "-f", fname], capture_output=True, text=True)
print(r.stdout.strip()); print(r.stderr.strip())
os.unlink(fname)
PYEOF

oc set volume deployment/ambient-control-plane -n "$NAMESPACE" \
  --add --name=kubeconfig \
  --type=secret \
  --secret-name=ambient-control-plane-kubeconfig \
  --mount-path=/var/run/secrets/kubeconfig \
  --overwrite 2>&1 | grep -v "^$" || true

oc set env deployment/ambient-control-plane -n "$NAMESPACE" \
  KUBECONFIG=/var/run/secrets/kubeconfig/kubeconfig

echo "==> Step 5: Waiting for rollouts"
for deploy in ambient-api-server-db ambient-api-server ambient-control-plane; do
  echo "    Waiting for $deploy..."
  oc rollout status deployment/"$deploy" -n "$NAMESPACE" --timeout=300s
done

echo "==> Step 6: Verifying health"
API_HOST=$(oc get route ambient-api-server -n "$NAMESPACE" \
  -o jsonpath='{.spec.host}' 2>/dev/null || true)

if [[ -z "$API_HOST" ]]; then
  echo "ERROR: ambient-api-server route not found in $NAMESPACE"
  exit 1
fi

HEALTH=$(curl -fsS --connect-timeout 5 --max-time 20 \
  --retry 3 --retry-all-errors "https://${API_HOST}/api/ambient" 2>&1 || true)
echo "    API server: ${HEALTH:-<no response>}"

echo ""
echo "==> Ambient installed successfully in $NAMESPACE"
echo "    API server: https://${API_HOST}"
echo "    Image tag:  $IMAGE_TAG"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "api_server_url=https://${API_HOST}" >> "$GITHUB_OUTPUT"
  echo "namespace=$NAMESPACE" >> "$GITHUB_OUTPUT"
fi
