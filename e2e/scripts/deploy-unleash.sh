#!/bin/bash
set -euo pipefail

# Deploy Unleash feature flag server to Kubernetes/OpenShift
# Uses Helm chart with bundled PostgreSQL (same pattern as deploy-langfuse.sh)

# Parse command line arguments
PLATFORM="auto"
NAMESPACE="unleash"
while [[ $# -gt 0 ]]; do
  case $1 in
    --openshift|--crc)
      PLATFORM="openshift"
      shift
      ;;
    --kubernetes|--k8s|--kind)
      PLATFORM="kubernetes"
      shift
      ;;
    --namespace|-n)
      NAMESPACE="$2"
      shift 2
      ;;
    --help|-h)
      echo "Usage: $0 [--openshift|--kubernetes] [--namespace <name>]"
      echo ""
      echo "Options:"
      echo "  --openshift, --crc   Force OpenShift mode (use oc, create Route)"
      echo "  --kubernetes, --kind Force Kubernetes mode (use kubectl, create Ingress)"
      echo "  --namespace, -n      Target namespace (default: unleash)"
      echo "  (default)            Auto-detect based on available CLI and cluster type"
      echo ""
      echo "After deployment, connect the backend by setting:"
      echo "  UNLEASH_URL=http://unleash.<namespace>.svc.cluster.local:4242/api"
      echo "  UNLEASH_CLIENT_KEY=default:development.unleash-client-token"
      echo "  UNLEASH_ADMIN_URL=http://unleash.<namespace>.svc.cluster.local:4242"
      echo "  UNLEASH_ADMIN_TOKEN=*:*.unleash-admin-token"
      echo ""
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

echo "======================================"
echo "Deploying Unleash Feature Flag Server"
echo "======================================"
echo ""

# Detect platform if auto mode
if [ "$PLATFORM" = "auto" ]; then
  echo "Auto-detecting platform..."

  # Check if oc is available and we're on OpenShift
  if command -v oc &> /dev/null; then
    if oc api-resources --api-group=route.openshift.io &>/dev/null 2>&1; then
      PLATFORM="openshift"
      echo "   Detected OpenShift cluster"
    else
      PLATFORM="kubernetes"
      echo "   Detected Kubernetes cluster (oc CLI available)"
    fi
  elif command -v kubectl &> /dev/null; then
    PLATFORM="kubernetes"
    echo "   Detected Kubernetes cluster"
  else
    echo "Neither kubectl nor oc found. Please install Kubernetes CLI."
    exit 1
  fi
  echo ""
fi

# Set CLI tool based on platform
if [ "$PLATFORM" = "openshift" ]; then
  CLI="oc"
  PLATFORM_NAME="OpenShift"
else
  CLI="kubectl"
  PLATFORM_NAME="Kubernetes"
fi

echo "Platform: $PLATFORM_NAME"
echo "CLI: $CLI"
echo "Namespace: $NAMESPACE"
echo ""

# Check prerequisites
if ! command -v helm &> /dev/null; then
  echo "Helm not found. Please install Helm 3.x first."
  echo "   Visit: https://helm.sh/docs/intro/install/"
  exit 1
fi

# Check cluster connection
if ! $CLI cluster-info &>/dev/null; then
  echo "Not connected to $PLATFORM_NAME cluster"
  if [ "$PLATFORM" = "openshift" ]; then
    echo "   Please run: $CLI login <cluster-url>"
  else
    echo "   Please configure kubectl: kubectl config use-context <context>"
  fi
  exit 1
fi

CLUSTER_USER=$($CLI config view --minify -o jsonpath='{.contexts[0].context.user}' 2>/dev/null || echo "unknown")
CLUSTER_URL=$($CLI config view --minify -o jsonpath='{.clusters[0].cluster.server}')
echo "Connected to $PLATFORM_NAME:"
echo "   User: $CLUSTER_USER"
echo "   Cluster: $CLUSTER_URL"
echo ""

# Prompt for credentials or use defaults for testing (same pattern as Langfuse)
read -p "Use simple test passwords? (y/n, default: y): " USE_TEST_CREDS
USE_TEST_CREDS=${USE_TEST_CREDS:-y}

if [[ "$USE_TEST_CREDS" =~ ^[Yy]$ ]]; then
  echo "Setting simple passwords for test environment..."
  POSTGRES_PASSWORD="postgres123"
  echo "   Test credentials configured"
else
  echo "Generating secure random credentials..."
  POSTGRES_PASSWORD=$(openssl rand -base64 32)
  echo "   Secure credentials generated"
fi

# Add Bitnami Helm repository (for PostgreSQL)
echo ""
echo "Adding Helm repositories..."
helm repo add bitnami https://charts.bitnami.com/bitnami &>/dev/null || true
helm repo update &>/dev/null
echo "   Helm repositories updated"

# Create namespace
echo ""
echo "Creating namespace '$NAMESPACE'..."
if $CLI get namespace "$NAMESPACE" &>/dev/null; then
  echo "   Namespace '$NAMESPACE' already exists"
else
  $CLI create namespace "$NAMESPACE"
  echo "   Namespace created"
fi

# Deploy PostgreSQL using Bitnami Helm chart (same pattern as Langfuse)
echo ""
echo "Installing PostgreSQL with Helm..."
echo "   (This may take 2-3 minutes...)"
echo ""

helm upgrade --install unleash-postgresql bitnami/postgresql \
  --namespace "$NAMESPACE" \
  --set auth.username=unleash \
  --set auth.password="$POSTGRES_PASSWORD" \
  --set auth.database=unleash \
  --set primary.podAntiAffinityPreset=none \
  --set primary.persistence.size=1Gi \
  --set primary.resources.requests.memory=256Mi \
  --set primary.resources.limits.memory=512Mi \
  --set primary.resources.requests.cpu=100m \
  --set primary.resources.limits.cpu=500m \
  --wait \
  --timeout=5m

echo "   PostgreSQL installed"

# Wait for PostgreSQL to be ready
echo ""
echo "Waiting for PostgreSQL to be ready..."
$CLI wait --namespace "$NAMESPACE" \
  --for=condition=ready \
  --timeout=180s \
  pod -l app.kubernetes.io/name=postgresql
echo "   PostgreSQL is ready"

# Deploy Unleash using raw manifests (no official Helm chart with good defaults)
echo ""
echo "Deploying Unleash server..."

# Create Unleash secret with database URL
DATABASE_URL="postgres://unleash:${POSTGRES_PASSWORD}@unleash-postgresql:5432/unleash"
$CLI create secret generic unleash-secrets \
  --namespace "$NAMESPACE" \
  --from-literal=DATABASE_URL="$DATABASE_URL" \
  --dry-run=client -o yaml | $CLI apply -f -

cat <<EOF | $CLI apply -n "$NAMESPACE" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: unleash
  labels:
    app: unleash
    app.kubernetes.io/name: unleash
spec:
  replicas: 1
  selector:
    matchLabels:
      app: unleash
  template:
    metadata:
      labels:
        app: unleash
        app.kubernetes.io/name: unleash
    spec:
      containers:
      - name: unleash
        image: unleashorg/unleash-server:6
        ports:
        - containerPort: 4242
          name: http
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: unleash-secrets
              key: DATABASE_URL
        - name: DATABASE_SSL
          value: "false"
        - name: LOG_LEVEL
          value: "info"
        - name: INIT_ADMIN_API_TOKENS
          value: "*:*.unleash-admin-token"
        - name: INIT_CLIENT_API_TOKENS
          value: "default:development.unleash-client-token"
        - name: INIT_FRONTEND_API_TOKENS
          value: "default:development.unleash-frontend-token"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        readinessProbe:
          httpGet:
            path: /health
            port: 4242
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
        livenessProbe:
          httpGet:
            path: /health
            port: 4242
          initialDelaySeconds: 60
          periodSeconds: 30
          timeoutSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: unleash
  labels:
    app: unleash
    app.kubernetes.io/name: unleash
spec:
  selector:
    app: unleash
  ports:
  - port: 4242
    targetPort: 4242
    name: http
EOF
echo "   Unleash server deployed"

# Wait for Unleash to be ready
echo ""
echo "Waiting for Unleash to be ready (this may take 1-2 minutes)..."
$CLI wait --namespace "$NAMESPACE" \
  --for=condition=available \
  --timeout=180s \
  deployment/unleash
echo "   Unleash is ready"

# Create Route (OpenShift) or Ingress (Kubernetes)
if [ "$PLATFORM" = "openshift" ]; then
  echo ""
  echo "Creating OpenShift Route..."
  $CLI create route edge unleash \
    --service=unleash \
    --port=4242 \
    --namespace "$NAMESPACE" \
    --dry-run=client -o yaml | $CLI apply -f -

  UNLEASH_EXTERNAL_URL="https://$($CLI get route unleash -n "$NAMESPACE" -o jsonpath='{.spec.host}')"
  echo "   Route created: $UNLEASH_EXTERNAL_URL"
else
  echo ""
  echo "Creating Kubernetes Ingress..."
  cat <<EOF | $CLI apply -n "$NAMESPACE" -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: unleash
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: unleash.localhost
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: unleash
            port:
              number: 4242
EOF
  UNLEASH_EXTERNAL_URL="http://unleash.localhost"
  echo "   Ingress created"
  echo "   Note: For kind, use port-forward instead: make unleash-port-forward"
fi

# Print summary
echo ""
echo "======================================"
echo "Unleash Deployment Complete!"
echo "======================================"
echo ""
echo "Access Unleash UI:"
if [ "$PLATFORM" = "openshift" ]; then
  echo "   URL: $UNLEASH_EXTERNAL_URL"
else
  echo "   Run: kubectl port-forward svc/unleash 4242:4242 -n $NAMESPACE"
  echo "   Then access: http://localhost:4242"
fi
echo ""
echo "Default Admin Credentials:"
echo "   Username: admin"
echo "   Password: unleash4all"
echo ""
echo "Pre-configured API Tokens:"
echo "   Admin API Token:    *:*.unleash-admin-token"
echo "   Client API Token:   default:development.unleash-client-token"
echo "   Frontend API Token: default:development.unleash-frontend-token"
echo ""
echo "Backend Configuration (add to deployment env):"
echo "   UNLEASH_URL=http://unleash.$NAMESPACE.svc.cluster.local:4242/api"
echo "   UNLEASH_CLIENT_KEY=default:development.unleash-client-token"
echo "   UNLEASH_ADMIN_URL=http://unleash.$NAMESPACE.svc.cluster.local:4242"
echo "   UNLEASH_ADMIN_TOKEN=*:*.unleash-admin-token"
echo "   UNLEASH_PROJECT=default"
echo "   UNLEASH_ENVIRONMENT=development"
echo ""
echo "Frontend Configuration (if using direct Unleash proxy):"
echo "   NEXT_PUBLIC_UNLEASH_URL=http://unleash.$NAMESPACE.svc.cluster.local:4242/api/frontend"
echo "   NEXT_PUBLIC_UNLEASH_CLIENT_KEY=default:development.unleash-frontend-token"
echo ""
echo "Credentials used:"
echo "   PostgreSQL: $POSTGRES_PASSWORD"
echo ""
