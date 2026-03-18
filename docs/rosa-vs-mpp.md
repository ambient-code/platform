We've been deploying Ambient Code Platform to ROSA, and now we need to deploy it to IT's MP+ OSD environment.

In ROSA we've had full cluster-admin to deploy whatever we want, whereas IT's environment has different security restrictions.

# ROSA vs IT's MP+ (OSD): environment comparison

| Concern | ROSA (cluster-admin) | IT's MP+ (OSD) |
|---|---|---|
| **Namespace** | Created by kustomize (`kind: Namespace`) | IT creates via `TenantNamespace`; kustomize must not emit `Namespace` resources |
| **CRDs** | Created by kustomize (`AgenticSession`, `ProjectSettings`) | IT does not allow tenant-managed CRDs; must be installed separately by IT |
| **Secrets** | Created by kustomize (DB credentials, API keys) | Engineers create manually in-cluster (Vault later per RHOAIENG-47031); kustomize must not emit `Secret` resources |
| **RBAC** | kustomize creates 9 `ClusterRoles` + 4 `ClusterRoleBindings` | Runner SA cannot create cluster-scoped RBAC; IT manages `ClusterRoles`/`ClusterRoleBindings` outside kustomize |
| **ServiceAccounts** | 5 created by kustomize (in `rbac/`) | Only 2 created by kustomize (from `core/`); remaining 3 are bundled with RBAC and managed by IT |
| **PVC storage class** | Cluster default (gp3-csi) | Must specify `storageClassName: aws-ebs` |
| **PVC labels** | None required | Must have `paas.redhat.com/appcode: AMBC-001` |
| **PVC annotations** | None required | Must have `kubernetes.io/reclaimPolicy: Delete` |
| **Egress/firewall** | No restrictions (full internet) | AWS-level firewall rules (no `TenantEgress`); rules differ between us-east-1 and us-west-2 |
| **GitHub Actions runner** | GitHub-hosted (`ubuntu-latest`) | Self-hosted runner in `ambient-code--github-runner` namespace on the OSD cluster |
| **Deploy permissions** | Full cluster-admin token | Runner SA has namespace-scoped permissions in `ambient-code--runtime-int` only |
| **Container images** | Pull from any registry | Must pass through cluster-level AWS firewall allowlist |
