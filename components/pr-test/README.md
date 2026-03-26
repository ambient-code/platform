# Specification: Ephemeral Namespace Provisioning for S0.x Instances

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
- **Target cluster:** `dev-spoke-aws-us-east-1` (initially)
- **Namespace naming convention:** `ambient-code--<instance-id>`
- **Instance ID format:** derived from PR/branch identifier, e.g. `pr-123-feat-xyz`
- **Resulting namespace:** `ambient-code--pr-123-feat-xyz`

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
    tenant.paas.redhat.com/namespace-type: build   # always "build" for ephemeral instances
    tenant.paas.redhat.com/tenant: ambient-code
    ambient-code/instance-type: s0x                # for capacity counting
  name: <instance-id>                               # e.g. pr-123-feat-xyz
  namespace: ambient-code--config                   # always this namespace
spec:
  network:
    security-zone: internal
  type: build                                       # always "build" for ephemeral instances
```

### Verified Example

The following was applied and confirmed working on `dev-spoke-aws-us-east-1`:

```yaml
apiVersion: tenant.paas.redhat.com/v1alpha1
kind: TenantNamespace
metadata:
  labels:
    tenant.paas.redhat.com/namespace-type: build
    tenant.paas.redhat.com/tenant: ambient-code
  name: pr-123-example
  namespace: ambient-code--config
spec:
  network:
    security-zone: internal
  type: build
```

Resulting namespace `ambient-code--pr-123-example` was `Active` within 11 seconds with the following platform-injected labels:

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

The ServiceAccount running the provisioner needs the following on the `ambient-code--config` namespace:

```yaml
rules:
  - apiGroups: ["tenant.paas.redhat.com"]
    resources: ["tenantnamespaces"]
    verbs: ["get", "list", "create", "delete", "watch"]
```

On `dev-spoke-aws-us-east-1` this is satisfied by a `TenantServiceAccount` with role `tenant-admin`.

The existing `tenantserviceaccount-argocd.yaml` already carries `tenant-admin` — the provisioner should use a **separate** dedicated ServiceAccount, not the ArgoCD one.

---

## Instance Naming Convention

| Input | Instance ID | Resulting Namespace |
|-------|-------------|---------------------|
| PR #123, branch `feat-xyz` | `pr-123-feat-xyz` | `ambient-code--pr-123-feat-xyz` |
| PR #42, branch `fix-auth` | `pr-42-fix-auth` | `ambient-code--pr-42-fix-auth` |
| PR #7, branch `refactor-db` | `pr-7-refactor-db` | `ambient-code--pr-7-refactor-db` |

Rules:
- Lowercase only
- Hyphens as separators — no underscores, no dots
- Max length: 63 characters total for the namespace name (Kubernetes limit)
- Branch name truncated if needed, PR number always preserved

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
