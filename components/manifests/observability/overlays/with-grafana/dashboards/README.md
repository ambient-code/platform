# Grafana Dashboards

Each `.json` file in this directory is a Grafana dashboard provisioned automatically on deploy.

## Adding a new dashboard

1. Export or download a dashboard JSON file into this directory
2. Fix datasource references — replace any variable like `${DS_PROMETHEUS}`, `$datasource`, or `{"uid": "${datasource}"}` with:
   ```json
   {"type": "prometheus", "uid": "prometheus"}
   ```
3. Remove the `datasource` template variable from the `templating.list` array if present
4. Add a `configMapGenerator` entry in `kustomization.yaml`:
   ```yaml
   - name: grafana-dashboard-<name>
     files:
       - dashboards/<filename>.json
   ```
5. Add the volume and volumeMount in `grafana-deployment-patch.yaml`:
   ```yaml
   # In volumes:
   - name: dashboard-<name>
     configMap:
       name: grafana-dashboard-<name>

   # In collect-dashboards initContainer volumeMounts:
   - name: dashboard-<name>
     mountPath: /dashboards-src/<name>
   ```

## Current dashboards

| File | Description |
|------|-------------|
| `ambient-operator-dashboard.json` | Ambient Code operator session metrics |
| `k8s-cluster-monitoring.json` | Cluster-level CPU, memory, network (based on Grafana #9135) |
| `k8s-nodes.json` | Node-level resource usage |
| `k8s-namespace.json` | Namespace-level resource usage |
| `k8s-pods.json` | Pod-level resource usage |
