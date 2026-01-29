# Observability & Monitoring

Documentation for monitoring and observability features in the Ambient Code Platform.

## 📊 Available Observability Tools

### Langfuse - LLM Observability
**[Langfuse Guide](observability-langfuse.md)**

Track Claude API usage, costs, and performance:
- Turn-level generations with token and cost tracking
- Tool execution visibility
- Session grouping and multi-user cost allocation
- Real-time trace streaming
- Privacy-first with message masking enabled by default

**Deployment:** [Langfuse Deployment](../deployment/langfuse.md)

---

### Operator Metrics - Platform Monitoring
**[Operator Metrics Guide](operator-metrics-visualization.md)**

Visualize operator metrics using OpenShift User Workload Monitoring:
- Session startup duration
- Phase transitions and reconciliation performance
- Pod creation speed
- Error rates by namespace

**Metrics Available:**
- `ambient_session_startup_duration`
- `ambient_session_phase_transitions`
- `ambient_sessions_total`
- `ambient_sessions_completed`
- `ambient_reconcile_duration`

---

## Quick Start

### Deploy Langfuse

```bash
# Auto-detect platform
./e2e/scripts/deploy-langfuse.sh

# Or specify
./e2e/scripts/deploy-langfuse.sh --openshift
```

### Deploy Operator Metrics

```bash
make deploy-observability
```

### View Metrics

**OpenShift Console:**
- Navigate to: Observe → Metrics
- Query: `ambient_sessions_total`

**Grafana (optional):**
```bash
make add-grafana
```

## Privacy & Security

### Langfuse Message Masking

**Default:** User messages and Claude responses are **redacted** in traces

**What Gets Logged:**
- ✅ Token counts and costs
- ✅ Model names and metadata
- ✅ Tool names and execution status
- ❌ User prompts → `[REDACTED FOR PRIVACY]`
- ❌ Assistant responses → `[REDACTED FOR PRIVACY]`

See [Langfuse Guide](observability-langfuse.md) for configuration details.

## Cost Tracking

### Model Pricing

All Claude models have accurate pricing configured:
- [Model Pricing Reference](../reference/model-pricing.md)
- Prompt caching cost optimization (25% premium, 90% discount)
- Per-session cost tracking in Langfuse

### Cost Allocation

Track costs by:
- **User:** `user_id` in traces
- **Project:** `namespace` metadata
- **Session:** `session_id` grouping
- **Model:** Model name in metadata

## Troubleshooting

### Langfuse Not Receiving Traces

```bash
# Check runner has Langfuse config
kubectl get secret ambient-admin-langfuse-secret -n ambient-code

# Check runner logs
kubectl logs <session-pod> -n <namespace> | grep -i langfuse
```

### Operator Metrics Not Showing

```bash
# Check User Workload Monitoring enabled
oc get pods -n openshift-user-workload-monitoring

# Check ServiceMonitor exists
oc get servicemonitor ambient-otel-collector -n ambient-code

# Test OTel Collector
oc port-forward svc/otel-collector 8889:8889 -n ambient-code
curl http://localhost:8889/metrics | grep ambient
```

## Related Documentation

- [Deployment Guide](../deployment/) - Deploying observability components
- [Architecture](../architecture/) - System design
- [Model Pricing](../reference/model-pricing.md) - Claude pricing details

## References

- **Langfuse**: https://langfuse.com/docs
- **OpenTelemetry**: https://opentelemetry.io/docs/
- **Prometheus**: https://prometheus.io/docs/
- **Grafana**: https://grafana.com/docs/
