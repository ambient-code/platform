---
name: ambient-pr-test
description: >-
  Deploy a PR's images (api-server, control-plane, runner) to any OpenShift
  namespace for integration testing.  Works on both standard OpenShift clusters
  and MPP managed clusters.  Auto-detects the environment and chooses the
  right script.
---

# Ambient PR Test Skill

Deploy PR-tagged Ambient images into an OpenShift namespace for integration testing.

**Invoke with a PR URL:**
```
with skills/control-plane/ambient-pr-test  https://github.com/ambient-code/platform/pull/1599
```

Optional modifiers:
- **`--keep-alive`** — skip teardown
- **`deploy-only`** / **`teardown-only`** — single phase
- **`--namespace <ns>`** — target namespace (defaults to current project)

---

## Environment Detection

```bash
if oc api-resources --api-group=tenant.paas.redhat.com 2>/dev/null  < /dev/null |  grep -q TenantNamespace; then
  echo "MPP"    # use provision.sh + install.sh
else
  echo "Standard"  # use install-standard.sh
fi
```

| | Standard OpenShift | MPP |
|--|-------------------|-----|
| Namespace | Pre-existing | Created via TenantNamespace CR |
| Script | `install-standard.sh` | `provision.sh` + `install.sh` |
| Auth | Development (no JWT) | Production (RH SSO JWT) |
| Secrets | Auto-generated | Copied from runtime-int |

---

## Standard OpenShift

For any cluster where you have `oc` access and an existing namespace.

### Prerequisites

- `oc` logged in with create permissions for deployments, services, routes, secrets, ClusterRoles
- Existing namespace (e.g. `mturansk`)
- PR images at `quay.io/ambient_code/vteam_*:pr-<NUMBER>`

### Deploy

```bash
PR_NUMBER=1599
NAMESPACE=$(oc project -q)
bash components/pr-test/install-standard.sh "$NAMESPACE" "pr-${PR_NUMBER}"
```

The script:
1. Creates DB and app secrets (auto-generated)
2. Creates CP ServiceAccount + ClusterRole + ClusterRoleBinding
3. Deploys PostgreSQL, api-server (dev mode), control-plane (standard mode)
4. Creates Route (auto-assigned `.apps.*` host)
5. Waits for rollouts, smoke-checks health

### Verify

```bash
API_HOST=$(oc get route ambient-api-server -n "$NAMESPACE" -o jsonpath='{.spec.host}')
curl -sk "https://${API_HOST}/api/ambient"
acpctl login --url "https://${API_HOST}"
```

### Teardown

```bash
NAMESPACE=$(oc project -q)
oc delete deployment,svc,route,configmap,secret -l app=ambient-api-server -n "$NAMESPACE"
oc delete deployment,svc -l app=ambient-control-plane -n "$NAMESPACE"
oc delete secret ambient-control-plane-token ambient-cp-token-keypair -n "$NAMESPACE" --ignore-not-found
oc delete clusterrole,clusterrolebinding "ambient-control-plane-${NAMESPACE}" --ignore-not-found
```

---

## MPP Workflow

For MPP clusters (`dev-spoke-aws-us-east-1`).  See `components/pr-test/MPP-ENVIRONMENT.md`.

```bash
PR_NUMBER=1005; ID="pr-${PR_NUMBER}"
bash components/pr-test/build.sh "https://github.com/ambient-code/platform/pull/${PR_NUMBER}"
bash components/pr-test/provision.sh create "$ID"
bash components/pr-test/install.sh "ambient-code--${ID}" "pr-${PR_NUMBER}"
# teardown:
bash components/pr-test/provision.sh destroy "$ID"
```

---

## Troubleshooting

### Images not found

Check quay.io for the tag: `https://quay.io/repository/ambient_code/vteam_control_plane?tab=tags`

### CP can't reach api-server

```bash
oc get deployment ambient-control-plane -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.containers[0].env}' | python3 -m json.tool | grep AMBIENT
```

### JWT errors in standard mode

Verify `AMBIENT_ENV=development`:
```bash
oc get deployment ambient-api-server -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="api-server")].env[?(@.name=="AMBIENT_ENV")].value}'
```

### Route not resolving

```bash
oc get route ambient-api-server -n "$NAMESPACE" -o jsonpath='{.spec.host}'
```

### CP keypair secret

Auto-generated on first start as `ambient-cp-token-keypair`. Check CP logs:
```bash
oc logs deployment/ambient-control-plane -n "$NAMESPACE" | grep keypair
```

### Build fails

```bash
docker login quay.io  # or: podman login quay.io
```
