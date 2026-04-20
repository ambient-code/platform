---
name: ambient
description: >-
  Install and verify Ambient Code Platform on an OpenShift cluster using quay.io images.
  Use when deploying Ambient to any OpenShift namespace — production, ephemeral PR test
  instances, or developer clusters. Covers secrets, kustomize deploy, rollout verification,
  and troubleshooting.
---

# Ambient Installer Skill

You are an expert in deploying the Ambient Code Platform to OpenShift clusters. This skill covers everything needed to go from an empty namespace to a running Ambient installation using images from quay.io.

> **Developer registry override:** If you need to use images from the OpenShift internal registry instead of quay.io (e.g. for local dev builds), see `docs/internal/developer/local-development/openshift.md`.

---

## Platform Components

| Deployment | Image | Purpose |
|-----------|-------|---------|
| `backend-api` | `quay.io/ambient_code/vteam_backend` | Go REST API, manages K8s CRDs |
| `frontend` | `quay.io/ambient_code/vteam_frontend` | NextJS web UI |
| `agentic-operator` | `quay.io/ambient_code/vteam_operator` | Kubernetes operator |
| `ambient-api-server` | `quay.io/ambient_code/vteam_api_server` | Stateless API server |
| `ambient-api-server-db` | (postgres sidecar) | API server database |
| `public-api` | `quay.io/ambient_code/vteam_public_api` | External API gateway |
| `postgresql` | (upstream) | Unleash feature flag DB |
| `minio` | (upstream) | S3 object storage |
| `unleash` | (upstream) | Feature flag service |

Runner pods (`vteam_claude_runner`, `vteam_state_sync`) are spawned dynamically by the operator — they are not standing deployments.

---

## Prerequisites

- `oc` CLI installed and logged in to the target cluster
- `kustomize` installed (`curl -s https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh | bash`)
- Target namespace already exists and is Active
- Quay.io images are accessible from the cluster (public repos or image pull secret in place)

---

## Step 1: Apply CRDs and RBAC (cluster-scoped, once per cluster)

```bash
oc apply -k components/manifests/base/crds/
oc apply -k components/manifests/base/rbac/
```

These are idempotent. On a shared cluster where CRDs already exist from another namespace, this is safe to re-run.

---

## Step 2: Create Required Secrets

All secrets must exist **before** applying the kustomize overlay. The deployment will fail if any are missing.

```bash
NAMESPACE=<target-namespace>

oc create secret generic minio-credentials -n $NAMESPACE \
  --from-literal=root-user=<MINIO_ROOT_USER> \
  --from-literal=root-password=<MINIO_ROOT_PASSWORD>

oc create secret generic postgresql-credentials -n $NAMESPACE \
  --from-literal=db.host=postgresql \
  --from-literal=db.port=5432 \
  --from-literal=db.name=postgres \
  --from-literal=db.user=postgres \
  --from-literal=db.password=<POSTGRES_PASSWORD>

oc create secret generic unleash-credentials -n $NAMESPACE \
  --from-literal=database-url=postgres://postgres:<POSTGRES_PASSWORD>@postgresql:5432/unleash \
  --from-literal=database-ssl=false \
  --from-literal=admin-api-token='*:*.<UNLEASH_ADMIN_TOKEN>' \
  --from-literal=client-api-token=default:development.<UNLEASH_CLIENT_TOKEN> \
  --from-literal=frontend-api-token=default:development.<UNLEASH_FRONTEND_TOKEN> \
  --from-literal=default-admin-password=<UNLEASH_ADMIN_PASSWORD>

oc create secret generic github-app-secret -n $NAMESPACE \
  --from-literal=GITHUB_APP_ID="<GITHUB_APP_ID>" \
  --from-literal=GITHUB_PRIVATE_KEY="<GITHUB_PRIVATE_KEY>" \
  --from-literal=GITHUB_CLIENT_ID="<GITHUB_CLIENT_ID>" \
  --from-literal=GITHUB_CLIENT_SECRET="<GITHUB_CLIENT_SECRET>" \
  --from-literal=GITHUB_STATE_SECRET=<GITHUB_STATE_SECRET>
```

Use `--dry-run=client -o yaml | oc apply -f -` to make secret creation idempotent on re-runs.

### Anthropic API Key (required for runner pods)

```bash
oc create secret generic ambient-runner-secrets -n $NAMESPACE \
  --from-literal=ANTHROPIC_API_KEY=<key>
```

### Vertex AI (optional, instead of direct Anthropic)

```bash
oc create secret generic ambient-vertex -n $NAMESPACE \
  --from-file=ambient-code-key.json=/path/to/service-account-key.json
```

If using Vertex, set `USE_VERTEX=1` in the operator ConfigMap (see Step 4).

---

## Step 3: Deploy with Kustomize

### Scripted (preferred for ephemeral/PR namespaces)

`components/pr-test/install.sh` encapsulates Steps 2–6 into a single script. It copies secrets from the source namespace, deploys via a temp-dir kustomize overlay (no git working tree mutations), patches configmaps, and waits for rollouts:

```bash
bash components/pr-test/install.sh <namespace> <image-tag>
```

### Production deploy (`make deploy`)

For the production namespace (`ambient-code`), use:

```bash
make deploy
# calls components/manifests/deploy.sh — handles OAuth, restores kustomization after apply
```

`deploy.sh` mutates `kustomization.yaml` in-place and restores it post-apply. It also handles the OpenShift OAuth `OAuthClient` (requires cluster-admin). Use `make deploy` only for the canonical production namespace.

### Manual (for debugging or one-off namespaces)

Use a temp dir to avoid modifying the git working tree:

```bash
IMAGE_TAG=<tag>   # e.g. latest, pr-42-amd64, abc1234
NAMESPACE=<target-namespace>

TMPDIR=$(mktemp -d)
cp -r components/manifests/overlays/production/. "$TMPDIR/"
pushd "$TMPDIR"

kustomize edit set namespace $NAMESPACE

kustomize edit set image \
  quay.io/ambient_code/vteam_frontend:latest=quay.io/ambient_code/vteam_frontend:$IMAGE_TAG \
  quay.io/ambient_code/vteam_backend:latest=quay.io/ambient_code/vteam_backend:$IMAGE_TAG \
  quay.io/ambient_code/vteam_operator:latest=quay.io/ambient_code/vteam_operator:$IMAGE_TAG \
  quay.io/ambient_code/vteam_claude_runner:latest=quay.io/ambient_code/vteam_claude_runner:$IMAGE_TAG \
  quay.io/ambient_code/vteam_state_sync:latest=quay.io/ambient_code/vteam_state_sync:$IMAGE_TAG \
  quay.io/ambient_code/vteam_api_server:latest=quay.io/ambient_code/vteam_api_server:$IMAGE_TAG \
  quay.io/ambient_code/vteam_public_api:latest=quay.io/ambient_code/vteam_public_api:$IMAGE_TAG

oc apply -k . -n $NAMESPACE
popd
rm -rf "$TMPDIR"
```

---

## Step 4: Configure the Operator ConfigMap

The operator needs to know which runner images to spawn and whether to use Vertex AI:

```bash
NAMESPACE=<target-namespace>
IMAGE_TAG=<tag>

oc patch configmap operator-config -n $NAMESPACE --type=merge -p "{
  \"data\": {
    \"AMBIENT_CODE_RUNNER_IMAGE\": \"quay.io/ambient_code/vteam_claude_runner:$IMAGE_TAG\",
    \"STATE_SYNC_IMAGE\": \"quay.io/ambient_code/vteam_state_sync:$IMAGE_TAG\",
    \"USE_VERTEX\": \"0\",
    \"CLOUD_ML_REGION\": \"\",
    \"ANTHROPIC_VERTEX_PROJECT_ID\": \"\",
    \"GOOGLE_APPLICATION_CREDENTIALS\": \"\"
  }
}"
```

Also patch the agent registry ConfigMap so runner image refs point to the PR tag:

```bash
REGISTRY=$(oc get configmap ambient-agent-registry -n $NAMESPACE \
  -o jsonpath='{.data.agent-registry\.json}')

REGISTRY=$(echo "$REGISTRY" | sed \
  "s|quay.io/ambient_code/vteam_claude_runner[@:][^\"]*|quay.io/ambient_code/vteam_claude_runner:$IMAGE_TAG|g")
REGISTRY=$(echo "$REGISTRY" | sed \
  "s|quay.io/ambient_code/vteam_state_sync[@:][^\"]*|quay.io/ambient_code/vteam_state_sync:$IMAGE_TAG|g")

oc patch configmap ambient-agent-registry -n $NAMESPACE --type=merge \
  -p "{\"data\":{\"agent-registry.json\":$(echo "$REGISTRY" | jq -Rs .)}}"
```

---

## Step 5: Wait for Rollout

```bash
NAMESPACE=<target-namespace>

for deploy in backend-api frontend agentic-operator postgresql minio unleash public-api; do
  oc rollout status deployment/$deploy -n $NAMESPACE --timeout=300s
done
```

`ambient-api-server-db` and `ambient-api-server` may take longer due to DB init:

```bash
oc rollout status deployment/ambient-api-server-db -n $NAMESPACE --timeout=300s
oc rollout status deployment/ambient-api-server -n $NAMESPACE --timeout=300s
```

---

## Step 6: Verify Installation

### Pod Status

```bash
oc get pods -n $NAMESPACE
```

Expected — all pods `Running`:
```
NAME                                  READY   STATUS    RESTARTS
agentic-operator-xxxxx                1/1     Running   0
ambient-api-server-xxxxx              1/1     Running   0
ambient-api-server-db-xxxxx           1/1     Running   0
backend-api-xxxxx                     1/1     Running   0
frontend-xxxxx                        2/2     Running   0
minio-xxxxx                           1/1     Running   0
postgresql-xxxxx                      1/1     Running   0
public-api-xxxxx                      1/1     Running   0
unleash-xxxxx                         1/1     Running   0
```

Frontend shows `2/2` because of the oauth-proxy sidecar in the production overlay.

### Routes

```bash
oc get route -n $NAMESPACE
```

### Health Check

```bash
BACKEND_HOST=$(oc get route backend-route -n $NAMESPACE -o jsonpath='{.spec.host}')
curl -s https://$BACKEND_HOST/health
```

Expected: `{"status":"healthy"}`

### Database Tables

```bash
oc exec deployment/ambient-api-server-db -n $NAMESPACE -- \
  psql -U ambient -d ambient_api_server -c "\dt"
```

Expected: 6 tables (events, migrations, project_settings, projects, sessions, users).

### API Server gRPC Streams

```bash
oc logs deployment/ambient-api-server -n $NAMESPACE --tail=20 | grep "gRPC stream"
```

Expected:
```
gRPC stream started /ambient.v1.ProjectService/WatchProjects
gRPC stream started /ambient.v1.SessionService/WatchSessions
```

### SDK Environment Setup

```bash
export AMBIENT_TOKEN="$(oc whoami -t)"
export AMBIENT_PROJECT="$(oc project -q)"
export AMBIENT_API_URL="$(oc get route public-api-route -n $NAMESPACE \
  --template='https://{{.spec.host}}')"
```

---

## Cross-Namespace Image Pull (Required for Runner Pods)

The operator creates runner pods in dynamically-created project namespaces. Those pods pull images from quay.io directly — no cross-namespace image access issue with quay. However, if you're using the OpenShift internal registry, grant pull access:

```bash
oc policy add-role-to-group system:image-puller system:serviceaccounts --namespace=$NAMESPACE
```

---

## Troubleshooting

### ImagePullBackOff

```bash
oc describe pod <pod-name> -n $NAMESPACE | grep -A5 "Events:"
```

- If pulling from quay.io: verify the tag exists (`skopeo inspect docker://quay.io/ambient_code/vteam_backend:<tag>`)
- If private: create an image pull secret and link it to the default service account

### API Server TLS Certificate Missing

```bash
oc annotate service ambient-api-server \
  service.beta.openshift.io/serving-cert-secret-name=ambient-api-server-tls \
  -n $NAMESPACE
sleep 15
oc rollout restart deployment/ambient-api-server -n $NAMESPACE
```

### JWT Configuration

Production uses Red Hat SSO JWKS (`--jwk-cert-url=https://sso.redhat.com/...`). For ephemeral test instances, JWT validation may need to be disabled or pointed at a different issuer. Check the `ambient-api-server-jwt-args-patch.yaml` in the production overlay and adjust as needed for non-production contexts.

### CrashLoopBackOff

```bash
oc logs deployment/<name> -n $NAMESPACE --tail=100
oc describe pod -l app=<name> -n $NAMESPACE
```

Common causes: missing secret, wrong DB credentials, missing ConfigMap key.

### Rollout Timeout

```bash
oc get events -n $NAMESPACE --sort-by='.lastTimestamp' | tail -20
```

---

## CLI Access

```bash
acpctl login \
  --url https://$(oc get route ambient-api-server -n $NAMESPACE -o jsonpath='{.spec.host}') \
  --token $(oc whoami -t)
```
