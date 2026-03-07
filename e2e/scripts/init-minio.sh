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
  echo "MinIO pod not found"
  exit 1
fi

echo "MinIO pod: $MINIO_POD"

# Get MinIO credentials from secret
MINIO_USER=$(kubectl get secret -n ambient-code minio-credentials -o jsonpath='{.data.root-user}' | base64 -d)
MINIO_PASSWORD=$(kubectl get secret -n ambient-code minio-credentials -o jsonpath='{.data.root-password}' | base64 -d)

# Retry loop: MinIO pod can be "Ready" before the S3 API accepts connections
MAX_RETRIES=10
RETRY_DELAY=3

echo "Waiting for MinIO S3 API to accept connections..."
for i in $(seq 1 $MAX_RETRIES); do
  if output=$(kubectl exec -n ambient-code "$MINIO_POD" -- mc alias set myminio http://localhost:9000 "$MINIO_USER" "$MINIO_PASSWORD" 2>&1); then
    echo "MinIO S3 API is ready"
    break
  fi
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "Failed to connect to MinIO after $MAX_RETRIES attempts"
    echo "Last error: $output"
    exit 1
  fi
  echo "  Attempt $i/$MAX_RETRIES failed, retrying in ${RETRY_DELAY}s..."
  sleep "$RETRY_DELAY"
done

echo "Creating ambient-sessions bucket..."
if kubectl exec -n ambient-code "$MINIO_POD" -- mc mb myminio/ambient-sessions 2>/dev/null; then
  echo "  Created ambient-sessions bucket"
else
  echo "  Bucket may already exist, verifying..."
fi

# Verify bucket exists
if kubectl exec -n ambient-code "$MINIO_POD" -- mc ls myminio/ | grep -q ambient-sessions; then
  echo "  ambient-sessions bucket ready"
else
  echo "  Failed to verify ambient-sessions bucket"
  exit 1
fi

echo ""
echo "MinIO initialized successfully!"
