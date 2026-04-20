# Cluster Reader Service Account

Read-only service account using OpenShift's built-in `cluster-reader` ClusterRole.

## Use cases

- CI pipelines that observe cluster state (pod status, deployment health, events)
- Development tooling and dashboards
- Any automation that needs cluster-wide visibility without write access

## Permissions

- **Can read**: pods, deployments, nodes, namespaces, configmaps, events, and most other resources across all namespaces
- **Cannot read**: secrets
- **Cannot write**: anything (create, update, delete, patch all denied)

## Usage

```bash
# Apply (OpenShift only — cluster-reader does not exist on vanilla K8s/kind)
oc apply -k components/manifests/overlays/cluster-reader/

# Override namespace
cd components/manifests/overlays/cluster-reader
NS=my-namespace
kustomize edit set namespace "${NS}"
oc apply -k .

# Get a token (max 1 year)
oc create token readonly-admin -n "${NS}" --duration=8760h

# Verify read-only access
oc auth can-i get pods --all-namespaces --as=system:serviceaccount:${NS}:readonly-admin
oc auth can-i delete pods -n "${NS}" --as=system:serviceaccount:${NS}:readonly-admin
```
