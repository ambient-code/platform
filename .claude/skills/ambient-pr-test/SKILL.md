---
name: ambient-pr-test
description: >-
  End-to-end workflow for testing a pull request against the MPP dev cluster.
  Builds and pushes images, provisions an ephemeral TenantNamespace, deploys
  Ambient (mpp-openshift overlay), and tears down. Invoke with a PR URL.
---

# Ambient PR Test Skill

You are an expert in running ephemeral PR validation environments on the Ambient Code MPP dev cluster. This skill orchestrates the full lifecycle: build → namespace provisioning → Ambient deployment → teardown.

**Invoke this skill with a PR URL:**
```
with .claude/skills/ambient-pr-test  https://github.com/ambient-code/platform/pull/1005
```

Optional modifiers the user may specify:
- **`--keep-alive`** — do not tear down after the workflow; leave the instance online for human access
- **`provision-only`** / **`deploy-only`** / **`teardown-only`** — run a single phase instead of the full workflow

> **Overlay:** `components/manifests/overlays/mpp-openshift/` — api-server, control-plane, PostgreSQL only. No frontend, backend, operator, public-api, or CRDs.
> **Spec:** `components/manifests/overlays/mpp-openshift/README.md` — bootstrap steps, secret requirements, architecture.

Scripts in `components/pr-test/` implement all steps. Prefer them over inline commands.

---

## Cluster Context

- **Cluster:** `dev-spoke-aws-us-east-1` (context: `ambient-code--ambient-s2/...`)
- **Config namespace:** `ambient-code--config`
- **Namespace pattern:** `ambient-code--<instance-id>`
- **Instance ID pattern:** `pr-<PR_NUMBER>`
- **Image tag pattern:** `quay.io/ambient_code/vteam_*:pr-<PR_NUMBER>`

### Permissions

User tokens (`oc whoami -t`) do **not** have cluster-admin. `install.sh` uses the user token for the kustomize apply — the PR namespace's RBAC is set up by the tenant operator when the TenantNamespace CR is created. ClusterRoles and ClusterRoleBindings in the overlay (e.g. `ambient-control-plane-project-namespaces`) require cluster-admin to apply once; they are already in place on `dev-spoke-aws-us-east-1`.

### Namespace Type

PR test namespaces must be provisioned as `type: runtime`. Build namespaces cannot create Routes — the route admission webhook panics in `build` namespaces.

### No CRDs Required

The mpp-openshift overlay does **not** use Kubernetes CRDs (`agenticsessions`, `projectsettings`). The control plane manages sessions via the ambient-api-server REST/gRPC API, not via K8s custom resources.

### TenantNamespace Ready Condition

This cluster's tenant operator does not emit `Ready` conditions on `TenantNamespace.status.conditions`. `provision.sh` accepts `lastSuccessfulReconciliationTimestamp` as a sufficient signal that the namespace is ready.

---

## Full Workflow

```
0. Build: always run build.sh to build and push images tagged pr-<PR_NUMBER>
1. Derive instance-id from PR number
2. Provision: bash components/pr-test/provision.sh create <instance-id>
3. Deploy:    bash components/pr-test/install.sh <namespace> <image-tag>
4. Teardown:  bash components/pr-test/provision.sh destroy <instance-id>
             (skip if --keep-alive)
```

Phases can be run individually — see **Individual Phases** below.

---

## Step 0: Build and Push Images

Always run `build.sh` — CI may skip builds when no component source files changed (e.g. sync/merge branches), so never rely on CI to have pushed images:
```bash
bash components/pr-test/build.sh https://github.com/ambient-code/platform/pull/1005
```

This builds and pushes 3 images tagged `pr-<PR_NUMBER>`:
- `quay.io/ambient_code/vteam_api_server:pr-<PR_NUMBER>`
- `quay.io/ambient_code/vteam_control_plane:pr-<PR_NUMBER>`
- `quay.io/ambient_code/vteam_claude_runner:pr-<PR_NUMBER>`

Builds 3 images: `vteam_api_server`, `vteam_control_plane`, `vteam_claude_runner`.

| Variable | Default | Purpose |
|----------|---------|---------|
| `REGISTRY` | `quay.io/ambient_code` | Registry prefix |
| `PLATFORM` | `linux/amd64` | Build platform |
| `CONTAINER_ENGINE` | `docker` | `docker` or `podman` |

---

## Step 1: Derive Instance ID

```bash
PR_URL="https://github.com/ambient-code/platform/pull/1005"
PR_NUMBER=$(echo "$PR_URL" | grep -oE '[0-9]+$')

INSTANCE_ID="pr-${PR_NUMBER}"
NAMESPACE="ambient-code--${INSTANCE_ID}"
IMAGE_TAG="pr-${PR_NUMBER}"
```

---

## Step 2: Provision Namespace

```bash
bash components/pr-test/provision.sh create "$INSTANCE_ID"
```

Applies the `TenantNamespace` CR to `ambient-code--config`, waits for the namespace to become Active (~10–30s). Uses an atomic ConfigMap lock to prevent concurrent slot collisions; capacity capped at 5 concurrent instances.

---

## Step 3: Deploy Ambient

```bash
bash components/pr-test/install.sh "$NAMESPACE" "$IMAGE_TAG"
```

What `install.sh` does:
1. Verifies secrets exist in `ambient-code--runtime-int`: `ambient-vertex`, `ambient-api-server`, `ambient-api-server-db`, `tenantaccess-ambient-control-plane-token`
2. Copies those secrets to the PR namespace
3. Copies `mpp-openshift` overlay to a tmpdir, sets namespace and image tags, applies via `oc kustomize | oc apply`
4. Waits for rollouts: `ambient-api-server-db`, `ambient-api-server`, `ambient-control-plane`
5. Smoke-checks `GET /api/ambient` on the api-server Route

Deployed components:
- `ambient-api-server` — REST + gRPC API
- `ambient-api-server-db` — PostgreSQL (in-cluster, `emptyDir` storage)
- `ambient-control-plane` — gRPC fan-out, session orchestration, runner pod lifecycle

---

## Step 4: Teardown

Always run teardown after automated workflows, even on failure.

```bash
bash components/pr-test/provision.sh destroy "$INSTANCE_ID"
```

Deletes the `TenantNamespace` CR and waits for the namespace to be gone. Do not `oc delete namespace` directly — the tenant operator handles deletion via finalizers.

**`--keep-alive`**: skip teardown and leave the instance running. Use when:
- A human needs to log in and manually test the deployment
- Debugging a failure and the environment needs to stay up

When `--keep-alive` is set, print the API server URL prominently and remind the user to tear down manually:
```bash
echo "Instance is LIVE — tear down when finished:"
echo "  bash components/pr-test/provision.sh destroy $INSTANCE_ID"
```

---

## Individual Phases

When the user specifies a single phase, run only that step (always derive instance ID first).

**`provision-only`**
```bash
bash components/pr-test/provision.sh create "$INSTANCE_ID"
```
Use when: pre-provisioning before a delayed deploy, or re-provisioning after the namespace was manually deleted.

**`deploy-only`**
```bash
bash components/pr-test/install.sh "$NAMESPACE" "$IMAGE_TAG"
```
Confirm the namespace exists before running:
```bash
oc get namespace "$NAMESPACE" 2>/dev/null || echo "ERROR: namespace not found — provision first"
```
Use when: namespace already exists and you want to (re-)deploy without reprovisioning.

**`teardown-only`**
```bash
bash components/pr-test/provision.sh destroy "$INSTANCE_ID"
```
Use when: cleaning up a `--keep-alive` instance, or destroying after a failed deploy.

---

## Listing Active Instances

```bash
oc get tenantnamespace -n ambient-code--config \
  -l ambient-code/instance-type=s0x \
  -o custom-columns='NAME:.metadata.name,AGE:.metadata.creationTimestamp'
```

---

## Troubleshooting

### provision.sh times out waiting for Ready

This cluster's tenant operator does not emit `Ready` conditions. Check if the namespace is Active:
```bash
oc get namespace ambient-code--pr-NNN -o jsonpath='{.status.phase}'
oc get tenantnamespace pr-NNN -n ambient-code--config -o jsonpath='{.status}'
```
If `namespace.phase=Active` and `lastSuccessfulReconciliationTimestamp` is set, the namespace is ready — provision.sh should have exited successfully (it accepts `lastSuccessfulReconciliationTimestamp` as the ready signal).

### install.sh — secret missing

Required secrets must exist in `ambient-code--runtime-int`. If missing:
```bash
oc get secret ambient-api-server -n ambient-code--runtime-int
oc get secret ambient-api-server-db -n ambient-code--runtime-int
oc get secret tenantaccess-ambient-control-plane-token -n ambient-code--runtime-int
oc get secret ambient-vertex -n ambient-code--runtime-int
```

### Route host — wrong domain / 503

The filter script in `install.sh` rewrites the Route host to:
```
ambient-api-server-<namespace>.internal-router-shard.mpp-w2-preprod.cfln.p1.openshiftapps.com
```
The Route uses `shard: internal` and `tls: termination: edge` — matching the `ambient-code--runtime-int` production install. The `router-default` (`.apps.` domain) does not successfully route to these namespaces; only `internal-router-shard` works.

The internal hostname is not publicly DNS-resolvable. Access requires OCM tunnel or VPN (same as `runtime-int`). `acpctl login` works with this URL when the user has OCM tunnel active.

Neither the user token nor the ArgoCD SA token (`tenantaccess-argocd-account`) can **update** Routes in PR namespaces after creation — only create. If the Route is wrong, destroy and re-provision.

### Control plane can't reach api-server

`install.sh` Step 4 automatically patches `AMBIENT_API_SERVER_URL` and `AMBIENT_GRPC_SERVER_ADDR` to point at the PR namespace's api-server (the overlay hardcodes `ambient-code--runtime-int`). If the control plane still can't connect, verify the patch applied:
```bash
oc get deployment ambient-control-plane -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.containers[0].env}' | python3 -m json.tool | grep AMBIENT
```

### Build fails

Ensure `docker` (or `podman`) is logged in to `quay.io/ambient_code`:
```bash
docker login quay.io
```

### Images not found

Either `build.sh` was not run or the CI build workflow failed. Check Actions → `Build and Push Component Docker Images` for the PR.

### Runner pods can't reach external hosts (Squid proxy)

The MPP cluster routes outbound traffic through a Squid proxy (`proxy.squi-001.prod.iad2.dc.redhat.com:3128`). The `runtime-int` deployments have `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` set in their pod specs, but runner pods spawned by the control plane did not inherit these.

**Fix (merged):** The control plane reads `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` from its own environment and injects them into both the runner container (`buildEnv()`) and the MCP sidecar container (`buildMCPSidecar()`). No manifest change needed — the CP's deployment already has the proxy vars; they now propagate automatically.

**Pattern:** When the CP needs to forward platform-level env vars to spawned pods, add the field to `ControlPlaneConfig` → `KubeReconcilerConfig` → `buildEnv()`/`buildMCPSidecar()`.

### JWT / UNAUTHENTICATED errors in api-server

The production overlay configures JWT against Red Hat SSO. For ephemeral test instances without SSO integration:
```bash
oc set env deployment/ambient-api-server -n "$NAMESPACE" \
  --containers=api-server \
  -- \
  # Remove --jwk-cert-file and --grpc-jwk-cert-url args to disable JWT validation
```
Or patch the args ConfigMap to remove the JWK flags and restart.
