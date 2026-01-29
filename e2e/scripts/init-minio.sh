#!/bin/bash
set -euo pipefail

echo "======================================"
echo "Initializing MinIO Storage"
echo "======================================"

# Wait for MinIO pod to be ready
echo "Waiting for MinIO pod..."
kubectl wait --for=condition=ready --timeout=60s pod -n ambient-code -l app=minio

# Get MinIO pod name
MINIO_POD=$(kubectl get pod -n ambient-code -l app=minio -o jsonpath='{.items[0].metadata.name}')

if [ -z "$MINIO_POD" ]; then
  echo "❌ MinIO pod not found"
  exit 1
fi

echo "MinIO pod: $MINIO_POD"

# Get MinIO credentials from secret
MINIO_USER=$(kubectl get secret -n ambient-code minio-credentials -o jsonpath='{.data.root-user}' | base64 -d)
MINIO_PASSWORD=$(kubectl get secret -n ambient-code minio-credentials -o jsonpath='{.data.root-password}' | base64 -d)

echo "Setting up MinIO alias..."
kubectl exec -n ambient-code $MINIO_POD -- mc alias set myminio http://localhost:9000 $MINIO_USER $MINIO_PASSWORD 2>/dev/null || {
  echo "❌ Failed to connect to MinIO"
  exit 1
}

echo "Creating ambient-sessions bucket..."
kubectl exec -n ambient-code $MINIO_POD -- mc mb myminio/ambient-sessions 2>/dev/null || {
  echo "   ℹ️  Bucket may already exist, verifying..."
}

# Verify bucket exists
kubectl exec -n ambient-code $MINIO_POD -- mc ls myminio/ | grep -q ambient-sessions && {
  echo "   ✅ ambient-sessions bucket ready"
} || {
  echo "   ❌ Failed to verify bucket"
  exit 1
}

echo ""
echo "✅ MinIO initialized successfully!"
