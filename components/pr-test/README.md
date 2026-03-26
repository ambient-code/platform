# Specification: Ephemeral PR Test Environments on MPP

**Interface:**
```
with .claude/skills/ambient-pr-test  https://github.com/ambient-code/platform/pull/1005
```
or directly:
```bash
bash components/pr-test/build.sh     <pr-url>      # build + push images
bash components/pr-test/provision.sh create <instance-id>
bash components/pr-test/install.sh   <namespace> <image-tag>
bash components/pr-test/provision.sh destroy <instance-id>
```

> **Operational how-to:** `.claude/skills/ambient-pr-test/SKILL.md` — step-by-step PR test workflow that references this spec.

## Purpose

This specification defines how Ambient Code creates and destroys ephemeral OpenShift namespaces for S0.x merge queue test instances. Each S0.x instance is a fully independent, shared-nothing installation of Ambient, used for integration testing of a single candidate branch before it merges to `main`.

This is an extension of Ambient's own functionality — the provisioner is part of the Ambient platform, not external tooling.

---

## Context

- **Platform:** Red Hat OpenShift (MPP — Managed Platform Plus)
- **Tenant:** `ambient-code`
- **Config namespace:** `ambient-code--config`
- **ArgoCD namespace:** `ambient-code--argocd`
- **Source namespace:** `ambient-code--runtime-int` (secrets and route domain derived from here)
- **Target cluster:** `dev-spoke-aws-us-east-1` (initially)
- **Namespace naming convention:** `ambient-code--<instance-id>`
- **Instance ID format:** `pr-<PR_NUMBER>` — PR number only, no branch slug
- **Resulting namespace:** `ambient-code--pr-1005`

---

## MPP Tenant API

The MPP tenant operator exposes these CRDs (`tenant.paas.redhat.com/v1alpha1`):

| CRD | Purpose |
|-----|---------|
| `TenantNamespace` | Provision a managed namespace |
| `TenantServiceAccount` | Create a SA with cluster-linking tokens |
| `TenantEgress` | Outbound CIDR/DNS egress network policy |
| `TenantNamespaceEgress` | Pod-level egress NetworkPolicy |
| `TenantGroup` | Group management |
| `TenantCredentialManagement` | Cluster credential linking (unstable) |
| `TenantOperatorConfig` / `TenantOperatorOptIn` | Operator configuration |

There is **no `TenantRoute`**. Routes are standard OpenShift `Route` objects applied into runtime namespaces.

---

## Service Exposure — Known Constraints

External access to PR namespace services is constrained by the following cluster-side limitations (verified on `dev-spoke-aws-us-east-1`):

### Route admission webhook panic
All new `Route` creates fail cluster-wide:
```
admission webhook "v1.route.openshift.io" denied the request:
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
```
- Affects all namespaces including `ambient-code--runtime-int`
- Existing routes (pre-bug) continue to work
- Same error visible in production ArgoCD app status
- **This is a cluster-side bug — report to MPP cluster admins**

### LoadBalancer subnet exhaustion
`Service type: LoadBalancer` fails with:
```
InvalidSubnet: Not enough IP space available in subnet-0e04e2925720142be.
ELB requires at least 8 free IP addresses in each subnet.
```
- AWS ELB provisioning blocked by subnet IP exhaustion
- **This is a cluster-side infrastructure issue — report to MPP cluster admins**

### Workaround: oc port-forward
For manual smoke testing only — not suitable for automated E2E:
```bash
oc port-forward svc/frontend-service 3000:3000 -n ambient-code--pr-1005 &
# then: open http://localhost:3000
```

---

## Mechanism

Namespaces are created by applying a `TenantNamespace` CR to the `ambient-code--config` namespace. The MPP tenant operator watches for these CRs and reconciles the actual namespace within ~10 seconds.

**No GitOps round-trip is required.** Direct `oc apply` by an authorized ServiceAccount is sufficient and appropriate for ephemeral instances.

---

## TenantNamespace CR

### Schema

```yaml
apiVersion: tenant.paas.redhat.com/v1alpha1
kind: TenantNamespace
metadata:
  labels:
    tenant.paas.redhat.com/namespace-type: runtime  # must be "runtime" — "build" blocks Route creation
    tenant.paas.redhat.com/tenant: ambient-code
    ambient-code/instance-type: s0x                 # for capacity counting
  name: <instance-id>                               # e.g. pr-1005
  namespace: ambient-code--config                   # always this namespace
spec:
  network:
    security-zone: internal
  type: runtime                                     # must be "runtime" — see note below
```

> **Important:** Use `type: runtime`, not `type: build`. MPP `build` namespaces block Route creation at the admission webhook. Even with the current cluster-side route webhook panic, future Route creates require `runtime` type.

### Verified Example

The following was applied and confirmed working on `dev-spoke-aws-us-east-1`:

```yaml
apiVersion: tenant.paas.redhat.com/v1alpha1
kind: TenantNamespace
metadata:
  labels:
    tenant.paas.redhat.com/namespace-type: runtime
    tenant.paas.redhat.com/tenant: ambient-code
    ambient-code/instance-type: s0x
  name: pr-1005
  namespace: ambient-code--config
spec:
  network:
    security-zone: internal
  type: runtime
```

Resulting namespace `ambient-code--pr-1005` was `Active` within 11 seconds with the following platform-injected labels:

```
tenant.paas.redhat.com/tenant: ambient-code
tenant.paas.redhat.com/namespace-type: build
pipeline.paas.redhat.com/realm: ambient-code
paas.redhat.com/secret-decryption: enabled
pod-security.kubernetes.io/audit: baseline
openshift-pipelines.tekton.dev/namespace-reconcile-version: 1.20.2
```

These labels are injected by the tenant operator — the provisioner does not need to set them.

---

## Provisioner Behavior

### Create

```
input:  instance-id  (e.g. "pr-123-feat-xyz")

1. Check current S0.x instance count:
     oc get tenantnamespace -n ambient-code--config \
       -l ambient-code/instance-type=s0x --no-headers | wc -l

2. If count >= MAX_S0X_INSTANCES:
     report "at capacity" and exit (do not block — queue or skip)

3. Apply TenantNamespace CR with name = <instance-id>
     label: ambient-code/instance-type=s0x   (for counting/listing)

4. Wait for status.conditions[type=Ready].status == "True"
     poll oc get tenantnamespace <instance-id> -n ambient-code--config
     timeout: 60s

5. Confirm namespace ambient-code--<instance-id> exists and is Active

output: namespace name  ("ambient-code--pr-123-feat-xyz")
```

### Destroy

```
input:  instance-id  (e.g. "pr-123-feat-xyz")

1. Delete TenantNamespace CR:
     oc delete tenantnamespace <instance-id> -n ambient-code--config

2. Confirm namespace ambient-code--<instance-id> is gone
     poll until NotFound or timeout: 120s

   Note: the tenant operator handles namespace deletion via finalizers.
   The provisioner does not delete the namespace directly.
```

---

## Capacity Management

A label `ambient-code/instance-type=s0x` must be applied to all ephemeral `TenantNamespace` CRs at creation time. This allows the provisioner to count active instances without scanning all tenant namespaces.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `MAX_S0X_INSTANCES` | 5 | Maximum concurrent S0.x instances |
| `READY_TIMEOUT` | 60s | Max wait for namespace Ready |
| `DELETE_TIMEOUT` | 120s | Max wait for namespace deletion |

These should be configurable via environment variables on the provisioner.

---

## Required RBAC

### User token limitations
User tokens (`oc whoami -t`) do **not** have cluster-admin. They cannot:
- Create `ClusterRoleBinding` objects (escalation prevention)
- List/get CRDs at cluster scope (`oc get crd` → Forbidden)
- Get cluster ingress config (`oc get ingresses.config.openshift.io` → Forbidden)

### ArgoCD SA token — cluster-admin
`install.sh` uses the ArgoCD service account token for the kustomize apply:

```bash
ARGOCD_TOKEN=$(oc get secret tenantaccess-argocd-account-token \
  -n ambient-code--config \
  -o jsonpath='{.data.token}' | base64 -d)

kustomize build . | python3 filter.py | oc apply --token="$ARGOCD_TOKEN" -n "$NAMESPACE" -f -
```

This token is the `TenantServiceAccount` created for ArgoCD cluster linking (see MPP cluster linking docs). It has cluster-admin and can create ClusterRoleBindings, PVCs, and all namespace-scoped resources.

### Provisioner RBAC (TenantNamespace management)
The ServiceAccount running `provision.sh` needs:

```yaml
rules:
  - apiGroups: ["tenant.paas.redhat.com"]
    resources: ["tenantnamespaces"]
    verbs: ["get", "list", "create", "delete", "watch"]
```

On `dev-spoke-aws-us-east-1` this is satisfied by a `TenantServiceAccount` with role `tenant-admin`. The existing `tenantserviceaccount-argocd.yaml` already carries `tenant-admin`.

### CRD presence detection
Because `oc get crd` is Forbidden for user tokens, `install.sh` probes CRD presence via namespace-scoped access:
```bash
oc get agenticsessions -n "$NAMESPACE"   # errors if CRD missing
oc get projectsettings -n "$NAMESPACE"
```

### Cluster domain derivation
Because `oc get ingresses.config.openshift.io cluster` is Forbidden, the cluster domain is derived from an existing route in the source namespace:
```bash
CLUSTER_DOMAIN=$(oc get route frontend-route -n "$SOURCE_NAMESPACE" \
  -o jsonpath='{.spec.host}' | sed 's/^[^.]*\.//')
```

---

## Instance Naming Convention

| Input | Instance ID | Resulting Namespace | Image Tag |
|-------|-------------|---------------------|-----------|
| PR #1005 | `pr-1005` | `ambient-code--pr-1005` | `pr-1005-amd64` |
| PR #42 | `pr-42` | `ambient-code--pr-42` | `pr-42-amd64` |

Rules:
- Instance ID is **PR number only** — no branch slug (avoids namespace name length issues)
- Lowercase, hyphens only — no underscores, no dots
- `ambient-code--pr-N` is well within the 63-character Kubernetes namespace limit

Derivation from PR URL:
```bash
PR_URL="https://github.com/ambient-code/platform/pull/1005"
PR_NUMBER=$(echo "$PR_URL" | grep -oE '[0-9]+$')
INSTANCE_ID="pr-${PR_NUMBER}"
NAMESPACE="ambient-code--${INSTANCE_ID}"
IMAGE_TAG="pr-${PR_NUMBER}-amd64"
```

---

## MPP Restricted Environment — Resource Inventory

This section documents every resource type that requires special handling, override, or workaround in the MPP restricted environment. Updated from live testing against `dev-spoke-aws-us-east-1`.

### Cluster-side bugs (needs MPP admin — cannot be worked around)

| Resource | Issue |
|----------|-------|
| `Route` | Admission webhook `v1.route.openshift.io` panics with nil pointer dereference on all new creates cluster-wide. Existing routes work. |
| `Service type: LoadBalancer` | AWS ELB provisioning fails: `InvalidSubnet: Not enough IP space available in subnet-0e04e2925720142be. ELB requires at least 8 free IP addresses.` |

Both issues are confirmed cluster-wide, affect all namespaces, and require MPP cluster admin intervention. Workaround for manual testing only: `oc port-forward svc/frontend-service 3000:3000 -n $NAMESPACE`.

### Resources that must be created differently or filtered

| Resource | Issue | Fix |
|----------|-------|-----|
| `Namespace` | Cannot create directly — MPP requires `TenantNamespace` CR | Filter skips `Namespace` kind; `provision.sh` applies `TenantNamespace` |
| `TenantNamespace` | Must be `type: runtime` — `build` type blocks Route admission webhook | `provision.sh` uses `type: runtime` |
| `ClusterRoleBinding` | Base manifests hardcode `namespace: ambient-code` in subjects | Filter patches all subjects to PR namespace |
| `PersistentVolumeClaim` | MPP storage webhooks require appcode label, reclaimPolicy annotation, and explicit storageClass | Filter injects all three (see PVC requirements below) |
| `Route` | Hostname auto-generated from namespace name by OpenShift, resulting in hostnames >63 chars | Filter sets explicit short `spec.host` using `pr-N` prefix |

### PVC MPP admission requirements (all three required)

| Requirement | Type | Value |
|-------------|------|-------|
| `paas.redhat.com/appcode: AMBC-001` | **Label** (not annotation) | Required by storage webhook |
| `kubernetes.io/reclaimPolicy: Delete` | Annotation | Required by storage webhook |
| `storageClassName: aws-ebs` | Spec field | Required — default storageClass not accepted |

### Secrets — what must be copied from `ambient-code--runtime-int`

Verified against live PR namespace `ambient-code--pr-1005`:

| Secret | Status | Notes |
|--------|--------|-------|
| `ambient-vertex` | ✅ Copied by install.sh | Vertex AI credentials |
| `ambient-api-server` | ✅ Copied by install.sh | API server config |
| `ambient-api-server-db` | ✅ Copied by install.sh | DB connection for api-server |
| `postgresql-credentials` | ❌ Not copied — pod fails with `secret "postgresql-credentials" not found` | Exists in runtime-int; add to install.sh |
| `frontend-oauth-config` | ❌ Not copied — pod stuck with `MountVolume.SetUp failed: secret "frontend-oauth-config" not found` | Exists in runtime-int; add to install.sh |
| `minio-credentials` | ❌ Not in runtime-int — pod fails with `secret "minio-credentials" not found` | Must be generated or created from known values |

### Images — CI not pushing PR-tagged images

`manifest unknown` errors for all Ambient component images:
```
Failed to pull image "quay.io/ambient_code/vteam_operator:pr-1005-amd64": manifest unknown
```

Root cause: `components-build-deploy.yml` PR build step has `push: false`. Images are built but not pushed to quay. **Fix: change `push: false` → `push: true` in the PR build step.**

### Open items / pending fixes

| Item | Priority | Owner |
|------|----------|-------|
| Route webhook panic | Blocker for E2E | MPP cluster admin |
| LoadBalancer subnet exhaustion | Blocker for E2E | MPP cluster admin |
| Add `postgresql-credentials` and `frontend-oauth-config` to `install.sh` copy list | High | Platform team |
| Determine source of `minio-credentials` and add to install.sh | High | Platform team |
| Change `push: false` → `push: true` in `components-build-deploy.yml` | High | Platform team |

---

## Kustomize Filter Pipeline

`install.sh` runs:
```
kustomize build overlays/production | python3 filter.py | oc apply --token=$ARGOCD_TOKEN -n $NAMESPACE -f -
```

The Python filter transforms the kustomize output before applying:

| Kind | Transform |
|------|-----------|
| `Namespace` | Skipped — namespace managed by TenantNamespace CR |
| `ClusterRoleBinding` | Subject namespace patched from `ambient-code` → PR namespace |
| `PersistentVolumeClaim` | Adds `kubernetes.io/reclaimPolicy: Delete` annotation, `paas.redhat.com/appcode: AMBC-001` label, `storageClassName: aws-ebs` |
| `Route` | Sets explicit `spec.host` with short PR-id-based hostname |

### PVC MPP Admission Requirements
MPP storage webhooks require all PVCs to have:
- **Annotation:** `kubernetes.io/reclaimPolicy: Delete`
- **Label:** `paas.redhat.com/appcode: AMBC-001` (label, not annotation)
- **StorageClass:** `storageClassName: aws-ebs`

### ClusterRoleBinding Subject Patching
The base kustomize manifests hardcode `namespace: ambient-code` in ClusterRoleBinding subjects. The filter patches all subjects to the PR namespace:
```python
CRB_NS_RE = re.compile(r'(  namespace:\s*)ambient-code(\s*$)', re.MULTILINE)
doc = CRB_NS_RE.sub(r'\g<1>' + namespace + r'\g<2>', doc)
```

---

## Image Tagging Convention

PR builds in `components-build-deploy.yml` push images tagged:

```
quay.io/ambient_code/vteam_<component>:pr-<PR_NUMBER>-<arch>
```

e.g. `quay.io/ambient_code/vteam_backend:pr-42-amd64`

No SHA in the tag — `pr-<N>-<arch>` is overwritten on each new commit to the PR. The cluster always pulls the latest build for that PR. The test cluster is single-arch; no multi-arch manifest needed.

**Required change to `components-build-deploy.yml`:** In the PR build step (currently line 209), change `push: false` → `push: true`.

---

## What the Provisioner Does NOT Do

- It does not install Ambient into the namespace — that is the responsibility of the **Ambient installer** (separate spec)
- It does not create ArgoCD Applications
- It does not manage secrets or egress rules
- It does not interact with GitHub or GitLab

The provisioner has one job: **namespace exists** or **namespace does not exist**.

---

## Integration Point

The provisioner is called by the Ambient e2e test harness:

```
e2e harness
  ├── calls provisioner.create(instance-id)  → namespace ready
  ├── calls ambient-installer(namespace, image-tag, host)  → Ambient running
  ├── runs test suite against instance URL
  └── calls provisioner.destroy(instance-id)  → namespace gone
```

---

## File Layout

```
components/pr-test/
├── README.md               ← this document (spec)
├── build.sh                ← build and push all images for a PR
├── provision.sh            ← create/destroy TenantNamespace CR
└── install.sh              ← install Ambient into a provisioned namespace
```

```
.github/workflows/
├── pr-e2e-openshift.yml       ← build → provision → install → e2e → teardown
└── pr-namespace-cleanup.yml   ← PR closed → destroy (safety net)
```

```
.claude/skills/
├── ambient/SKILL.md           ← how to install Ambient into any OpenShift namespace
└── ambient-pr-test/SKILL.md   ← how to run the full PR test workflow (references this file)
```
