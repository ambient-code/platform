# OpenShift Cluster Development

This guide covers deploying Ambient Code on an OpenShift cluster using the **OpenShift internal image registry**. This is useful when iterating on local builds against a dev cluster without pushing to quay.io.

> **Standard deployment (quay.io images):** See the [Ambient installer skill](../../../../.claude/skills/ambient/SKILL.md) — it covers secrets, kustomize deploy, rollout verification, and troubleshooting for any OpenShift namespace.

> **PR test instances:** See the [ambient-pr-test skill](../../../../.claude/skills/ambient-pr-test/SKILL.md).

---

## When to Use This Guide

Use the internal registry approach when:
- You are iterating on local builds and do not want to push to quay.io on every change
- You are on a dev cluster with direct podman/docker access
- You need to test image changes that are not yet ready for a PR

For all other cases (PRs, production, ephemeral test instances), images are in quay.io and you should use the ambient skill directly.

---

## Prerequisites

- `oc` CLI installed and logged in
- `podman` or `docker` installed locally
- Access to an OpenShift cluster (CRC, ROSA, OCP on cloud)

---

## Enable the OpenShift Internal Registry

```bash
oc patch configs.imageregistry.operator.openshift.io/cluster \
  --type merge --patch '{"spec":{"defaultRoute":true}}'

REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry \
  --template='{{ .spec.host }}')

oc whoami -t | podman login --tls-verify=false -u kubeadmin \
  --password-stdin "$REGISTRY_HOST"
```

---

## Build and Push to Internal Registry

```bash
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry \
  --template='{{ .spec.host }}')
INTERNAL_REG="image-registry.openshift-image-registry.svc:5000/ambient-code"

for img in vteam_frontend vteam_backend vteam_operator vteam_public_api vteam_claude_runner vteam_state_sync vteam_api_server vteam_mcp vteam_control_plane; do
  podman tag localhost/${img}:latest ${REGISTRY_HOST}/ambient-code/${img}:latest
  podman push ${REGISTRY_HOST}/ambient-code/${img}:latest
done

oc rollout restart deployment backend-api frontend agentic-operator public-api ambient-api-server ambient-control-plane -n ambient-code
```

---

## Deploy with Internal Registry Images

**⚠️ CRITICAL**: Never commit `kustomization.yaml` with internal registry refs.

```bash
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry \
  --template='{{ .spec.host }}')

cd components/manifests/overlays/production
sed -i "s#newName: quay.io/ambient_code/#newName: ${REGISTRY_HOST}/ambient-code/#g" kustomization.yaml

cd ../..
./deploy.sh

cd overlays/production
git checkout kustomization.yaml
```

---

## JWT Configuration for Dev Clusters

The production overlay configures JWT against Red Hat SSO (`sso.redhat.com`). On a personal dev cluster without SSO, disable JWT:

```bash
oc set env deployment/ambient-api-server -n ambient-code \
  --containers=api-server \
  ENABLE_JWT=false
oc rollout restart deployment/ambient-api-server -n ambient-code
```

Or patch the `ambient-api-server-jwt-args-patch.yaml` to set `--enable-jwt=false` before deploying.

---

## Cross-Namespace Image Pull

Runner pods are created in dynamic project namespaces and must pull from the `ambient-code` namespace in the internal registry:

```bash
oc policy add-role-to-group system:image-puller system:serviceaccounts \
  --namespace=ambient-code
```

Without this, runner pods fail with `ErrImagePull` / `authentication required`.

---

## Next Steps

Once deployed, follow the verification and access steps in the [ambient skill](../../../../.claude/skills/ambient/SKILL.md#step-6-verify-installation).
