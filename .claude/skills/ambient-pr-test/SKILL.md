---
name: ambient-pr-test
description: >-
  End-to-end workflow for testing a pull request against the MPP dev cluster.
  Builds and pushes images, provisions an ephemeral TenantNamespace, deploys
  Ambient, runs E2E tests, and tears down. Invoke with a PR URL.
---

# Ambient PR Test Skill

You are an expert in running ephemeral PR validation environments on the Ambient Code MPP dev cluster. This skill orchestrates the full lifecycle: build → namespace provisioning → Ambient deployment → E2E test → teardown.

**Invoke this skill with a PR URL:**
```
with .claude/skills/ambient-pr-test  https://github.com/ambient-code/platform/pull/1005
```

> **Spec:** `components/pr-test/README.md` — TenantNamespace CR schema, naming rules, capacity parameters, RBAC, image tagging convention, provisioner contracts.
> **Deployment detail:** `.claude/skills/ambient/SKILL.md` — how to install Ambient into a namespace.

Scripts in `components/pr-test/` implement all steps below. Prefer them over inline commands.

---

## Cluster Context

- **Cluster:** `dev-spoke-aws-us-east-1`
- **Config namespace:** `ambient-code--config`
- **Namespace pattern:** `ambient-code--<instance-id>`
- **Instance ID pattern:** `pr-<PR_NUMBER>`
- **Image tag pattern:** `quay.io/ambient_code/vteam_*:pr-<PR_NUMBER>-amd64`

For naming rules and slug budget, see `components/pr-test/README.md` § Instance Naming Convention.

### Permissions

User tokens (`oc whoami -t`) do **not** have cluster-admin. `install.sh` uses the `tenantaccess-argocd-account-token` from `ambient-code--config` (the ArgoCD SA token) for the kustomize apply — it has cluster-admin and can create ClusterRoleBindings, PVCs, and all namespace-scoped resources.

- `oc get crd` at cluster scope → Forbidden for user token (expected) — `install.sh` probes via `oc get agenticsessions -n $NAMESPACE` instead
- CRDs and ClusterRoles must already exist — applied once by cluster-admin
- ClusterRoleBindings are patched by the filter script to point subjects at the PR namespace

### Namespace Type

PR test namespaces must be provisioned as `type: runtime` (not `build`). MPP `build` namespaces cannot create Routes — the route admission webhook panics on all Route creates in `build` namespaces.

---

## Full Workflow

```
0. Build and push images: bash components/pr-test/build.sh <pr-url>
1. Derive instance-id from PR number + branch name
2. Provision namespace: bash components/pr-test/provision.sh create <instance-id>
3. Deploy Ambient: bash components/pr-test/install.sh <namespace> <image-tag>
4. Run E2E tests
5. Teardown: bash components/pr-test/provision.sh destroy <instance-id>
```

---

## Step 0: Build and Push Images

```bash
bash components/pr-test/build.sh https://github.com/ambient-code/platform/pull/1005
```

Builds all 7 component images from the current checkout and pushes them to quay with the `pr-N-amd64` tag. Optional env vars:

| Variable | Default | Purpose |
|----------|---------|---------|
| `REGISTRY` | `quay.io/ambient_code` | Registry prefix |
| `PLATFORM` | `linux/amd64` | Build platform |
| `CONTAINER_ENGINE` | `docker` | `docker` or `podman` |

Skip this step if CI already pushed images (e.g. the PR's `Build and Push Component Docker Images` workflow completed successfully).

---

## Step 1: Derive Instance ID

```bash
PR_URL="https://github.com/ambient-code/platform/pull/1005"
PR_NUMBER=$(echo "$PR_URL" | grep -oE '[0-9]+$')

INSTANCE_ID="pr-${PR_NUMBER}"
NAMESPACE="ambient-code--${INSTANCE_ID}"
IMAGE_TAG="pr-${PR_NUMBER}-amd64"
```

---

## Step 2: Provision Namespace

```bash
bash components/pr-test/provision.sh create "$INSTANCE_ID"
```

This applies the `TenantNamespace` CR to `ambient-code--config` and waits for the namespace to become Active (~10s). For the CR schema and capacity rules, see `components/pr-test/README.md` §§ TenantNamespace CR, Capacity Management.

---

## Step 3: Deploy Ambient

```bash
bash components/pr-test/install.sh "$NAMESPACE" "$IMAGE_TAG"
```

This copies secrets from `ambient-code--runtime-int`, deploys the production overlay with PR image tags, patches operator and agent-registry ConfigMaps, and waits for all rollouts. See `.claude/skills/ambient/SKILL.md` for detail on each step.

---

## Step 4: Run E2E Tests

```bash
FRONTEND_URL="https://$(oc get route frontend-route -n $NAMESPACE -o jsonpath='{.spec.host}')"

cd e2e
CYPRESS_BASE_URL="$FRONTEND_URL" \
CYPRESS_ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  npx cypress run --browser chrome
```

---

## Step 5: Teardown

Always run teardown, even on failure.

```bash
bash components/pr-test/provision.sh destroy "$INSTANCE_ID"
```

Deletes the `TenantNamespace` CR and waits for the namespace to be gone. The tenant operator handles namespace deletion via finalizers — do not `oc delete namespace` directly.

---

## GitHub Actions Integration

The workflow `.github/workflows/pr-e2e-openshift.yml` automates steps 1–5 (build is handled by `components-build-deploy.yml`):

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

### Kustomize "no such file or directory" for `../../base`
The production overlay uses relative paths (`../../base`). Copying only the overlay directory into a tmpdir breaks these references. `install.sh` copies the entire `components/manifests/` tree into the tmpdir and runs kustomize from `overlays/production/` within it.

### CRD apply fails with Forbidden
This is expected when running as a user token (not cluster-admin). `install.sh` probes CRD presence via `oc get agenticsessions -n $NAMESPACE`. If that returns an error (not "No resources found"), CRDs are missing — ask a cluster-admin to apply them once.

### Route admission webhook — shard label
Routes require `paas.redhat.com/appcode: AMBC-001` label (injected by filter). Do **not** add `shard: internal` — that requires a host on the internal domain. Without a shard label OpenShift auto-assigns a host on the external domain. The previous nil-pointer panic in the route admission webhook was a cluster-side bug, now fixed.

### ClusterRoleBindings — using ArgoCD SA token
User tokens cannot create ClusterRoleBindings. `install.sh` fetches the `tenantaccess-argocd-account-token` secret from `ambient-code--config` and uses it for the full kustomize apply. This token has cluster-admin level access and can create ClusterRoleBindings. The Python filter script patches ClusterRoleBinding subjects from `ambient-code` to the PR namespace before applying.

### Build fails
Check that `docker` (or `podman`) is logged in to `quay.io/ambient_code` before running `build.sh`. Use `docker login quay.io` or set `CONTAINER_ENGINE=podman`.

### Images not found in quay
Either `build.sh` was not run, or the CI build workflow failed. Check Actions → `Build and Push Component Docker Images` for the PR.

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
