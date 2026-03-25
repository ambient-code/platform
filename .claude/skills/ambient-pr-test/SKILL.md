---
name: ambient-pr-test
description: >-
  End-to-end workflow for testing a pull request against the MPP dev cluster.
  Provisions an ephemeral TenantNamespace, deploys Ambient with PR images,
  runs E2E tests, and tears down. Use when validating a PR on dev-spoke-aws-us-east-1.
---

# Ambient PR Test Skill

You are an expert in running ephemeral PR validation environments on the Ambient Code MPP dev cluster. This skill orchestrates the full lifecycle: namespace provisioning → Ambient deployment → E2E test → teardown.

> **Spec:** `components/pr-test/README.md` — TenantNamespace CR schema, naming rules, capacity parameters, RBAC, image tagging convention, provisioner contracts.
> **Deployment detail:** `.claude/skills/ambient/SKILL.md` — how to install Ambient into a namespace.

Scripts in `components/pr-test/` implement all steps below. Prefer them over inline commands.

---

## Cluster Context

- **Cluster:** `dev-spoke-aws-us-east-1`
- **Config namespace:** `ambient-code--config`
- **Namespace pattern:** `ambient-code--<instance-id>`
- **Instance ID pattern:** `pr-<PR_NUMBER>-<branch-slug>`
- **Image tag pattern:** `quay.io/ambient_code/vteam_*:pr-<PR_NUMBER>-amd64`

For naming rules and slug budget, see `components/pr-test/README.md` § Instance Naming Convention.

---

## Full Workflow

```
1. Derive instance-id from PR number + branch name
2. Verify all 7 component images exist in quay for the PR tag
3. Provision namespace: bash components/pr-test/provision.sh create <instance-id>
4. Deploy Ambient: bash components/pr-test/install.sh <namespace> <image-tag>
5. Run E2E tests
6. Teardown: bash components/pr-test/provision.sh destroy <instance-id>
```

---

## Step 1: Derive Instance ID

```bash
PR_NUMBER=<number>
BRANCH=$(gh pr view $PR_NUMBER --json headRefName -q .headRefName)

SAFE_BRANCH=$(echo "$BRANCH" | tr '[:upper:]' '[:lower:]' \
  | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-\|-$//g' | cut -c1-64)

PR_LEN=${#PR_NUMBER}
SLUG_MAX=$(( 63 - 14 - 4 - PR_LEN ))
BRANCH_SLUG="${SAFE_BRANCH:0:$SLUG_MAX}"

INSTANCE_ID="pr-${PR_NUMBER}-${BRANCH_SLUG}"
NAMESPACE="ambient-code--${INSTANCE_ID}"
IMAGE_TAG="pr-${PR_NUMBER}-amd64"
```

See `components/pr-test/README.md` § Instance Naming Convention for the slug budget formula.

---

## Step 2: Verify Images in Quay

All 7 component images must exist before provisioning. Use `skopeo` to inspect without pulling:

```bash
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
  echo "ERROR: Missing images:"
  printf '  %s\n' "${MISSING[@]}"
  echo "Wait for the build workflow to complete on PR #${PR_NUMBER}."
  exit 1
fi
```

If images are missing: check Actions → `Build and Push Component Docker Images` for the PR. The build pushes all images on every PR commit — see `components/pr-test/README.md` § Image Tagging Convention.

---

## Step 3: Provision Namespace

```bash
bash components/pr-test/provision.sh create "$INSTANCE_ID"
```

This applies the `TenantNamespace` CR to `ambient-code--config` and waits for the namespace to become Active (~10s). For the CR schema and capacity rules, see `components/pr-test/README.md` §§ TenantNamespace CR, Capacity Management.

---

## Step 4: Deploy Ambient

```bash
bash components/pr-test/install.sh "$NAMESPACE" "$IMAGE_TAG"
```

This copies secrets from `ambient-code--runtime-int`, deploys the production overlay with PR image tags, patches operator and agent-registry ConfigMaps, and waits for all rollouts. See `.claude/skills/ambient/SKILL.md` for detail on each step.

---

## Step 5: Run E2E Tests

```bash
FRONTEND_URL="https://$(oc get route frontend-route -n $NAMESPACE -o jsonpath='{.spec.host}')"

cd e2e
CYPRESS_BASE_URL="$FRONTEND_URL" \
CYPRESS_ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  npx cypress run --browser chrome
```

---

## Step 6: Teardown

Always run teardown, even on failure.

```bash
bash components/pr-test/provision.sh destroy "$INSTANCE_ID"
```

Deletes the `TenantNamespace` CR and waits for the namespace to be gone. The tenant operator handles namespace deletion via finalizers — do not `oc delete namespace` directly.

---

## GitHub Actions Integration

The workflow `.github/workflows/pr-e2e-openshift.yml` automates this entire flow:

```
PR push
  → components-build-deploy.yml builds + pushes all images :pr-N-amd64
  → pr-e2e-openshift.yml triggers on workflow_run completion
      job: provision  → provision.sh create
      job: install    → install.sh
      job: e2e        → cypress
      job: teardown   → always: provision.sh destroy

PR closed
  → pr-namespace-cleanup.yml → provision.sh destroy (safety net)
```

Required secrets:
- `TEST_OPENSHIFT_SERVER` — API URL of dev-spoke-aws-us-east-1
- `TEST_OPENSHIFT_TOKEN` — ServiceAccount token with tenant-admin on `ambient-code--config`
- `ANTHROPIC_API_KEY` — for runner pods in test instances

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
Build workflow did not complete or failed. Check Actions → `Build and Push Component Docker Images` for the PR.

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

MPP enforces resource quotas on `build` type namespaces.

### JWT errors in ambient-api-server
The production overlay configures JWT against Red Hat SSO. For ephemeral test instances, disable JWT validation:
```bash
oc set env deployment/ambient-api-server -n $NAMESPACE ENABLE_JWT=false
oc rollout restart deployment/ambient-api-server -n $NAMESPACE
```
