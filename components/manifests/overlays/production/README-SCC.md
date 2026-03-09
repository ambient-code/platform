# OpenShift Security Context Constraints (SCC)

The **postgresql**, **minio**, and **ambient-api-server-db** deployments run as fixed UIDs (999 or 1000) for their data volumes. OpenShift's default **restricted-v2** SCC only allows UIDs in the namespace range, so these pods must use the **nonroot** SCC.

## One-time grant (cluster-admin)

After deploying, grant the nonroot SCC to all three service accounts:

```bash
oc adm policy add-scc-to-user nonroot -z postgresql -n ambient-code
oc adm policy add-scc-to-user nonroot -z minio -n ambient-code
oc adm policy add-scc-to-user nonroot -z ambient-api-server-db -n ambient-code
```

Then restart the deployments so pods are recreated with the new SCC:

```bash
oc rollout restart deployment/postgresql deployment/minio deployment/ambient-api-server-db -n ambient-code
```
