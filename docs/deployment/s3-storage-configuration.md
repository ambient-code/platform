# S3 Storage Configuration

## Overview

The Ambient Code Platform uses S3-compatible storage to persist session state, artifacts, and user uploads across pod restarts. This document explains how to configure S3 storage per project and access session data.

## Why S3 Storage?

Sessions no longer use Persistent Volume Claims (PVCs) for storage. Instead:

- **Session pods use EmptyDir** - Fast startup, no PVC provisioning delays
- **State persists in S3** - Session history, artifacts, and uploads are automatically synced to S3
- **Code persists in Git** - Code changes are managed via git push (user or agent driven)

**Benefits:**
- Fast session startup (no PVC provisioning wait)
- Cost-effective storage (S3 is cheaper than block storage)
- Per-project isolation (each project configures its own S3 bucket)
- Easy artifact access (browse S3 bucket directly)

## S3 Storage Structure

```
s3://your-bucket/
  └── {namespace}/
      └── {session-name}/
          ├── .claude/           # Claude session history for resume
          ├── artifacts/         # Generated files
          ├── uploads/           # User uploaded files
          └── metadata.json      # Sync metadata
```

## Configuration

### 1. Configure S3 in Project Settings

Navigate to your project's settings page:

```
https://your-deployment/projects/{project-name}?section=settings
```

Find the **S3 Storage Configuration** section and configure:

| Field | Description | Example |
|-------|-------------|---------|
| **Enable S3 Storage** | Enable/disable S3 persistence | ✅ Checked |
| **S3_ENDPOINT** | S3-compatible endpoint | `https://s3.amazonaws.com` |
| **S3_BUCKET** | Bucket name for session storage | `ambient-sessions` |
| **S3_REGION** | AWS region (optional) | `us-east-1` |
| **S3_ACCESS_KEY** | S3 access key ID | `AKIAIOSFODNN7EXAMPLE` |
| **S3_SECRET_KEY** | S3 secret access key | `wJalrXUtnFEMI/K7MDENG/...` |

Click **Save Integration Secrets** to persist the configuration.

### 2. Create S3 Bucket (AWS)

If using AWS S3:

```bash
# Create bucket
aws s3 mb s3://ambient-sessions --region us-east-1

# Set bucket lifecycle policy (optional - auto-delete old sessions)
aws s3api put-bucket-lifecycle-configuration \
  --bucket ambient-sessions \
  --lifecycle-configuration file://lifecycle.json
```

**lifecycle.json:**
```json
{
  "Rules": [
    {
      "Id": "DeleteOldSessions",
      "Status": "Enabled",
      "Prefix": "",
      "Expiration": {
        "Days": 30
      }
    }
  ]
}
```

### 3. Create IAM User and Policy (AWS)

Create an IAM user with the following policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::ambient-sessions",
        "arn:aws:s3:::ambient-sessions/*"
      ]
    }
  ]
}
```

### 4. Using MinIO (Self-Hosted)

For self-hosted S3-compatible storage:

```bash
# Deploy MinIO in your cluster
kubectl create namespace minio
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minio
  namespace: minio
spec:
  replicas: 1
  selector:
    matchLabels:
      app: minio
  template:
    metadata:
      labels:
        app: minio
    spec:
      containers:
      - name: minio
        image: minio/minio:latest
        command:
        - /bin/bash
        - -c
        - minio server /data --console-address ":9001"
        ports:
        - containerPort: 9000
          name: api
        - containerPort: 9001
          name: console
        env:
        - name: MINIO_ROOT_USER
          value: "admin"
        - name: MINIO_ROOT_PASSWORD
          value: "changeme123"
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: minio-data
---
apiVersion: v1
kind: Service
metadata:
  name: minio
  namespace: minio
spec:
  ports:
  - port: 9000
    name: api
  - port: 9001
    name: console
  selector:
    app: minio
EOF

# Create bucket
mc alias set myminio http://minio.minio.svc:9000 admin changeme123
mc mb myminio/ambient-sessions
```

Configure in project settings:
- **S3_ENDPOINT**: `http://minio.minio.svc:9000`
- **S3_BUCKET**: `ambient-sessions`
- **S3_ACCESS_KEY**: `admin`
- **S3_SECRET_KEY**: `changeme123`

## Accessing Session Artifacts

### Option 1: AWS CLI

```bash
# List sessions
aws s3 ls s3://ambient-sessions/my-project/

# List artifacts for a session
aws s3 ls s3://ambient-sessions/my-project/session-abc/artifacts/

# Download artifacts
aws s3 cp s3://ambient-sessions/my-project/session-abc/artifacts/ ./local-dir/ --recursive

# Download session history
aws s3 cp s3://ambient-sessions/my-project/session-abc/.claude/ ./.claude/ --recursive
```

### Option 2: MinIO Client (mc)

```bash
# Configure alias
mc alias set ambient http://minio.minio.svc:9000 admin changeme123

# List sessions
mc ls ambient/ambient-sessions/my-project/

# Download artifacts
mc cp --recursive ambient/ambient-sessions/my-project/session-abc/artifacts/ ./local-dir/
```

### Option 3: S3 Browser

Use any S3 browser tool:
- [Cyberduck](https://cyberduck.io/)
- [S3 Browser](https://s3browser.com/)
- [MinIO Console](https://min.io/docs/minio/kubernetes/upstream/operations/monitoring/minio-console.html) (for MinIO)

## Behavior Without S3

If S3 is not configured or disabled:

- ✅ Sessions will still work
- ⚠️ Session state is ephemeral (lost on pod restart)
- ⚠️ No session resume functionality
- ⚠️ Artifacts lost when pod terminates

**Recommended:** Always configure S3 for production use.

## Troubleshooting

### Session Fails to Start

Check operator logs:
```bash
oc logs -f deployment/agentic-operator -n ambient-code | grep S3
```

Common errors:
- **"incomplete S3 configuration"** - Missing endpoint, bucket, or credentials
- **"S3 disabled in project settings"** - S3_ENABLED is set to "false"
- **"failed to read project secret"** - Permission issue accessing project secret

### Init Container Fails

Check init-hydrate logs:
```bash
oc logs {pod-name} -c init-hydrate -n {namespace}
```

Common errors:
- **"connection refused"** - S3 endpoint unreachable
- **"access denied"** - Invalid credentials or missing permissions
- **"bucket not found"** - Bucket doesn't exist, create it first

### State-Sync Sidecar Not Syncing

Check state-sync logs:
```bash
oc logs {pod-name} -c state-sync -n {namespace} -f
```

Look for sync confirmations:
```
[2024-12-22T15:30:00Z] Starting sync to S3...
  Syncing .claude/...
  Syncing artifacts/...
[2024-12-22T15:30:05Z] Sync complete (2 paths synced)
```

## Security Considerations

### Credential Management

- **Per-project credentials**: Each project stores its own S3 credentials
- **Kubernetes Secrets**: Credentials stored in `ambient-non-vertex-integrations` secret
- **No global credentials**: No cluster-wide S3 secret

### Bucket Isolation

- **Use separate buckets per environment**: `ambient-sessions-prod`, `ambient-sessions-dev`
- **Use bucket policies**: Restrict access to specific IAM users/roles
- **Enable encryption**: Use S3 bucket encryption (SSE-S3 or SSE-KMS)

### Access Control

Example bucket policy for multi-tenant isolation:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::123456789012:user/project-a-user"
      },
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": "arn:aws:s3:::ambient-sessions/project-a/*"
    }
  ]
}
```

## Migration from PVCs

If you have existing sessions with PVCs:

1. **Existing sessions continue working** - They'll use their PVCs until completion
2. **New sessions use S3** - EmptyDir + S3 sync automatically
3. **Clean up old PVCs** - After sessions complete:

```bash
# List orphaned PVCs
oc get pvc -n {namespace} -l app=ambient-workspace

# Delete old PVCs
oc delete pvc ambient-workspace-{session-name} -n {namespace}
```

## Performance Tuning

### Sync Interval

Default: 60 seconds. Adjust via operator config:

```yaml
# components/manifests/base/operator-deployment.yaml
env:
  - name: SYNC_INTERVAL
    value: "30"  # Sync every 30 seconds (more frequent = safer but higher cost)
```

### Max Sync Size

Default: 1GB. Prevent runaway storage costs:

```yaml
env:
  - name: MAX_SYNC_SIZE
    value: "2147483648"  # 2GB limit
```

### Exclude Patterns

Customize what gets synced by editing `sync.sh`:

```bash
EXCLUDE_PATTERNS=(
    "repos/**"           # Git handles this
    "node_modules/**"    # Add more as needed
    ".venv/**"
    "__pycache__/**"
)
```

## FAQ

### Q: What happens if S3 is down?

**A:** Session pods will fail to start if S3 is unreachable during hydration. The init-hydrate container will timeout and pod creation will fail.

### Q: Can I use multiple S3 endpoints for different projects?

**A:** Yes! Each project configures its own S3 endpoint and bucket. Projects can use different S3 providers.

### Q: How do I migrate session state between buckets?

**A:** Use `rclone` or `aws s3 sync` to copy data between buckets:

```bash
rclone sync s3-old:old-bucket/namespace/session s3-new:new-bucket/namespace/session
```

### Q: What if I don't configure S3?

**A:** Sessions work but use ephemeral storage only. State is lost on pod restart. Not recommended for production.

### Q: How much does S3 storage cost?

**A:** Varies by provider. AWS S3 Standard: ~$0.023/GB/month. For 100 sessions @ 100MB each: ~$0.23/month.

## Support

For issues or questions:
- Check operator logs: `oc logs -f deployment/agentic-operator -n ambient-code`
- Check pod logs: `oc logs {pod-name} -c state-sync -n {namespace}`
- File an issue: [GitHub Issues](https://github.com/gkrumbac/vTeam/issues)

