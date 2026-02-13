#!/bin/bash
#
# setup-vertex-kind.sh - Configure Vertex AI for Ambient Code Platform on kind
#
# This script configures a kind cluster to use Google Cloud Vertex AI instead of
# the Anthropic API for Claude. This is useful when you have Vertex AI access
# but not a direct Anthropic API key.
#
# PREREQUISITES:
#   1. kind cluster running: make kind-up
#   2. GCP service account with Vertex AI permissions
#   3. Claude Code CLI installed (uses the same environment variables)
#
# REQUIRED ENVIRONMENT VARIABLES:
#   These are the same variables used by Claude Code CLI for Vertex AI:
#
#   GOOGLE_APPLICATION_CREDENTIALS
#       Path to your GCP service account JSON key file.
#       Example: /Users/you/.config/gcloud/ambient-code-sa.json
#
#   ANTHROPIC_VERTEX_PROJECT_ID
#       Your GCP project ID where Claude is enabled on Vertex AI.
#       Example: my-gcp-project-123
#
#   CLOUD_ML_REGION
#       GCP region for Vertex AI API calls.
#       Example: us-east5 (Claude is available in limited regions)
#
# SETUP:
#   Add these to your shell profile (~/.zshrc or ~/.bashrc):
#
#     export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/gcloud/your-sa-key.json"
#     export ANTHROPIC_VERTEX_PROJECT_ID="your-gcp-project-id"
#     export CLOUD_ML_REGION="us-east5"
#
#   Then reload your shell: source ~/.zshrc
#
# USAGE:
#   ./scripts/setup-vertex-kind.sh
#
#   Or with a custom namespace:
#   NAMESPACE=my-namespace ./scripts/setup-vertex-kind.sh
#
# WHAT THIS SCRIPT DOES:
#   1. Creates a Kubernetes secret with your GCP service account credentials
#   2. Patches the operator-config ConfigMap to enable Vertex AI mode
#   3. Restarts the operator to pick up the new configuration
#
# VERIFICATION:
#   After running, check operator logs:
#   kubectl logs -l app=agentic-operator -n ambient-code | grep -i vertex
#

set -e

# Show help if requested
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    head -50 "$0" | tail -n +2 | sed 's/^# //' | sed 's/^#//'
    exit 0
fi

NAMESPACE="${NAMESPACE:-ambient-code}"

echo "=== Vertex AI Setup for Kind Cluster ==="
echo ""

# Check if kubectl can reach the cluster
if ! kubectl cluster-info &>/dev/null; then
    echo "Error: Cannot connect to Kubernetes cluster"
    echo "Make sure your kind cluster is running: make kind-up"
    exit 1
fi

# Check if namespace exists
if ! kubectl get namespace "$NAMESPACE" &>/dev/null; then
    echo "Error: Namespace '$NAMESPACE' does not exist"
    echo "Deploy the platform first: make deploy"
    exit 1
fi

# Check required environment variables
echo "Checking environment variables (same as Claude Code CLI)..."
echo ""

missing_vars=0

if [ -z "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
    echo "  [MISSING] GOOGLE_APPLICATION_CREDENTIALS"
    echo "            Path to your GCP service account JSON key file"
    echo ""
    missing_vars=1
elif [ ! -f "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
    echo "  [ERROR]   GOOGLE_APPLICATION_CREDENTIALS=$GOOGLE_APPLICATION_CREDENTIALS"
    echo "            File does not exist!"
    echo ""
    missing_vars=1
else
    echo "  [OK] GOOGLE_APPLICATION_CREDENTIALS=$GOOGLE_APPLICATION_CREDENTIALS"
fi

if [ -z "$ANTHROPIC_VERTEX_PROJECT_ID" ]; then
    echo "  [MISSING] ANTHROPIC_VERTEX_PROJECT_ID"
    echo "            Your GCP project ID with Claude on Vertex AI"
    echo ""
    missing_vars=1
else
    echo "  [OK] ANTHROPIC_VERTEX_PROJECT_ID=$ANTHROPIC_VERTEX_PROJECT_ID"
fi

if [ -z "$CLOUD_ML_REGION" ]; then
    echo "  [MISSING] CLOUD_ML_REGION"
    echo "            GCP region (e.g., us-east5)"
    echo ""
    missing_vars=1
else
    echo "  [OK] CLOUD_ML_REGION=$CLOUD_ML_REGION"
fi

if [ $missing_vars -eq 1 ]; then
    echo ""
    echo "Add missing variables to your ~/.zshrc or ~/.bashrc:"
    echo ""
    echo '  export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/gcloud/your-sa-key.json"'
    echo '  export ANTHROPIC_VERTEX_PROJECT_ID="your-gcp-project-id"'
    echo '  export CLOUD_ML_REGION="us-east5"'
    echo ""
    echo "Then reload: source ~/.zshrc"
    exit 1
fi

echo ""

# Step 1: Create the secret with credentials
echo "Step 1/3: Creating ambient-vertex secret..."
kubectl delete secret ambient-vertex -n "$NAMESPACE" 2>/dev/null || true
kubectl create secret generic ambient-vertex \
  --from-file=ambient-code-key.json="$GOOGLE_APPLICATION_CREDENTIALS" \
  -n "$NAMESPACE"
echo "  Done"
echo ""

# Step 2: Patch the operator-config ConfigMap with env vars
echo "Step 2/3: Patching operator-config ConfigMap..."
kubectl patch configmap operator-config -n "$NAMESPACE" --type merge -p "{
  \"data\": {
    \"CLAUDE_CODE_USE_VERTEX\": \"1\",
    \"ANTHROPIC_VERTEX_PROJECT_ID\": \"$ANTHROPIC_VERTEX_PROJECT_ID\",
    \"CLOUD_ML_REGION\": \"$CLOUD_ML_REGION\",
    \"GOOGLE_APPLICATION_CREDENTIALS\": \"/app/vertex/ambient-code-key.json\"
  }
}"
echo "  Done"
echo ""

# Step 3: Restart operator to pick up changes
echo "Step 3/3: Restarting operator to apply changes..."
kubectl rollout restart deployment agentic-operator -n "$NAMESPACE"
kubectl rollout status deployment agentic-operator -n "$NAMESPACE" --timeout=60s
echo ""

echo "=== Setup Complete ==="
echo ""
echo "Vertex AI is now configured for the kind cluster."
echo ""
echo "Configuration applied:"
echo "  - Namespace: $NAMESPACE"
echo "  - Project:   $ANTHROPIC_VERTEX_PROJECT_ID"
echo "  - Region:    $CLOUD_ML_REGION"
echo ""

# Verify Vertex mode is active in operator logs
echo "Verifying Vertex AI configuration..."
sleep 3
if kubectl logs -l app=agentic-operator -n "$NAMESPACE" --tail=100 2>/dev/null | grep -qi "vertex ai mode enabled"; then
    echo "  ✓ Vertex AI mode is active"
else
    echo "  ⚠ Could not verify Vertex mode in logs yet"
    echo "    Check manually: kubectl logs -l app=agentic-operator -n $NAMESPACE | grep -i vertex"
fi
echo ""

echo "Next steps:"
echo "  1. Create a session via the UI at http://localhost:8080"
echo ""
echo "To switch back to Anthropic API, update the ConfigMap:"
echo "  kubectl patch configmap operator-config -n $NAMESPACE --type merge \\"
echo "    -p '{\"data\":{\"CLAUDE_CODE_USE_VERTEX\":\"0\"}}'"
echo "  kubectl rollout restart deployment agentic-operator -n $NAMESPACE"
