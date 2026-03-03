#!/bin/bash
set -euo pipefail

echo "Waiting for all deployments to be ready..."
echo ""

# Wait for backend
echo "⏳ Waiting for backend-api..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/backend-api \
  -n ambient-code

# Wait for operator
echo "⏳ Waiting for agentic-operator..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/agentic-operator \
  -n ambient-code

# Wait for frontend
echo "⏳ Waiting for frontend..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/frontend \
  -n ambient-code

# Wait for MinIO (required for session state persistence)
echo "⏳ Waiting for minio..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/minio \
  -n ambient-code 2>/dev/null || echo "⚠️  MinIO not deployed (S3 persistence disabled)"

echo ""
echo "✅ All pods are ready!"
echo ""

# Show pod status
kubectl get pods -n ambient-code
