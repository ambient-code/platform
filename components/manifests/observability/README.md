# Observability Stack for Ambient Code Platform

Observability for OpenShift using **User Workload Monitoring** (no dedicated Prometheus needed).

## Architecture

```
Operator (OTel SDK) → OTel Collector → OpenShift Prometheus
                                              ↓
                                       OpenShift Console
                                              ↓
                                       Grafana (optional)
```

## Quick Start

### Deploy Base Stack

```bash
# From repository root
make deploy-observability

# Or manually
kubectl apply -k components/manifests/observability/
```

**What you get**: OTel Collector + ServiceMonitor (128MB)

### View Metrics

Open **OpenShift Console → Observe → Metrics** and query:
- `ambient_sessions_total`
- `ambient_session_startup_duration_bucket`
- `ambient_session_errors`

---

## Optional: Add Grafana

If you want custom dashboards:

```bash
# Add Grafana overlay
kubectl apply -k components/manifests/observability/overlays/with-grafana/
```

**Adds**: Grafana (additional 128MB) - still uses OpenShift Prometheus

**Access Grafana**:
```bash
# Create route
oc create route edge grafana --service=grafana -n ambient-code

# Get URL
oc get route grafana -n ambient-code -o jsonpath='{.spec.host}'
# Login: admin/admin
```

**Import dashboard**: Upload `dashboards/ambient-operator-dashboard.json` in Grafana UI

---

## Components

| Component | What It Does | Resource Usage |
|-----------|--------------|----------------|
| **OTel Collector** | Receives metrics from operator, exports to Prometheus format | 128MB RAM |
| **ServiceMonitor** | Tells OpenShift Prometheus to scrape OTel Collector | None |
| **Grafana** (optional) | Custom dashboards | 128MB RAM, 5GB storage |

## Metrics Available

All metrics are prefixed with `ambient_`:

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| `ambient_session_startup_duration` | Histogram | Time from creation to Running phase | p95 > 60s |
| `ambient_session_phase_transitions` | Counter | Phase transition events | - |
| `ambient_sessions_total` | Counter | Total sessions created | Sudden spikes |
| `ambient_sessions_completed` | Counter | Sessions that reached terminal states | - |
| `ambient_reconcile_duration` | Histogram | Reconciliation loop performance | p95 > 10s |
| `ambient_pod_creation_duration` | Histogram | Time to create runner pods | p95 > 30s |
| `ambient_token_provision_duration` | Histogram | Token provisioning time | p95 > 5s |
| `ambient_session_errors` | Counter | Errors during reconciliation | Rate > 0.1/s |

## Accessing Components

### OpenShift Console (Options 1 & 2)

Navigate to **Observe → Metrics** and query:

```promql
# Total sessions created
ambient_sessions_total

# Session creation rate
rate(ambient_sessions_total[5m])

# p95 startup time
histogram_quantile(0.95, rate(ambient_session_startup_duration_bucket[5m]))

# Error rate by namespace
sum by (namespace) (rate(ambient_session_errors[5m]))
```

### OTel Collector Logs

```bash
kubectl logs -n ambient-code -l app=otel-collector -f
```

## Production Setup

### Enable OpenShift User Workload Monitoring

Check if enabled:
```bash
oc -n openshift-user-workload-monitoring get pod
```

If not:
```bash
oc apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
data:
  config.yaml: |
    enableUserWorkload: true
EOF
```

## Troubleshooting

### No metrics showing in OpenShift Console

1. **Verify User Workload Monitoring is enabled**:
   ```bash
   oc -n openshift-user-workload-monitoring get pod
   # Should see prometheus-user-workload pods
   ```

2. **Check ServiceMonitor is discovered**:
   ```bash
   oc get servicemonitor ambient-otel-collector -n ambient-code
   oc describe servicemonitor ambient-otel-collector -n ambient-code
   ```

3. **Check OTel Collector is receiving metrics**:
   ```bash
   kubectl logs -n ambient-code -l app=otel-collector | grep -i "metric"
   ```

4. **Check operator is sending metrics**:
   ```bash
   kubectl logs -n ambient-code -l app=agentic-operator | grep -i "otel\|metric"
   ```

5. **Test direct query to OTel Collector**:
   ```bash
   kubectl port-forward svc/otel-collector 8889:8889 -n ambient-code
   curl http://localhost:8889/metrics | grep ambient
   ```

### Grafana shows "No data"

1. **Verify Grafana ServiceAccount has permissions**:
   ```bash
   oc auth can-i get --subresource=metrics pods \
     --as=system:serviceaccount:ambient-code:grafana -n openshift-monitoring
   # Should return "yes"
   ```

2. **Check datasource configuration** in Grafana:
   - Go to Configuration → Data Sources
   - Test the OpenShift Prometheus datasource
   - Check for authentication errors

3. **Verify you're querying the right metrics**:
   - Metrics should be prefixed with `ambient_`
   - Try simple query first: `ambient_sessions_total`

## Cleanup

```bash
kubectl delete -k components/manifests/observability/overlays/with-grafana/  # If Grafana deployed
kubectl delete -k components/manifests/observability/
```

