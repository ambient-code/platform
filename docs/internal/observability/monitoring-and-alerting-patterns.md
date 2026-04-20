# Monitoring and Alerting Patterns

## Overview

The Ambient Code Platform uses **cluster-native monitoring** for Kubernetes resources via Prometheus and **external workflows** for non-cluster tasks. This guide helps you choose the right approach.

## Decision Tree

### Use PrometheusRule (cluster-native) when:

- ✅ Monitoring **Kubernetes resources** (Deployments, DaemonSets, Pods, Services)
- ✅ Alerting on **cluster health** or **runtime state**
- ✅ Need **immediate alerting** (sub-minute latency)
- ✅ Metrics already exposed via Prometheus/ServiceMonitor
- ✅ Alert should trigger **oncall pages** or integrate with existing alertmanager

**Examples:**
- DaemonSet pod unavailable on nodes
- Deployment has no available replicas
- High error rate in service metrics
- Pod crash loops or restarts

### Use GitHub Actions workflows when:

- ✅ Checking **external systems** (GitHub API, third-party services)
- ✅ Scheduled **validation tasks** (dependency updates, drift detection)
- ✅ Multi-step **automation** (discover → validate → PR)
- ✅ No immediate alerting needed (hourly/daily cadence is fine)
- ✅ Requires GitHub API access or repository operations

**Examples:**
- Dependabot alert scanning
- Model registry sync checks
- SDK drift detection
- Documentation freshness checks

## Cluster-Native Monitoring (Prometheus)

### PrometheusRule Pattern

The platform includes Prometheus monitoring via OpenShift User Workload Monitoring. Alerts are defined as PrometheusRule resources in component manifests.

**Example:** `components/manifests/components/image-puller/prometheusrule.yaml`

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: acp-image-puller
  namespace: acp-image-puller
  labels:
    app.kubernetes.io/part-of: acp-image-puller
spec:
  groups:
    - name: acp-image-puller
      rules:
        - alert: ImagePullerDaemonSetUnavailable
          expr: |
            kube_daemonset_status_number_unavailable{namespace="acp-image-puller", daemonset="acp-image-puller-ds"} > 0
          for: 15m
          labels:
            severity: warning
          annotations:
            summary: "Image puller DaemonSet has unavailable pods"
            description: >-
              {{ $value }} node(s) do not have a running image puller pod.
```

### Key Characteristics

- **Co-located with component**: PrometheusRule lives in `components/manifests/components/<name>/`
- **Namespace scoped**: Deployed to the same namespace as the component
- **Severity labels**: `critical` (pages oncall), `warning` (tickets/notifications)
- **For duration**: Avoid flapping alerts with `for: 10m` or similar
- **Actionable descriptions**: Include impact and suggested remediation

### Available Metrics

**Platform metrics** (from operator):
- `ambient_session_startup_duration`
- `ambient_session_phase_transitions`
- `ambient_sessions_total`
- `ambient_sessions_completed`
- `ambient_reconcile_duration`

**Kubernetes metrics** (from kube-state-metrics):
- `kube_deployment_status_available_replicas`
- `kube_daemonset_status_number_unavailable`
- `kube_pod_status_phase`
- `kube_pod_container_status_restarts_total`

See [Operator Metrics Guide](operator-metrics-visualization.md) for full metric reference.

## External Workflows (GitHub Actions)

### Scheduled Workflow Pattern

For tasks that check external systems or perform periodic validation without immediate alerting requirements.

**Example:** `.github/workflows/model-discovery.yml`

```yaml
name: Model Discovery Sync
on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours
  workflow_dispatch:

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - name: Discover models
        run: ./scripts/discover-models.sh
      - name: Create PR if drift detected
        run: gh pr create --title "sync: update model registry"
```

### Key Characteristics

- **Scheduled cadence**: Hourly, daily, or weekly checks
- **GitHub API integration**: Uses `gh` CLI or GitHub API
- **Self-service validation**: Creates PRs for human review, doesn't auto-merge critical changes
- **Workflow dispatch**: Manual triggers for testing

## When NOT to Use Either

- **Application-level observability**: Use [Langfuse](observability-langfuse.md) for LLM tracing and cost tracking
- **Log aggregation**: Use cluster logging (Loki, CloudWatch) for centralized logs
- **APM tracing**: Use OpenTelemetry for distributed tracing

## References

- [Prometheus Operator Docs](https://prometheus-operator.dev/docs/)
- [OpenShift User Workload Monitoring](https://docs.openshift.com/container-platform/latest/monitoring/enabling-monitoring-for-user-defined-projects.html)
- [GitHub Actions Scheduled Events](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#schedule)
- [Operator Metrics Guide](operator-metrics-visualization.md)
- [Langfuse Observability](observability-langfuse.md)
