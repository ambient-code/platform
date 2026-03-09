# OpenShift Cluster Development

This guide covers deploying the Ambient Code Platform on OpenShift clusters for development and testing. Use this when you need to test OpenShift-specific features like Routes, OAuth integration, or service mesh capabilities.

## Prerequisites

- `oc` CLI installed
- `podman` or `docker` installed
- Access to an OpenShift cluster

## OpenShift Cluster Setup

### Option 1: OpenShift Local (CRC)
For local development, see [crc.md](crc.md) for detailed CRC setup instructions.

### Option 2: Cloud OpenShift Cluster
For cloud clusters (ROSA, OCP on AWS/Azure/GCP), ensure you have cluster-admin access.

### Option 3: Temporary Test Cluster
For temporary testing clusters, you can use cluster provisioning tools available in your organization.

## Registry Configuration

### Enable OpenShift Internal Registry

Expose the internal image registry:

```bash
oc patch configs.imageregistry.operator.openshift.io/cluster --type merge --patch '{"spec":{"defaultRoute":true}}'
```

Get the registry hostname:

```bash
oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}'
```

### Login to Registry

Authenticate podman to the OpenShift registry:

```bash
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
oc whoami -t | podman login --tls-verify=false -u kubeadmin --password-stdin "$REGISTRY_HOST"
```

## Required Secrets Setup

**IMPORTANT**: Create all required secrets **before** deploying. The deployment will fail if these secrets are missing.

Create the project namespace:
```bash
oc new-project ambient-code
```

**MinIO credentials:**

```bash
oc create secret generic minio-credentials -n ambient-code \
  --from-literal=root-user=admin \
  --from-literal=root-password=changeme123
```

**PostgreSQL credentials (for Unleash feature flag database):**

```bash
oc create secret generic postgresql-credentials -n ambient-code \
  --from-literal=db.host="postgresql" \
  --from-literal=db.port="5432" \
  --from-literal=db.name="postgres" \
  --from-literal=db.user="postgres" \
  --from-literal=db.password="postgres123"
```

**Unleash credentials (for feature flag service):**

```bash
oc create secret generic unleash-credentials -n ambient-code \
  --from-literal=database-url="postgres://postgres:postgres123@postgresql:5432/unleash" \
  --from-literal=database-ssl="false" \
  --from-literal=admin-api-token="*:*.unleash-admin-token" \
  --from-literal=client-api-token="default:development.unleash-client-token" \
  --from-literal=frontend-api-token="default:development.unleash-frontend-token" \
  --from-literal=default-admin-password="unleash123"
```

## Platform Deployment

The production kustomization in `components/manifests/overlays/production/kustomization.yaml` references `quay.io/ambient_code/*` images by default. When deploying to an OpenShift cluster using the internal registry, you must temporarily point the image refs at the internal registry, deploy, then **immediately revert** before committing.

**⚠️ CRITICAL**: Never commit `kustomization.yaml` while it contains internal registry refs.

**Patch kustomization to internal registry, deploy, then revert:**

```bash
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
INTERNAL_REG="image-registry.openshift-image-registry.svc:5000/ambient-code"

# Temporarily override image refs to internal registry
cd components/manifests/overlays/production
sed -i "s#newName: quay.io/ambient_code/#newName: ${INTERNAL_REG}/#g" kustomization.yaml

# Deploy
cd ../..
./deploy.sh

# IMMEDIATELY revert — do not commit with internal registry refs
cd overlays/production
git checkout kustomization.yaml
```

## Common Deployment Issues and Fixes

### Issue 1: Images not found (ImagePullBackOff)

```bash
# Build and push required images to internal registry
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')

# Tag and push key images (adjust based on what's available locally)
podman tag localhost/ambient_control_plane:latest ${REGISTRY_HOST}/ambient-code/ambient_control_plane:latest
podman tag localhost/vteam_frontend:latest ${REGISTRY_HOST}/ambient-code/vteam_frontend:latest
podman tag localhost/vteam_api_server:latest ${REGISTRY_HOST}/ambient-code/vteam_api_server:latest
podman tag localhost/vteam_backend:latest ${REGISTRY_HOST}/ambient-code/vteam_backend:latest
podman tag localhost/vteam_operator:latest ${REGISTRY_HOST}/ambient-code/vteam_operator:latest
podman tag localhost/vteam_public_api:latest ${REGISTRY_HOST}/ambient-code/vteam_public_api:latest
podman tag localhost/vteam_claude_runner:latest ${REGISTRY_HOST}/ambient-code/vteam_claude_runner:latest

# Push images
for img in ambient_control_plane vteam_frontend vteam_api_server vteam_backend vteam_operator vteam_public_api vteam_claude_runner; do
  podman push ${REGISTRY_HOST}/ambient-code/${img}:latest
done

# Restart deployments to pick up new images
oc rollout restart deployment ambient-control-plane backend-api frontend public-api agentic-operator -n ambient-code
```

### Issue 2: API server TLS certificate missing

```bash
# Add service annotation to generate TLS certificate
oc annotate service ambient-api-server service.beta.openshift.io/serving-cert-secret-name=ambient-api-server-tls -n ambient-code

# Wait for certificate generation
sleep 10

# Restart API server to mount certificate
oc rollout restart deployment ambient-api-server -n ambient-code
```

### Issue 3: API server HTTPS configuration

The ambient-api-server includes TLS support for production deployments. For development clusters, you may need to adjust the configuration:

```bash
# Check if HTTPS is properly configured in the deployment
oc get deployment ambient-api-server -n ambient-code -o yaml | grep -A5 -B5 enable-https

# Verify TLS certificate is mounted
oc describe deployment ambient-api-server -n ambient-code | grep -A10 -B5 tls
```

**Note:** The gRPC TLS for control plane communication provides end-to-end encryption for session monitoring.

## Cross-Namespace Image Access

The operator creates runner pods in dynamically-created project namespaces (e.g. `hyperfleet-test`). Those pods need to pull images from the `ambient-code` namespace. Grant all service accounts pull access:

```bash
oc policy add-role-to-group system:image-puller system:serviceaccounts --namespace=ambient-code
```

Without this, runner pods will fail with `ErrImagePull` / `authentication required`.

## Deployment Verification

### Check Pod Status

```bash
oc get pods -n ambient-code
```

**Expected output:** All pods should show `1/1 Running` or `2/2 Running` (frontend has oauth-proxy):
```
NAME                                     READY   STATUS    RESTARTS   AGE
agentic-operator-xxxxx-xxxxx             1/1     Running   0          5m
ambient-api-server-xxxxx-xxxxx           1/1     Running   0          5m
ambient-api-server-db-xxxxx-xxxxx        1/1     Running   0          5m
ambient-control-plane-xxxxx-xxxxx        1/1     Running   0          5m
backend-api-xxxxx-xxxxx                  1/1     Running   0          5m
frontend-xxxxx-xxxxx                     2/2     Running   0          5m
minio-xxxxx-xxxxx                        1/1     Running   0          5m
postgresql-xxxxx-xxxxx                   1/1     Running   0          5m
public-api-xxxxx-xxxxx                   1/1     Running   0          5m
unleash-xxxxx-xxxxx                      1/1     Running   0          5m
```

### Test Database Connection

```bash
oc exec deployment/ambient-api-server-db -n ambient-code -- psql -U ambient -d ambient_api_server -c "\dt"
```

**Expected:** Should show 6 database tables (events, migrations, project_settings, projects, sessions, users).

### Verify Control Plane TLS Functionality

```bash
# Check control plane is connecting via TLS gRPC
oc logs deployment/ambient-control-plane -n ambient-code --tail=10 | grep -i grpc

# Verify API server gRPC streams are active  
oc logs deployment/ambient-api-server -n ambient-code --tail=20 | grep "gRPC stream started"
```

**Expected:** You should see successful gRPC stream connections like:
```
gRPC stream started /ambient.v1.ProjectService/WatchProjects
gRPC stream started /ambient.v1.SessionService/WatchSessions
```

## Platform Access

### Get Platform URLs

```bash
oc get route -n ambient-code
```

**Main routes:**
- **Frontend**: https://ambient-code.apps.<cluster-domain>/
- **Backend API**: https://backend-route-ambient-code.apps.<cluster-domain>/  
- **Public API**: https://public-api-route-ambient-code.apps.<cluster-domain>/
- **Ambient API Server**: https://ambient-api-server-ambient-code.apps.<cluster-domain>/

### Health Check

```bash
curl -k https://backend-route-ambient-code.apps.<cluster-domain>/health
# Expected: {"status":"healthy"}
```

## SDK Testing

### Setup Environment Variables

Set the SDK environment variables based on your current `oc` client configuration:

```bash
# Auto-configure from current oc context
export AMBIENT_TOKEN="$(oc whoami -t)"                    # Use current user token
export AMBIENT_PROJECT="$(oc project -q)"                 # Use current project/namespace
export AMBIENT_API_URL="$(oc get route public-api-route --template='https://{{.spec.host}}')"  # Get public API route
```

**Verify configuration:**
```bash
echo "Token: ${AMBIENT_TOKEN:0:12}... (${#AMBIENT_TOKEN} chars)"
echo "Project: $AMBIENT_PROJECT"
echo "API URL: $AMBIENT_API_URL"
```

### Test Go SDK

```bash
cd components/ambient-sdk/go-sdk
go run main.go
```

### Test Python SDK

```bash
cd components/ambient-sdk/python-sdk
./test.sh
```

Both SDKs should output successful session creation and listing.

## CLI Testing

Login to the ambient-control-plane using the CLI:

```bash
acpctl login --url https://ambient-api-server-ambient-code.apps.<cluster-domain> --token $(oc whoami -t)
```

## Authentication Configuration

### API Token Setup

The control plane authenticates to the API server using a bearer token. By default `deploy.sh` uses `oc whoami -t` (your current cluster token). To use a dedicated long-lived token instead, set it before deploying:

```bash
export AMBIENT_API_TOKEN=<your-token>
```

If `AMBIENT_API_TOKEN` is not set, the deploy script automatically creates the secret using your current `oc` session token.

### Vertex AI Integration (Optional)

The `deploy.sh` script reads `ANTHROPIC_VERTEX_PROJECT_ID` from your environment and sets `CLAUDE_CODE_USE_VERTEX=1` in the operator configmap. The operator then **requires** the `ambient-vertex` secret to exist in `ambient-code`.

**Create this secret before running `make deploy` if using Vertex AI:**

First, ensure you have Application Default Credentials:

```bash
gcloud auth application-default login
```

Then create the secret:

```bash
oc create secret generic ambient-vertex -n ambient-code \
  --from-file=ambient-code-key.json="$HOME/.config/gcloud/application_default_credentials.json"
```

Alternatively, if you have a service account key file:

```bash
oc create secret generic ambient-vertex -n ambient-code \
  --from-file=ambient-code-key.json="/path/to/your-service-account-key.json"
```

**Note:** If you do NOT want to use Vertex AI and prefer direct Anthropic API, unset the env var before deploying:

```bash
unset ANTHROPIC_VERTEX_PROJECT_ID
```

## OAuth Configuration

OAuth configuration requires cluster-admin permissions for creating the OAuthClient resource. If you don't have cluster-admin, the deployment will warn you but other components will still deploy.

## What the Deployment Provides

- ✅ **Applies all CRDs** (Custom Resource Definitions)
- ✅ **Creates RBAC** roles and service accounts  
- ✅ **Deploys all components** with correct OpenShift-compatible security contexts
- ✅ **Configures OAuth** integration automatically (with cluster-admin)
- ✅ **Creates all routes** for external access
- ✅ **Database migrations** run automatically with proper permissions

## Troubleshooting

### Missing public-api-route

```bash
# Check if public-api is deployed
oc get route public-api-route -n $AMBIENT_PROJECT

# If missing, deploy public-api component:
cd components/manifests
./deploy.sh
```

### Authentication errors

```bash
# Verify token is valid
oc whoami

# Check project access
oc get pods -n $AMBIENT_PROJECT
```

### API connection errors

```bash
# Test API directly
curl -H "Authorization: Bearer $(oc whoami -t)" \
     -H "X-Ambient-Project: $(oc project -q)" \
     "$AMBIENT_API_URL/health"
```

## Next Steps

1. Access the frontend URL (from `oc get route -n ambient-code`)
2. Configure ANTHROPIC_API_KEY in project settings
3. Test SDKs using the commands above
4. Create your first AgenticSession via UI or SDK
5. Monitor with: `oc get pods -n ambient-code -w`