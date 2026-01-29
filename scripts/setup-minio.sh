#!/bin/bash
# setup-minio.sh - Set up MinIO and create initial bucket for Ambient Code sessions

set -e

NAMESPACE="${NAMESPACE:-ambient-code}"
BUCKET_NAME="${BUCKET_NAME:-ambient-sessions}"

# Get credentials from existing secret (more secure than defaults)
if kubectl get secret minio-credentials -n "${NAMESPACE}" >/dev/null 2>&1; then
    MINIO_USER=$(kubectl get secret minio-credentials -n "${NAMESPACE}" -o jsonpath='{.data.root-user}' | base64 -d)
    MINIO_PASSWORD=$(kubectl get secret minio-credentials -n "${NAMESPACE}" -o jsonpath='{.data.root-password}' | base64 -d)
else
    echo "ERROR: minio-credentials secret not found in namespace ${NAMESPACE}"
    echo "Please create it first:"
    echo "  1. Copy components/manifests/base/minio-credentials-secret.yaml.example to minio-credentials-secret.yaml"
    echo "  2. Edit with secure credentials"
    echo "  3. kubectl apply -f minio-credentials-secret.yaml -n ${NAMESPACE}"
    exit 1
fi

echo "========================================="
echo "MinIO Setup for Ambient Code Platform"
echo "========================================="
echo "Namespace: ${NAMESPACE}"
echo "Bucket: ${BUCKET_NAME}"
echo "========================================="

# Check if MinIO is deployed
echo "Checking MinIO deployment..."
if ! kubectl get deployment minio -n "${NAMESPACE}" >/dev/null 2>&1; then
    echo "Error: MinIO deployment not found in namespace ${NAMESPACE}"
    echo "Deploy MinIO first: kubectl apply -f components/manifests/base/minio-deployment.yaml"
    exit 1
fi

# Wait for MinIO to be ready
echo "Waiting for MinIO to be ready..."
kubectl wait --for=condition=ready pod -l app=minio -n "${NAMESPACE}" --timeout=120s

# Get MinIO pod name
MINIO_POD=$(kubectl get pod -l app=minio -n "${NAMESPACE}" -o jsonpath='{.items[0].metadata.name}')
echo "MinIO pod: ${MINIO_POD}"

# Set up MinIO client alias
echo "Configuring MinIO client..."
kubectl exec -n "${NAMESPACE}" "${MINIO_POD}" -- mc alias set local http://localhost:9000 "${MINIO_USER}" "${MINIO_PASSWORD}"

# Create bucket if it doesn't exist
echo "Creating bucket: ${BUCKET_NAME}..."
if kubectl exec -n "${NAMESPACE}" "${MINIO_POD}" -- mc ls "local/${BUCKET_NAME}" >/dev/null 2>&1; then
    echo "Bucket ${BUCKET_NAME} already exists"
else
    kubectl exec -n "${NAMESPACE}" "${MINIO_POD}" -- mc mb "local/${BUCKET_NAME}"
    echo "Created bucket: ${BUCKET_NAME}"
fi

# Set bucket to private (default)
echo "Setting bucket policy..."
kubectl exec -n "${NAMESPACE}" "${MINIO_POD}" -- mc anonymous set none "local/${BUCKET_NAME}"

# Enable versioning (optional - helps with recovery)
echo "Enabling versioning..."
kubectl exec -n "${NAMESPACE}" "${MINIO_POD}" -- mc version enable "local/${BUCKET_NAME}"

# Show bucket info
echo ""
echo "========================================="
echo "MinIO Setup Complete!"
echo "========================================="
echo "Bucket: ${BUCKET_NAME}"
echo "Endpoint: http://minio.${NAMESPACE}.svc:9000"
echo ""
echo "MinIO Console Access:"
echo "  kubectl port-forward svc/minio 9001:9001 -n ${NAMESPACE}"
echo "  Then open: http://localhost:9001"
echo "  Login: ${MINIO_USER} / ${MINIO_PASSWORD}"
echo ""
echo "Configure in Project Settings:"
echo "  S3_ENDPOINT: http://minio.${NAMESPACE}.svc:9000"
echo "  S3_BUCKET: ${BUCKET_NAME}"
echo "  S3_ACCESS_KEY: ${MINIO_USER}"
echo "  S3_SECRET_KEY: ${MINIO_PASSWORD}"
echo "========================================="

