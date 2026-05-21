#!/usr/bin/env bash
set -euo pipefail

PR_INPUT="${1:-}"
REGISTRY="${REGISTRY:-quay.io/ambient_code}"
CLI="${OC:-oc}"

usage() {
  echo "Usage: $0 <pr-url-or-number>"
  echo "  pr-url-or-number:  e.g. https://github.com/ambient-code/platform/pull/1599  or  1599"
  echo ""
  echo "Creates namespace pr-<NUMBER>, deploys api-server + control-plane + PostgreSQL."
  echo ""
  echo "Environment variables:"
  echo "  REGISTRY   Image registry prefix (default: quay.io/ambient_code)"
  echo "  OC         oc/kubectl binary (default: oc)"
  echo "  SKIP_RBAC         Set to 1 to skip ClusterRole/ClusterRoleBinding creation"
  echo "  ANTHROPIC_API_KEY  Anthropic API key for runner pods"
  echo "  VERTEX_SOURCE_NS   Namespace to copy ambient-vertex secret from (enables Vertex AI)"
  echo ""
  echo "  Set either ANTHROPIC_API_KEY or VERTEX_SOURCE_NS for runners to work."
  exit 1
}

[[ -z "$PR_INPUT" ]] && usage

PR_NUMBER=$(echo "$PR_INPUT" | grep -oE '[0-9]+$')
if [[ -z "$PR_NUMBER" ]]; then
  echo "ERROR: Could not extract PR number from: $PR_INPUT"
  exit 1
fi

NAMESPACE="pr-${PR_NUMBER}"
IMAGE_TAG="pr-${PR_NUMBER}"

if ! $CLI get namespace "$NAMESPACE" &>/dev/null 2>&1; then
  echo "==> Creating namespace $NAMESPACE"
  $CLI create namespace "$NAMESPACE"
else
  echo "==> Namespace $NAMESPACE already exists"
fi

echo "==> Installing Ambient into $NAMESPACE (standard OpenShift mode)"
echo "    Images: ${REGISTRY}/vteam_*:${IMAGE_TAG}"
echo ""

echo "==> Step 1: Ensuring secrets"

if ! $CLI get secret ambient-api-server-db -n "$NAMESPACE" &>/dev/null; then
  echo "    Creating ambient-api-server-db secret (PostgreSQL credentials)"
  DB_PASS=$(python3 -c "import secrets; print(secrets.token_urlsafe(24))")
  $CLI create secret generic ambient-api-server-db -n "$NAMESPACE" \
    --from-literal=db.host=ambient-api-server-db \
    --from-literal=db.port=5432 \
    --from-literal=db.user=ambient \
    --from-literal=db.password="$DB_PASS" \
    --from-literal=db.name=ambient
else
  echo "    Secret OK: ambient-api-server-db"
fi

if ! $CLI get secret ambient-api-server -n "$NAMESPACE" &>/dev/null; then
  echo "    Creating ambient-api-server secret (app config — dev mode, no OIDC)"
  $CLI create secret generic ambient-api-server -n "$NAMESPACE" \
    --from-literal=clientId=dev-client \
    --from-literal=clientSecret=dev-secret
else
  echo "    Secret OK: ambient-api-server"
fi

VERTEX_SOURCE_NS="${VERTEX_SOURCE_NS:-}"
USE_VERTEX=0
VERTEX_KEY_FILE="${VERTEX_KEY_FILE:-unused}"
VERTEX_PROJECT_ID="${VERTEX_PROJECT_ID:-}"
VERTEX_REGION="${VERTEX_REGION:-global}"

if [[ -n "$VERTEX_SOURCE_NS" ]]; then
  if ! $CLI get secret ambient-vertex -n "$NAMESPACE" &>/dev/null; then
    echo "    Copying ambient-vertex secret from $VERTEX_SOURCE_NS"
    $CLI get secret ambient-vertex -n "$VERTEX_SOURCE_NS" -o json | \
      python3 -c "import json,sys; s=json.load(sys.stdin); s['metadata']={'name':s['metadata']['name'],'namespace':'$NAMESPACE'}; json.dump(s,sys.stdout)" | \
      $CLI apply -n "$NAMESPACE" -f -
  else
    echo "    Secret OK: ambient-vertex"
  fi
  USE_VERTEX=1
  VERTEX_KEY_FILE=$($CLI get secret ambient-vertex -n "$NAMESPACE" -o jsonpath='{.data}' | python3 -c "import json,sys; print(list(json.load(sys.stdin).keys())[0])")
  if [[ -z "$VERTEX_PROJECT_ID" ]]; then
    VERTEX_PROJECT_ID=$(echo "$VERTEX_KEY_FILE" | sed 's/\.json$//')-claude
    echo "    Auto-detected VERTEX_PROJECT_ID=$VERTEX_PROJECT_ID (override with VERTEX_PROJECT_ID env var)"
  fi
elif [[ -n "${ANTHROPIC_API_KEY:-}" ]]; then
  if ! $CLI get secret ambient-anthropic -n "$NAMESPACE" &>/dev/null; then
    echo "    Creating ambient-anthropic secret (Anthropic API key)"
    $CLI create secret generic ambient-anthropic -n "$NAMESPACE" \
      --from-literal=api-key="$ANTHROPIC_API_KEY"
  else
    echo "    Secret OK: ambient-anthropic"
  fi
else
  echo "    WARNING: Neither ANTHROPIC_API_KEY nor VERTEX_SOURCE_NS set — runners will fail"
fi

echo "==> Step 2: Ensuring ServiceAccount and RBAC for control-plane"

SA_NAME="ambient-control-plane"

$CLI apply -n "$NAMESPACE" -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${SA_NAME}
  namespace: ${NAMESPACE}
  labels:
    app: ambient-control-plane
EOF

if ! $CLI get secret ambient-control-plane-token -n "$NAMESPACE" &>/dev/null; then
  echo "    Creating SA token secret"
  $CLI apply -n "$NAMESPACE" -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: ambient-control-plane-token
  namespace: ${NAMESPACE}
  annotations:
    kubernetes.io/service-account.name: ${SA_NAME}
  labels:
    app: ambient-control-plane
type: kubernetes.io/service-account-token
EOF
else
  echo "    Secret OK: ambient-control-plane-token"
fi

if [[ "${SKIP_RBAC:-}" != "1" ]]; then
  echo "    Creating ClusterRole and ClusterRoleBinding"
  CR_NAME="ambient-control-plane-${NAMESPACE}"
  $CLI apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ${CR_NAME}
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: ["rbac.authorization.k8s.io"]
  resources: ["roles", "rolebindings"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["secrets", "serviceaccounts", "services", "pods"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete", "deletecollection"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["networking.k8s.io"]
  resources: ["networkpolicies"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ${CR_NAME}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ${CR_NAME}
subjects:
- kind: ServiceAccount
  name: ${SA_NAME}
  namespace: ${NAMESPACE}
EOF
else
  echo "    Skipping RBAC (SKIP_RBAC=1)"
fi

echo "==> Step 3: Deploying api-server-db (PostgreSQL)"

$CLI apply -n "$NAMESPACE" -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: ambient-api-server-db
  labels:
    app: ambient-api-server
    component: database
spec:
  ports:
  - name: postgresql
    port: 5432
    protocol: TCP
    targetPort: 5432
  selector:
    app: ambient-api-server
    component: database
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ambient-api-server-db
  labels:
    app: ambient-api-server
    component: database
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ambient-api-server
      component: database
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: ambient-api-server
        component: database
    spec:
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: postgresql
        image: registry.redhat.io/rhel9/postgresql-16:latest
        ports:
        - containerPort: 5432
          name: postgresql
        env:
        - name: POSTGRESQL_USER
          valueFrom:
            secretKeyRef:
              key: db.user
              name: ambient-api-server-db
        - name: POSTGRESQL_PASSWORD
          valueFrom:
            secretKeyRef:
              key: db.password
              name: ambient-api-server-db
        - name: POSTGRESQL_DATABASE
          valueFrom:
            secretKeyRef:
              key: db.name
              name: ambient-api-server-db
        volumeMounts:
        - name: data
          mountPath: /var/lib/pgsql/data
          subPath: pgdata
        readinessProbe:
          exec:
            command: ["/bin/sh", "-c", "pg_isready -U \"\$POSTGRESQL_USER\""]
          initialDelaySeconds: 10
          periodSeconds: 10
        livenessProbe:
          exec:
            command: ["/bin/sh", "-c", "pg_isready -U \"\$POSTGRESQL_USER\""]
          initialDelaySeconds: 30
          periodSeconds: 30
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
      volumes:
      - name: data
        emptyDir: {}
EOF

echo "==> Step 3b: Waiting for PostgreSQL to be ready"
$CLI rollout status deployment/ambient-api-server-db -n "$NAMESPACE" --timeout=120s
echo "    Waiting for pg_isready..."
for i in $(seq 1 30); do
  if $CLI exec -n "$NAMESPACE" deploy/ambient-api-server-db -- \
    /bin/sh -c 'pg_isready -U "$POSTGRESQL_USER"' &>/dev/null; then
    echo "    PostgreSQL is accepting connections"
    break
  fi
  if [[ $i -eq 30 ]]; then
    echo "ERROR: PostgreSQL did not become ready in time"
    exit 1
  fi
  sleep 2
done

echo "==> Step 4: Deploying api-server (development mode — no JWT)"

$CLI apply -n "$NAMESPACE" -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: ambient-api-server-auth
  labels:
    app: ambient-api-server
    component: auth
data:
  jwks.json: '{"keys":[]}'
  acl.yml: |
    - claim: email
      pattern: ^.*$
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ambient-api-server
  labels:
    app: ambient-api-server
    component: api
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ambient-api-server
      component: api
  template:
    metadata:
      labels:
        app: ambient-api-server
        component: api
    spec:
      serviceAccountName: default
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      initContainers:
      - name: migration
        image: ${REGISTRY}/vteam_api_server:${IMAGE_TAG}
        imagePullPolicy: Always
        command:
        - /usr/local/bin/ambient-api-server
        - migrate
        - --db-host-file=/secrets/db/db.host
        - --db-port-file=/secrets/db/db.port
        - --db-user-file=/secrets/db/db.user
        - --db-password-file=/secrets/db/db.password
        - --db-name-file=/secrets/db/db.name
        - --db-sslmode=disable
        - --alsologtostderr
        - -v=4
        volumeMounts:
        - name: db-secrets
          mountPath: /secrets/db
        resources:
          requests:
            cpu: 50m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: false
          capabilities:
            drop: ["ALL"]
      containers:
      - name: api-server
        image: ${REGISTRY}/vteam_api_server:${IMAGE_TAG}
        imagePullPolicy: Always
        command:
        - /usr/local/bin/ambient-api-server
        - serve
        - --db-host-file=/secrets/db/db.host
        - --db-port-file=/secrets/db/db.port
        - --db-user-file=/secrets/db/db.user
        - --db-password-file=/secrets/db/db.password
        - --db-name-file=/secrets/db/db.name
        - --db-sslmode=disable
        - --enable-authz=false
        - --enable-https=false
        - --api-server-bindaddress=:8000
        - --metrics-server-bindaddress=:4433
        - --health-check-server-bindaddress=:4434
        - --db-max-open-connections=50
        - --enable-db-debug=false
        - --enable-metrics-https=false
        - --http-read-timeout=5s
        - --http-write-timeout=30s
        - --cors-allowed-origins=*
        - --cors-allowed-headers=X-Ambient-Project
        - --enable-grpc=true
        - --grpc-server-bindaddress=:9000
        - --grpc-enable-tls=false
        - --alsologtostderr
        - -v=4
        env:
        - name: AMBIENT_ENV
          value: development
        - name: AMBIENT_API_TOKEN
          valueFrom:
            secretKeyRef:
              name: ambient-control-plane-token
              key: token
              optional: true
        ports:
        - name: api
          containerPort: 8000
          protocol: TCP
        - name: grpc
          containerPort: 9000
          protocol: TCP
        - name: metrics
          containerPort: 4433
          protocol: TCP
        - name: health
          containerPort: 4434
          protocol: TCP
        volumeMounts:
        - name: db-secrets
          mountPath: /secrets/db
        - name: app-secrets
          mountPath: /secrets/service
        - name: auth-config
          mountPath: /configs/authentication
        resources:
          requests:
            cpu: 200m
            memory: 512Mi
          limits:
            cpu: 1
            memory: 1Gi
        livenessProbe:
          httpGet:
            path: /api/ambient
            port: 8000
            scheme: HTTP
          initialDelaySeconds: 15
          periodSeconds: 5
        readinessProbe:
          httpGet:
            path: /healthcheck
            port: 4434
            scheme: HTTP
            httpHeaders:
            - name: User-Agent
              value: Probe
          initialDelaySeconds: 20
          periodSeconds: 10
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: false
          capabilities:
            drop: ["ALL"]
      volumes:
      - name: db-secrets
        secret:
          secretName: ambient-api-server-db
      - name: app-secrets
        secret:
          secretName: ambient-api-server
      - name: auth-config
        configMap:
          name: ambient-api-server-auth
---
apiVersion: v1
kind: Service
metadata:
  name: ambient-api-server
  labels:
    app: ambient-api-server
    component: api
spec:
  selector:
    app: ambient-api-server
    component: api
  ports:
  - name: api
    port: 8000
    targetPort: 8000
    protocol: TCP
  - name: grpc
    port: 9000
    targetPort: 9000
    protocol: TCP
  - name: metrics
    port: 4433
    targetPort: 4433
    protocol: TCP
  - name: health
    port: 4434
    targetPort: 4434
    protocol: TCP
EOF

echo "==> Step 5: Deploying control-plane"

$CLI apply -n "$NAMESPACE" -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: ambient-control-plane
  labels:
    app: ambient-control-plane
spec:
  selector:
    app: ambient-control-plane
  ports:
  - name: token
    port: 8080
    targetPort: 8080
    protocol: TCP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ambient-control-plane
  labels:
    app: ambient-control-plane
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ambient-control-plane
  template:
    metadata:
      labels:
        app: ambient-control-plane
    spec:
      serviceAccountName: ${SA_NAME}
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: ambient-control-plane
        image: ${REGISTRY}/vteam_control_plane:${IMAGE_TAG}
        imagePullPolicy: Always
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop: ["ALL"]
        env:
        - name: AMBIENT_API_TOKEN
          valueFrom:
            secretKeyRef:
              name: ambient-control-plane-token
              key: token
              optional: true
        - name: AMBIENT_API_SERVER_URL
          value: "http://ambient-api-server.${NAMESPACE}.svc:8000"
        - name: AMBIENT_GRPC_SERVER_ADDR
          value: "ambient-api-server.${NAMESPACE}.svc:9000"
        - name: AMBIENT_GRPC_USE_TLS
          value: "false"
        - name: MODE
          value: "kube"
        - name: PLATFORM_MODE
          value: "standard"
        - name: LOG_LEVEL
          value: "info"
        - name: RUNNER_IMAGE
          value: "${REGISTRY}/vteam_claude_runner:${IMAGE_TAG}"
        - name: MCP_IMAGE
          value: "${REGISTRY}/vteam_mcp:${IMAGE_TAG}"
        - name: MCP_API_SERVER_URL
          value: "http://ambient-api-server.${NAMESPACE}.svc:8000"
        - name: CP_TOKEN_URL
          value: "http://ambient-control-plane.${NAMESPACE}.svc:8080/token"
        - name: CP_RUNTIME_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: USE_VERTEX
          value: "${USE_VERTEX}"
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: ambient-anthropic
              key: api-key
              optional: true
        - name: ANTHROPIC_VERTEX_PROJECT_ID
          value: "${VERTEX_PROJECT_ID:-}"
        - name: CLOUD_ML_REGION
          value: "${VERTEX_REGION:-global}"
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: "/app/vertex/${VERTEX_KEY_FILE:-unused}"
        - name: VERTEX_SECRET_NAME
          value: "ambient-vertex"
        - name: VERTEX_SECRET_NAMESPACE
          value: "${NAMESPACE}"
        volumeMounts:
        - name: vertex-credentials
          mountPath: /app/vertex
          readOnly: true
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 256Mi
      volumes:
      - name: vertex-credentials
        secret:
          secretName: ambient-vertex
          optional: true
          defaultMode: 420
      restartPolicy: Always
EOF

echo "==> Step 6: Creating Route"

$CLI apply -n "$NAMESPACE" -f - <<EOF
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: ambient-api-server
  labels:
    app: ambient-api-server
    component: api
  annotations:
    haproxy.router.openshift.io/timeout: 10m
spec:
  to:
    kind: Service
    name: ambient-api-server
  port:
    targetPort: api
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect
EOF

echo "==> Step 7: Waiting for rollouts"
for deploy in ambient-api-server-db ambient-api-server ambient-control-plane; do
  echo "    Waiting for $deploy..."
  $CLI rollout status deployment/"$deploy" -n "$NAMESPACE" --timeout=300s
done

echo "==> Step 8: Verifying health"
API_HOST=$($CLI get route ambient-api-server -n "$NAMESPACE" \
  -o jsonpath='{.spec.host}' 2>/dev/null || true)

if [[ -z "$API_HOST" ]]; then
  echo "WARNING: Route not found — checking via port-forward"
  API_HOST="(use oc port-forward svc/ambient-api-server 8000:8000 -n $NAMESPACE)"
else
  HEALTH=$(curl -fsSk --connect-timeout 5 --max-time 20 \
    --retry 3 --retry-all-errors "https://${API_HOST}/api/ambient" 2>&1 || true)
  echo "    API server: ${HEALTH:-<no response>}"
fi

echo ""
echo "==> Ambient installed successfully in $NAMESPACE"
echo "    API server: https://${API_HOST}"
echo "    Image tag:  $IMAGE_TAG"
echo ""
echo "    Login:  acpctl login --url https://${API_HOST}"
echo ""
echo "    Teardown:"
echo "      bash components/pr-test/teardown-standard.sh $PR_NUMBER"
