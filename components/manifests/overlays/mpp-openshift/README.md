# MPP OpenShift Overlay

Kustomize overlay for the Managed Platform Plus (MPP) OpenShift environment: `ambient-code--runtime-int`.

## Apply

```bash
kubectl apply -k components/manifests/overlays/mpp-openshift/
```

## What This Overlay Does

- Targets namespace `ambient-code--runtime-int`
- Sets `PLATFORM_MODE=mpp` so the CP uses `MPPNamespaceProvisioner` (namespaces as `ambient-code--<project>`)
- Configures OIDC client credentials auth (no static K8s SA token)
- Adds `--grpc-jwk-cert-url` so the api-server validates RH SSO tokens on gRPC
- Mounts `tenantaccess-ambient-control-plane-token` for the CP's project kube client
- Mounts `ambient-runner-api-token` for runner pods to authenticate as service callers on gRPC
- Adds `allow-ambient-tenant-ingress` NetworkPolicy (ports 8000/9000 from all `ambient-code` tenant namespaces)

## ⚠️ One-Time Manual Bootstrap

Two secrets must be created manually once per cluster. They are **not** managed by kustomize (to avoid committing secret values) and are **not** required per session — only per cluster.

### Step A — TenantServiceAccount

Grants the CP's service account `namespace-admin` in every current and future tenant namespace via the tenant-access-operator.

```bash
# Apply the TenantServiceAccount CR to ambient-code--config (NOT via kustomize)
kubectl apply -f components/manifests/overlays/mpp-openshift/ambient-cp-tenant-sa.yaml
```

Wait ~30s for the operator to create `tenantaccess-ambient-control-plane-token` in `ambient-code--config`, then copy it to the runtime namespace:

```bash
kubectl get secret tenantaccess-ambient-control-plane-token \
  -n ambient-code--config \
  -o json \
  | python3 -c "
import json, sys
s = json.load(sys.stdin)
del s['metadata']['namespace']
del s['metadata']['resourceVersion']
del s['metadata']['uid']
del s['metadata']['creationTimestamp']
s['metadata'].pop('ownerReferences', None)
s['metadata'].pop('annotations', None)
s['type'] = 'Opaque'
print(json.dumps(s))
" | kubectl apply -n ambient-code--runtime-int -f -
```

**Effect:** The operator automatically injects a `namespace-admin` RoleBinding into every `ambient-code--*` namespace, including ones created after this step. The CP mounts this token as its `projectKube` client for all namespace-scoped operations.

### Step B — Static Runner API Token

The runner uses a static token to authenticate as a gRPC service caller, bypassing the per-user session ownership check on `WatchSessionMessages`.

```bash
# Generate a random token — record this value; you will need it for Step C
STATIC_TOKEN=$(python3 -c "import secrets; print(secrets.token_urlsafe(32))")

kubectl create secret generic ambient-runner-api-token \
  --from-literal=token=${STATIC_TOKEN} \
  -n ambient-code--runtime-int
```

**Do not commit the token value.**

### Step C — Set AMBIENT_API_TOKEN on the api-server

The api-server must know the static token so it can recognise the runner as a service caller:

```bash
# Patch the api-server args to include the token file
# (or set AMBIENT_API_TOKEN directly if your deployment supports it)
# The token value must match what was set in Step B
```

> **Note:** Step C is currently pending implementation — see the open gap `WatchSessionMessages PERMISSION_DENIED` in `workflows/control-plane/control-plane.workflow.md`.

## Files in This Overlay

| File | Purpose |
|------|---------|
| `kustomization.yaml` | Root kustomize config; sets namespace, images, patches |
| `ambient-control-plane.yaml` | CP Deployment — OIDC env, `PROJECT_KUBE_TOKEN_FILE`, project-kube volume mount |
| `ambient-api-server.yaml` | api-server Deployment base |
| `ambient-api-server-args-patch.yaml` | api-server command args — db, grpc, OIDC JWKS URL |
| `ambient-api-server-service-ca-patch.yaml` | Service CA annotation for TLS |
| `ambient-api-server-db.yaml` | PostgreSQL Deployment + Service |
| `ambient-api-server-route.yaml` | OpenShift Route for external access |
| `ambient-control-plane-sa.yaml` | ServiceAccount for the CP |
| `ambient-control-plane-rbac.yaml` | RBAC for the CP SA |
| `ambient-tenant-ingress-netpol.yaml` | NetworkPolicy allowing runner→api-server traffic |
| `ambient-cp-tenant-sa.yaml` | TenantServiceAccount CR (applied manually — see Step A) |

## Re-Bootstrap Required?

Only if `ambient-code--runtime-int` is destroyed, which MPP should never do to runtime/config namespaces. Session namespaces (`ambient-code--<project>`) are created and destroyed per session with no manual action required.
