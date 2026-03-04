# Langfuse Deployment

This guide covers deploying Langfuse for LLM observability in the Ambient Code Platform.

## Quick Start

Deploy Langfuse to your cluster:

```bash
# Auto-detect platform (OpenShift or Kubernetes)
./e2e/scripts/deploy-langfuse.sh

# Or explicitly specify platform
./e2e/scripts/deploy-langfuse.sh --openshift
./e2e/scripts/deploy-langfuse.sh --kubernetes
```

## What Gets Deployed

- **Langfuse Web** (UI and API)
- **Langfuse Worker** (Background processing)
- **PostgreSQL** (Metadata storage)
- **ClickHouse** (Analytics database with minimal logging configuration)
- **Redis** (Caching)
- **MinIO** (S3-compatible object storage)
- **Zookeeper** (ClickHouse coordination)

## Privacy & Security

### Message Masking (Enabled by Default)

By default, the Claude Code runner **redacts user messages and assistant responses** in Langfuse traces to protect privacy. Only usage metrics (tokens, costs) and metadata are logged.

**Important**: Masking is **enabled by default** without requiring any environment variable. The runner code defaults to `LANGFUSE_MASK_MESSAGES=true` if the variable is not set.

**To verify masking is enabled** (check runner logs):
```
Langfuse: Privacy masking ENABLED - user messages and responses will be redacted
```

**To disable masking** (dev/testing only):
```bash
# Add to runner environment (NOT recommended for production!)
# Only needed if you want to DISABLE the default masking behavior
LANGFUSE_MASK_MESSAGES=false
```

### What Gets Logged

**With Masking Enabled** (default):
- ✅ Token counts (input, output, cache)
- ✅ Cost calculations
- ✅ Model names and versions
- ✅ Session/turn metadata
- ✅ Tool names and execution status
- ❌ User prompts → `[REDACTED FOR PRIVACY]`
- ❌ Assistant responses → `[REDACTED FOR PRIVACY]`
- ❌ Long tool outputs → `[REDACTED FOR PRIVACY]`

**With Masking Disabled** (dev/testing only):
- ⚠️ Everything above PLUS full message content
- ⚠️ May expose sensitive user data!

### Implementation

Masking is implemented in the Claude Code runner:
- **File**: `components/runners/claude-code-runner/observability.py`
- **Function**: `_privacy_masking_function()`
- **Tests**: `components/runners/claude-code-runner/tests/test_privacy_masking.py`

## Post-Deployment Configuration

After deployment completes:

1. **Access Langfuse UI**:
   - OpenShift: `https://langfuse-<namespace>.apps.<cluster-domain>`
   - Kubernetes: `https://langfuse.local` (configure DNS or `/etc/hosts`)

2. **Create Account**:
   - Sign up in the Langfuse UI
   - Create a project

3. **Generate API Keys**:
   - Navigate to: Settings → API Keys
   - Click "Create new API keys"
   - Save both public and secret keys

4. **Configure Runner**:
   ```bash
   # Create secret in runner namespace (e.g., ambient-code)
   kubectl create secret generic ambient-admin-langfuse-secret \
     --from-literal=LANGFUSE_ENABLED=true \
     --from-literal=LANGFUSE_PUBLIC_KEY=pk-lf-xxx \
     --from-literal=LANGFUSE_SECRET_KEY=sk-lf-xxx \
     --from-literal=LANGFUSE_HOST=http://langfuse-web.langfuse.svc.cluster.local:3000 \
     --namespace=ambient-code
   ```

5. **Optional: Disable Masking** (dev/testing only):
   ```bash
   kubectl patch secret ambient-admin-langfuse-secret \
     --type=merge \
     -p '{"stringData":{"LANGFUSE_MASK_MESSAGES":"false"}}' \
     --namespace=ambient-code
   ```

## ClickHouse Configuration

The deployment includes optimized ClickHouse configuration to minimize disk usage:

### Minimal System Logging

The `langfuse-values-clickhouse-minimal-logging.yaml` file disables most ClickHouse internal system tables:
- ❌ `system.query_log` - Disabled (saves ~5GB+)
- ❌ `system.query_thread_log` - Disabled
- ❌ `system.part_log` - Disabled
- ❌ Other internal logs - Disabled
- ✅ Langfuse trace data - Fully enabled

### TTL Configuration

The `configure-clickhouse-ttl.sh` script sets retention policies on remaining system logs:
- Automatically runs after deployment
- Default: 7-day retention for system logs
- Prevents disk space issues on long-running deployments

**Manual TTL configuration**:
```bash
cd e2e/scripts
./configure-clickhouse-ttl.sh \
  --namespace langfuse \
  --password <clickhouse-password> \
  --retention-days 7
```

## Troubleshooting

### Check Deployment Status

```bash
kubectl get pods -n langfuse
kubectl get svc -n langfuse

# OpenShift
oc get route -n langfuse

# Kubernetes
kubectl get ingress -n langfuse
```

### View Logs

```bash
# All Langfuse components
kubectl logs -n langfuse -l app.kubernetes.io/name=langfuse --tail=50 -f

# Specific component
kubectl logs -n langfuse deployment/langfuse-web -f
kubectl logs -n langfuse deployment/langfuse-worker -f
```

### Common Issues

**S3 Credentials Missing**:
- The deployment script automatically patches S3 credentials after Helm install
- If you see S3 errors, re-run the patching section of the script

**ClickHouse Disk Space**:
- Check if TTL configuration succeeded: `kubectl logs -n langfuse <langfuse-web-pod> | grep TTL`
- Manually run: `./e2e/scripts/configure-clickhouse-ttl.sh`
- Verify minimal logging is enabled in values file

**Masking Not Working**:
- Check runner logs for: `Langfuse: Privacy masking ENABLED`
- Verify `LANGFUSE_MASK_MESSAGES` is not set to `false`
- Run tests: `cd components/runners/claude-code-runner && python tests/test_privacy_masking.py`

## Cleanup

**WARNING**: This deletes all Langfuse data!

```bash
kubectl delete namespace langfuse
```

## Architecture

```
┌─────────────────────────────────────────────┐
│ Claude Code Runner (Job Pod)               │
│ ┌─────────────────────────────────────────┐ │
│ │ observability.py                        │ │
│ │ ├─ Privacy Masking Function             │ │
│ │ ├─ Langfuse SDK Client                  │ │
│ │ └─ Trace Creation (turn/tool spans)     │ │
│ └─────────────────────────────────────────┘ │
└──────────────────┬──────────────────────────┘
                   │ Masked traces (HTTP/JSON)
                   ▼
┌─────────────────────────────────────────────┐
│ Langfuse Service                            │
│ ┌─────────────────────────────────────────┐ │
│ │ langfuse-web (API + UI)                 │ │
│ │ Port: 3000                              │ │
│ └─────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────┐ │
│ │ langfuse-worker (Background jobs)       │ │
│ └─────────────────────────────────────────┘ │
└──────────┬──────────────────┬───────────────┘
           │                  │
           ▼                  ▼
┌──────────────────┐ ┌────────────────────────┐
│ PostgreSQL       │ │ ClickHouse             │
│ (Metadata)       │ │ (Analytics + Traces)   │
│                  │ │ - Minimal system logs  │
│                  │ │ - 7-day TTL            │
└──────────────────┘ └────────────────────────┘
```

## References

- **Langfuse Documentation**: https://langfuse.com/docs
- **Platform Docs**: See [CLAUDE.md](../../CLAUDE.md) - "Langfuse Observability" section
- **Implementation**: `components/runners/claude-code-runner/observability.py`
- **Tests**: `components/runners/claude-code-runner/tests/test_privacy_masking.py`
- **Deployment Script**: `e2e/scripts/deploy-langfuse.sh`
