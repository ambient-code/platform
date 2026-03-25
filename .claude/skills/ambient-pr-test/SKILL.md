---
name: ambient-pr-test
description: >-
  End-to-end workflow for testing a pull request against the MPP dev cluster.
  Verifies quay.io images exist for the PR, provisions an ephemeral TenantNamespace,
  deploys Ambient with PR images using the ambient skill, runs E2E tests, and tears
  down the namespace. Use when validating a PR on the dev-spoke-aws-us-east-1 cluster.
---

# Ambient PR Test Skill

You are an expert in running ephemeral PR validation environments on the Ambient Code MPP dev cluster. This skill orchestrates the full lifecycle: image verification → namespace provisioning → Ambient deployment → E2E test → teardown.

> This skill calls the **ambient** skill for deployment steps. Read `.claude/skills/ambient/SKILL.md` for deployment detail.

> For namespace provisioning mechanics, see `components/pr-test/SPEC.md`.

---

## Cluster Context

- **Cluster:** `dev-spoke-aws-us-east-1`
- **Config namespace:** `ambient-code--config`
- **Namespace pattern:** `ambient-code--<PR_NUMBER>`
- **Instance ID pattern:** `pr-<PR_NUMBER>-<branch-slug>`
- **Image tag pattern:** `quay.io/ambient_code/vteam_*:pr-<PR_NUMBER>-amd64`

---

## Full Workflow

```
1. Derive instance-id from PR number + branch name
2. Verify all 7 component images exist in quay for the PR tag
3. Check S0.x capacity (max 5 concurrent instances)
4. Apply TenantNamespace CR → await namespace Active (~10s)
5. Deploy Ambient using ambient skill with PR image tag
6. Verify installation health
7. Run E2E tests
8. Teardown: delete TenantNamespace CR → await namespace gone
```

---

## Step 1: Derive Instance ID

```bash
PR_NUMBER=<number>
BRANCH=$(gh pr view $PR_NUMBER --json headRefName -q .headRefName)

# Slugify: lowercase, replace non-alphanumeric with hyphens, truncate
BRANCH_SLUG=$(echo "$BRANCH" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-\|-$//g')

# Max 63 chars for full namespace name: "ambient-code--" (14) + instance-id (max 49)
INSTANCE_ID="pr-${PR_NUMBER}-${BRANCH_SLUG:0:45}"
NAMESPACE="ambient-code--${INSTANCE_ID}"
IMAGE_TAG="pr-${PR_NUMBER}-amd64"

echo "Instance:  $INSTANCE_ID"
echo "Namespace: $NAMESPACE"
echo "Image tag: $IMAGE_TAG"
```

---

## Step 2: Verify Images in Quay

All 7 component images must exist before provisioning. Use `skopeo` to inspect without pulling:

```bash
IMAGE_TAG="pr-${PR_NUMBER}-amd64"

IMAGES=(
  quay.io/ambient_code/vteam_frontend
  quay.io/ambient_code/vteam_backend
  quay.io/ambient_code/vteam_operator
  quay.io/ambient_code/vteam_claude_runner
  quay.io/ambient_code/vteam_state_sync
  quay.io/ambient_code/vteam_public_api
  quay.io/ambient_code/vteam_api_server
)

MISSING=()
for img in "${IMAGES[@]}"; do
  if ! skopeo inspect --no-creds "docker://${img}:${IMAGE_TAG}" &>/dev/null; then
    MISSING+=("${img}:${IMAGE_TAG}")
  fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
  echo "ERROR: Missing images in quay:"
  printf '  %s\n' "${MISSING[@]}"
  echo "Wait for the build workflow to complete on PR #${PR_NUMBER}."
  exit 1
fi

echo "All images verified in quay."
```

**If images are missing:** The `components-build-deploy.yml` workflow builds and pushes all images on every PR commit. Check the Actions tab for the PR — the build job must complete successfully before proceeding.

---

## Step 3: Check Capacity

```bash
ACTIVE=$(oc get tenantnamespace -n ambient-code--config \
  -l ambient-code/instance-type=s0x --no-headers 2>/dev/null | wc -l)

MAX_S0X_INSTANCES=${MAX_S0X_INSTANCES:-5}

if [ "$ACTIVE" -ge "$MAX_S0X_INSTANCES" ]; then
  echo "At capacity: $ACTIVE/$MAX_S0X_INSTANCES S0.x instances active."
  echo "Active instances:"
  oc get tenantnamespace -n ambient-code--config \
    -l ambient-code/instance-type=s0x -o name
  exit 1
fi

echo "Capacity OK: $ACTIVE/$MAX_S0X_INSTANCES instances active."
```

---

## Step 4: Provision TenantNamespace

```bash
cat <<EOF | oc apply -f -
apiVersion: tenant.paas.redhat.com/v1alpha1
kind: TenantNamespace
metadata:
  labels:
    tenant.paas.redhat.com/namespace-type: build
    tenant.paas.redhat.com/tenant: ambient-code
    ambient-code/instance-type: s0x
  name: ${INSTANCE_ID}
  namespace: ambient-code--config
spec:
  network:
    security-zone: internal
  type: build
EOF
```

### Await Namespace Active

```bash
READY_TIMEOUT=${READY_TIMEOUT:-60}
DEADLINE=$((SECONDS + READY_TIMEOUT))

echo "Waiting for namespace ${NAMESPACE} to become Active..."
while [ $SECONDS -lt $DEADLINE ]; do
  STATUS=$(oc get namespace "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
  if [ "$STATUS" == "Active" ]; then
    echo "Namespace ${NAMESPACE} is Active."
    break
  fi
  echo "  status=${STATUS:-NotFound}, waiting..."
  sleep 3
done

if [ "$STATUS" != "Active" ]; then
  echo "ERROR: Namespace did not become Active within ${READY_TIMEOUT}s."
  exit 1
fi
```

---

## Step 5: Deploy Ambient

Follow the **ambient** skill (`SKILL.md`) with:
- `NAMESPACE=$NAMESPACE`
- `IMAGE_TAG=$IMAGE_TAG`

Key steps (summarized):
1. Apply CRDs and RBAC (idempotent, cluster-scoped)
2. Create all required secrets in `$NAMESPACE`
3. Deploy production overlay with `kustomize edit set image` overrides
4. Patch operator ConfigMap with PR image tags for runner pods
5. Patch agent registry ConfigMap with PR image tags
6. Wait for all rollouts

For the operator ConfigMap, `USE_VERTEX=0` and set `ANTHROPIC_API_KEY` via the `ambient-runner-secrets` secret.

---

## Step 6: Verify Installation

```bash
oc get pods -n $NAMESPACE
oc get route -n $NAMESPACE

BACKEND_HOST=$(oc get route backend-route -n $NAMESPACE -o jsonpath='{.spec.host}')
curl -sk https://$BACKEND_HOST/health
```

Expected: `{"status":"healthy"}`

---

## Step 7: Run E2E Tests

```bash
FRONTEND_URL="https://$(oc get route frontend-route -n $NAMESPACE -o jsonpath='{.spec.host}')"
TEST_TOKEN=$(oc get secret test-user-token -n $NAMESPACE \
  -o jsonpath='{.data.token}' 2>/dev/null | base64 -d || echo "")

cd e2e
CYPRESS_BASE_URL="$FRONTEND_URL" \
CYPRESS_TEST_TOKEN="$TEST_TOKEN" \
CYPRESS_ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  npx cypress run --browser chrome
```

---

## Step 8: Teardown

Always run teardown, even on failure.

```bash
oc delete tenantnamespace "$INSTANCE_ID" -n ambient-code--config --ignore-not-found=true

DELETE_TIMEOUT=${DELETE_TIMEOUT:-120}
DEADLINE=$((SECONDS + DELETE_TIMEOUT))

echo "Waiting for namespace ${NAMESPACE} to be deleted..."
while [ $SECONDS -lt $DEADLINE ]; do
  if ! oc get namespace "$NAMESPACE" &>/dev/null; then
    echo "Namespace ${NAMESPACE} deleted."
    break
  fi
  sleep 5
done
```

The tenant operator handles namespace deletion via finalizers. Do not `oc delete namespace` directly.

---

## Scripts

The above steps are implemented as scripts in `components/pr-test/`:

| Script | Purpose |
|--------|---------|
| `components/pr-test/provision.sh create <PR_NUMBER>` | Steps 3–4: capacity check + TenantNamespace CR |
| `components/pr-test/provision.sh destroy <PR_NUMBER>` | Step 8: teardown |
| `components/pr-test/install.sh <namespace> <image-tag>` | Step 5: full Ambient deploy |

---

## GitHub Actions Integration

The workflow `.github/workflows/pr-e2e-openshift.yml` automates this entire flow:

```
PR push
  → components-build-deploy.yml builds + pushes all images :pr-N-amd64
  → pr-e2e-openshift.yml triggers on workflow_run completion
      job: provision  → runs provision.sh create
      job: install    → runs install.sh
      job: e2e        → runs cypress
      job: teardown   → always runs provision.sh destroy

PR closed
  → pr-namespace-cleanup.yml → provision.sh destroy (safety net)
```

Required GitHub Actions secrets:
- `TEST_OPENSHIFT_SERVER` — API URL of dev-spoke-aws-us-east-1
- `TEST_OPENSHIFT_TOKEN` — ServiceAccount token with tenant-admin on ambient-code--config
- `ANTHROPIC_API_KEY` — for runner pods in test instances
- `QUAY_USERNAME` / `QUAY_PASSWORD` — already present for image builds

---

## Listing Active Instances

```bash
oc get tenantnamespace -n ambient-code--config \
  -l ambient-code/instance-type=s0x \
  -o custom-columns='NAME:.metadata.name,AGE:.metadata.creationTimestamp'
```

---

## Troubleshooting

### Images not found in quay
Build workflow did not complete or failed. Check Actions → `Build and Push Component Docker Images` for the PR. All components are built on every PR push.

### TenantNamespace not becoming Active
```bash
oc describe tenantnamespace $INSTANCE_ID -n ambient-code--config
oc get events -n ambient-code--config --sort-by='.lastTimestamp' | tail -20
```

### Namespace exists but pods won't schedule
```bash
oc get nodes
oc describe namespace $NAMESPACE
oc get resourcequota -n $NAMESPACE
```

MPP may enforce resource quotas on `build` type namespaces.

### JWT errors in ambient-api-server
The production overlay configures JWT against Red Hat SSO. For ephemeral test instances, disable JWT validation by patching the deployment:
```bash
oc set env deployment/ambient-api-server -n $NAMESPACE \
  ENABLE_JWT=false
oc rollout restart deployment/ambient-api-server -n $NAMESPACE
```
