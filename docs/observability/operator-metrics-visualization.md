# Operator Metrics Visualization Guide

Visualize Ambient Code operator metrics on OpenShift using User Workload Monitoring.

## Architecture

```
Operator (OTel) → OTel Collector → OpenShift Prometheus
                                          ↓
                                   OpenShift Console
                                          ↓
                                   Grafana (optional)
```

## Quick Start

```bash
# Deploy observability
make deploy-observability

# View metrics in OpenShift Console → Observe → Metrics

# Optional: Add Grafana
make add-grafana
```

**Full deployment guide**: See `components/manifests/observability/README.md`

---

## Available Metrics

| Metric | Description |
|--------|-------------|
| `ambient_session_startup_duration` | Time from Pending to Running |
| `ambient_session_phase_transitions` | Phase transition count |
| `ambient_sessions_total` | Total sessions created |
| `ambient_sessions_completed` | Sessions in terminal states |
| `ambient_reconcile_duration` | Reconciliation performance |
| `ambient_pod_creation_duration` | Pod provisioning speed |
| `ambient_session_errors` | Errors during reconciliation |

---

## Example Queries

In **OpenShift Console → Observe → Metrics**:

```promql
# Total sessions created
ambient_sessions_total

# Session creation rate
rate(ambient_sessions_total[5m])

# p95 session startup time
histogram_quantile(0.95, rate(ambient_session_startup_duration_bucket[5m]))

# Error rate by namespace
sum by (namespace) (rate(ambient_session_errors[5m]))

# Sessions by final phase
sum by (final_phase) (increase(ambient_sessions_completed[1h]))
```

---

## Adding Grafana (Optional)

For custom dashboards:

```bash
# Deploy Grafana
make add-grafana

# Create route
oc create route edge grafana --service=grafana -n ambient-code

# Get URL
oc get route grafana -n ambient-code -o jsonpath='{.spec.host}'
# Login: admin/admin
```

**Import dashboard**: Upload `components/manifests/observability/dashboards/ambient-operator-dashboard.json`

---

## Troubleshooting

### No metrics in OpenShift Console

1. Check User Workload Monitoring is enabled:
   ```bash
   oc -n openshift-user-workload-monitoring get pod
   ```

2. Verify ServiceMonitor exists:
   ```bash
   oc get servicemonitor ambient-otel-collector -n ambient-code
   ```

3. Check OTel Collector is receiving metrics:
   ```bash
   oc logs -l app=otel-collector -n ambient-code | grep -i metric
   ```

4. Test OTel Collector endpoint:
   ```bash
   oc port-forward svc/otel-collector 8889:8889 -n ambient-code
   curl http://localhost:8889/metrics | grep ambient
   ```

### Grafana shows "No data"

1. Check Grafana ServiceAccount has permissions:
   ```bash
   oc auth can-i get --subresource=metrics pods \
     --as=system:serviceaccount:ambient-code:grafana
   ```

2. Test datasource in Grafana UI:
   - Configuration → Data Sources → OpenShift Prometheus → Test

---

## For Non-OpenShift Deployments

If you're not on OpenShift, you need to deploy Prometheus yourself:

```bash
kubectl apply -f components/manifests/observability/prometheus.yaml
```

Update `grafana-datasource-patch.yaml` to point to `http://prometheus:9090` instead of OpenShift Prometheus.
