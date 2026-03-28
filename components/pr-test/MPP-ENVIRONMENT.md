# MPP Restricted Environment vs Standard OpenShift

Differences observed from live testing on `dev-spoke-aws-us-east-1` and `mpp-w2-preprod`.

## Namespace Management

| | Standard OpenShift | MPP TenantNamespace |
|--|-------------------|---------------------|
| Create namespace | `oc create namespace foo` or `Namespace` CR | Apply `TenantNamespace` CR to `ambient-code--config`; operator creates it |
| Delete namespace | `oc delete namespace foo` | Delete `TenantNamespace` CR; operator finalizes deletion |
| Namespace type | N/A | Must be `type: runtime` — `build` blocks Route admission |
| Labels | You set them | Platform injects tenant labels; cannot be set directly |

## RBAC

| | Standard OpenShift | MPP TenantNamespace |
|--|-------------------|---------------------|
| `ClusterRole` creation | Token with cluster-admin | Forbidden for user tokens; requires ArgoCD SA token |
| `ClusterRoleBinding` creation | Token with cluster-admin | Forbidden for user tokens; requires ArgoCD SA token |
| CRD management | Token with cluster-admin | Forbidden for user tokens — must be pre-applied by cluster admin |
| `oc get crd` | Works | Forbidden — probe CRD presence via namespace-scoped resource access instead |
| `oc get ingresses.config.openshift.io` | Works | Forbidden — derive cluster domain from existing routes instead |

The ArgoCD service account (`tenantaccess-argocd-account-token` in `ambient-code--config`) has cluster-admin and is used for operations that require it. See `install.sh` Step 4.

## Routes

| | Standard OpenShift | MPP TenantNamespace |
|--|-------------------|---------------------|
| Create route | `oc apply` | Requires `paas.redhat.com/appcode: AMBC-001` label |
| Shard routing | Optional `shard:` label | `shard: internal` → internal domain; no shard → external domain (auto-assigned) |
| Host assignment | Auto or explicit | Auto-assigned if no `spec.host`; must match shard domain if explicitly set |

Do **not** set `shard: internal` unless you intend to use the internal domain (`apps.int.spoke.dev.us-east-1.aws.paas.redhat.com`). Without a shard label, OpenShift auto-assigns hosts on the external domain (`apps.dev-osd-east-1.mxty.p1.openshiftapps.com`).

## PersistentVolumeClaims

All three of the following are required by MPP storage admission webhooks:

| Requirement | Type | Value |
|-------------|------|-------|
| `paas.redhat.com/appcode: AMBC-001` | **Label** (not annotation) | Required by storage webhook |
| `kubernetes.io/reclaimPolicy: Delete` | Annotation | Required by storage webhook |
| `storageClassName: aws-ebs` | Spec field | Default storageClass not accepted |

## Service Exposure

| | Standard OpenShift | MPP TenantNamespace |
|--|-------------------|---------------------|
| `LoadBalancer` service | Works if cloud provider configured | Blocked — AWS subnet IP exhaustion on `dev-spoke-aws-us-east-1` |
| `NodePort` | Works | Available but nodes not directly reachable externally |
| `Route` | Works | Works — requires `paas.redhat.com/appcode` label, no `shard: internal` |

## Secrets

| | Standard OpenShift | MPP TenantNamespace |
|--|-------------------|---------------------|
| Image pull secrets | Optional | Must be present per namespace — quay.io credentials required |
| App secrets | You manage | Must be manually seeded into `SOURCE_NAMESPACE` before install |

Required secrets that must exist in `SOURCE_NAMESPACE` (`ambient-code--runtime-int`) before `install.sh` runs:

- `ambient-vertex`
- `ambient-api-server`
- `postgresql-credentials`
- `frontend-oauth-config`

## Cluster-Admin Operations

| | Standard OpenShift | MPP TenantNamespace |
|--|-------------------|---------------------|
| Cluster-admin token | Your token | `tenantaccess-argocd-account-token` SA in `ambient-code--config` |
| ArgoCD cluster linking | Standard ArgoCD | Via `TenantServiceAccount` + Secret in ArgoCD namespace |
| Credential management | Direct | `TenantCredentialManagement` CR (documented as unstable) or manual |

## MPP Tenant API — Available CRDs

(`tenant.paas.redhat.com/v1alpha1` unless noted)

| CRD | Purpose |
|-----|---------|
| `TenantNamespace` | Provision a managed namespace |
| `TenantServiceAccount` | Create a SA with cluster-linking tokens |
| `TenantEgress` | Outbound CIDR/DNS egress policy |
| `TenantNamespaceEgress` | Pod-level egress NetworkPolicy |
| `TenantGroup` | Group management |
| `TenantCredentialManagement` (`tenantaccess.paas.redhat.com/v1alpha1`) | Cluster credential linking (unstable) |
| `TenantOperatorConfig` / `TenantOperatorOptIn` | Operator configuration |

There is **no `TenantRoute`**. Routes are standard OpenShift `Route` objects.

## Reference

| Resource | URL |
|----------|-----|
| Tenant Operator | https://gitlab.cee.redhat.com/paas/tenant-operator |
| Tenant Operator Access | https://gitlab.cee.redhat.com/ddis/ai/devops/ddis-ai-gitops |
