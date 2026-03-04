# MinIO Quickstart for Ambient Code

## Overview

MinIO provides in-cluster S3-compatible storage for Ambient Code session state, artifacts, and uploads. This guide shows you how to deploy and configure MinIO.

## Quick Setup

### 1. Deploy MinIO

```bash
# Create MinIO credentials secret
cd components/manifests/base
cp minio-credentials-secret.yaml.example minio-credentials-secret.yaml

# Edit credentials (change admin/changeme123 to secure values)
vi minio-credentials-secret.yaml

# Apply the secret
kubectl apply -f minio-credentials-secret.yaml -n ambient-code

# MinIO deployment is included in base manifests, so deploy normally
make deploy NAMESPACE=ambient-code
```

### 2. Create Bucket

```bash
# Run automated setup
make setup-minio NAMESPACE=ambient-code

# Or manually:
kubectl port-forward svc/minio 9001:9001 -n ambient-code &
open http://localhost:9001
# Login with credentials, create bucket "ambient-sessions"
```

### 3. Configure Project

Navigate to project settings in the UI and configure:

| Field | Value |
|-------|-------|
| **Enable S3 Storage** | ✅ Checked |
| **S3_ENDPOINT** | `http://minio.ambient-code.svc:9000` |
| **S3_BUCKET** | `ambient-sessions` |
| **S3_REGION** | `us-east-1` (not used by MinIO but required field) |
| **S3_ACCESS_KEY** | Your MinIO root user |
| **S3_SECRET_KEY** | Your MinIO root password |

Click **Save Integration Secrets**.

## Accessing MinIO Console

### Option 1: Port Forward

```bash
make minio-console NAMESPACE=ambient-code
# Opens at http://localhost:9001
```

### Option 2: Create Route (OpenShift)

```bash
oc create route edge minio-console \
  --service=minio \
  --port=9001 \
  -n ambient-code

# Get URL
oc get route minio-console -n ambient-code -o jsonpath='{.spec.host}'
```

## Viewing Session Artifacts

### Via MinIO Console

1. Open MinIO console: `make minio-console`
2. Navigate to "Buckets" → "ambient-sessions"
3. Browse: `{namespace}/{session-name}/`
   - `.claude/` - Session history
   - `artifacts/` - Generated files
   - `uploads/` - User uploads

### Via MinIO Client (mc)

```bash
# Install mc
brew install minio/stable/mc

# Configure alias
kubectl port-forward svc/minio 9000:9000 -n ambient-code &
mc alias set ambient http://localhost:9000 admin changeme123

# List sessions
mc ls ambient/ambient-sessions/

# List session artifacts
mc ls ambient/ambient-sessions/my-project/session-abc/artifacts/

# Download artifacts
mc cp --recursive ambient/ambient-sessions/my-project/session-abc/artifacts/ ./local-dir/

# Download session history
mc cp --recursive ambient/ambient-sessions/my-project/session-abc/.claude/ ./.claude/
```

### Via kubectl exec

```bash
# Get MinIO pod
MINIO_POD=$(kubectl get pod -l app=minio -n ambient-code -o jsonpath='{.items[0].metadata.name}')

# List sessions
kubectl exec -n ambient-code "${MINIO_POD}" -- mc ls local/ambient-sessions/

# Download file
kubectl exec -n ambient-code "${MINIO_POD}" -- mc cp "local/ambient-sessions/my-project/session-abc/artifacts/report.pdf" /tmp/
kubectl cp "ambient-code/${MINIO_POD}:/tmp/report.pdf" ./report.pdf
```

## Management Commands

```bash
# Check MinIO status
make minio-status NAMESPACE=ambient-code

# View MinIO logs
make minio-logs NAMESPACE=ambient-code

# Port forward to MinIO API (for mc commands)
kubectl port-forward svc/minio 9000:9000 -n ambient-code
```

## Bucket Lifecycle Management

### Set Auto-Delete Policy

Keep storage costs down by auto-deleting old sessions:

```bash
# Create lifecycle policy
cat > /tmp/lifecycle.json << 'EOF'
{
  "Rules": [
    {
      "ID": "expire-old-sessions",
      "Status": "Enabled",
      "Expiration": {
        "Days": 30
      }
    }
  ]
}
EOF

# Apply policy
kubectl exec -n ambient-code "${MINIO_POD}" -- mc ilm import "local/ambient-sessions" /tmp/lifecycle.json
```

### Monitor Storage Usage

```bash
# Check bucket size
kubectl exec -n ambient-code "${MINIO_POD}" -- mc du local/ambient-sessions

# List largest sessions
kubectl exec -n ambient-code "${MINIO_POD}" -- mc du --depth 2 local/ambient-sessions | sort -n -r | head -10
```

## Backup and Restore

### Backup MinIO Data

```bash
# Backup to local directory
kubectl exec -n ambient-code "${MINIO_POD}" -- mc mirror local/ambient-sessions /tmp/backup/
kubectl cp "ambient-code/${MINIO_POD}:/tmp/backup" ./minio-backup/

# Or use external mc client
mc mirror ambient/ambient-sessions ./minio-backup/
```

### Restore from Backup

```bash
# Copy backup to pod
kubectl cp ./minio-backup/ "ambient-code/${MINIO_POD}:/tmp/restore"

# Restore
kubectl exec -n ambient-code "${MINIO_POD}" -- mc mirror /tmp/restore local/ambient-sessions
```

## Troubleshooting

### MinIO Pod Not Starting

```bash
# Check events
kubectl get events -n ambient-code --sort-by='.lastTimestamp' | grep minio

# Check PVC
kubectl get pvc minio-data -n ambient-code

# Check pod logs
kubectl logs -f deployment/minio -n ambient-code
```

### Can't Access MinIO Console

```bash
# Check service
kubectl get svc minio -n ambient-code

# Test connection from within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -n ambient-code -- \
  curl -v http://minio.ambient-code.svc:9000/minio/health/live
```

### Session Init Failing

```bash
# Check session pod init container logs
kubectl logs {session-pod} -c init-hydrate -n {namespace}

# Common issues:
# - Wrong S3 endpoint (check project settings)
# - Bucket doesn't exist (create in MinIO console)
# - Wrong credentials (verify in project settings)
```

## Production Considerations

### High Availability

For production, deploy MinIO in distributed mode:

```bash
# Use MinIO Operator
kubectl apply -k "github.com/minio/operator"
kubectl apply -f - <<EOF
apiVersion: minio.min.io/v2
kind: Tenant
metadata:
  name: ambient-sessions
  namespace: ambient-code
spec:
  pools:
  - servers: 4
    volumesPerServer: 4
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 50Gi
EOF
```

### Security

1. **Change default credentials**: Use strong passwords in production
2. **Enable TLS**: Configure MinIO with TLS certificates
3. **Network policies**: Restrict access to MinIO service
4. **Encryption**: Enable server-side encryption (SSE-S3 or SSE-KMS)

### Monitoring

```bash
# Enable Prometheus metrics
kubectl exec -n ambient-code "${MINIO_POD}" -- mc admin prometheus generate local

# Access metrics
kubectl port-forward svc/minio 9000:9000 -n ambient-code
curl http://localhost:9000/minio/v2/metrics/cluster
```

## Alternative: External S3

If you prefer AWS S3 or another provider:

1. **Skip MinIO deployment**: Don't apply `minio-deployment.yaml`
2. **Configure in project settings**:
   - S3_ENDPOINT: `https://s3.amazonaws.com` (or your provider)
   - S3_BUCKET: Your bucket name
   - S3_ACCESS_KEY: IAM access key
   - S3_SECRET_KEY: IAM secret key
3. **Ensure bucket exists** in your S3 provider
4. **Set IAM permissions**: GetObject, PutObject, DeleteObject, ListBucket

## Next Steps

- [S3 Storage Configuration](s3-storage-configuration.md) - Detailed S3 setup
- [Create your first session](../getting-started.md) - Test S3 persistence
- [MinIO Documentation](https://min.io/docs/minio/kubernetes/upstream/) - Official MinIO docs

