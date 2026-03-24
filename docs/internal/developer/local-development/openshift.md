# OpenShift Cluster Development

This guide covers deploying the Ambient Code Platform on OpenShift clusters for development and testing. Use this when you need to test OpenShift-specific features like Routes, OAuth integration, or service mesh capabilities.

## Prerequisites

- `oc` CLI installed and logged in (`oc whoami` should succeed)
- `podman` or `docker` installed
- Access to an OpenShift cluster with cluster-admin (for OAuthClient and registry setup)

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

**Control plane API token** (the token the control plane uses to authenticate to the API server):

The API server validates tokens against **Red Hat SSO** (`sso.redhat.com/auth/realms/redhat-external`). `oc whoami -t` cluster tokens use a different signing key and will be rejected. You must use an RH SSO access token.

Get a fresh token via the `ocm` CLI:

```bash
ocm login  # if not already logged in
ocm token  # prints a valid RH SSO access token
```

Create the secret:

```bash
oc create secret generic ambient-control-plane-token -n ambient-code \
  --from-literal=token="$(ocm token)"
```

This secret is mounted by the `ambient-control-plane` deployment. RH SSO access tokens expire after ~15 minutes. To refresh:

```bash
oc delete secret ambient-control-plane-token -n ambient-code
oc create secret generic ambient-control-plane-token -n ambient-code \
  --from-literal=token="$(ocm token)"
oc rollout restart deployment/ambient-control-plane -n ambient-code
```

Verify the control plane connected successfully:

```bash
oc logs deployment/ambient-control-plane -n ambient-code --tail=10
# Expected: "project watch stream established", "session watch stream established"
```

## Building and Pushing Images

The production overlay uses `image-registry.openshift-image-registry.svc:5000/ambient-code/*` for images that must be built locally (control plane, runner). Other components pull from `quay.io/ambient_code/*`.

### Quick Deploy (recommended)

Builds the control plane and runner, pushes to the internal registry, and applies production manifests in one step:

```bash
make deploy-openshift
```

This target:
1. Verifies you are logged in (`oc whoami`)
2. Detects the internal registry hostname
3. Logs podman into the registry using your current `oc` token
4. Builds `ambient-control-plane` and `vteam_claude_runner` images
5. Pushes both to `image-registry.openshift-image-registry.svc:5000/ambient-code/`
6. Applies `components/manifests/overlays/production/` via `kubectl kustomize | kubectl apply`
7. Restarts the `ambient-control-plane` deployment and waits for rollout

### Manual Image Build and Push

If you need to build and push images individually:

```bash
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')

# Build
make build-control-plane
make build-runner

# Push control plane
podman tag ambient_control_plane:latest ${REGISTRY_HOST}/ambient-code/ambient_control_plane:latest
podman push --tls-verify=false ${REGISTRY_HOST}/ambient-code/ambient_control_plane:latest

# Push runner
podman tag vteam_claude_runner:latest ${REGISTRY_HOST}/ambient-code/vteam_claude_runner:latest
podman push --tls-verify=false ${REGISTRY_HOST}/ambient-code/vteam_claude_runner:latest
```

### Pushing All Images (full rebuild)

If you have built all components from source:

```bash
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')

make build-all

for img in vteam_frontend vteam_backend vteam_operator vteam_claude_runner vteam_api_server ambient_control_plane; do
  podman tag ${img}:latest ${REGISTRY_HOST}/ambient-code/${img}:latest
  podman push --tls-verify=false ${REGISTRY_HOST}/ambient-code/${img}:latest
done

# Restart deployments to pick up new images
oc rollout restart deployment ambient-control-plane backend-api frontend public-api agentic-operator -n ambient-code
```

## Platform Deployment

### Apply Production Manifests

The `production` kustomize overlay references `image-registry.openshift-image-registry.svc:5000/ambient-code/*` for control-plane and runner images (via `control-plane-image-patch.yaml`), and `quay.io/ambient_code/*` for everything else.

```bash
kubectl kustomize components/manifests/overlays/production/ | kubectl apply --validate=false -f -
```

Or use `make deploy-openshift` which handles the full build→push→apply flow.

**⚠️ Never commit `kustomization.yaml` while it contains local registry refs** — the production overlay should always use `image-registry.openshift-image-registry.svc:5000` for CP/runner (managed via patch files) and `quay.io/ambient_code` for everything else.

## Common Deployment Issues and Fixes

### Issue 1: Images not found (ImagePullBackOff)

```bash
REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')

# Rebuild and push missing images
make build-control-plane build-runner

podman tag ambient_control_plane:latest ${REGISTRY_HOST}/ambient-code/ambient_control_plane:latest
podman push --tls-verify=false ${REGISTRY_HOST}/ambient-code/ambient_control_plane:latest

podman tag vteam_claude_runner:latest ${REGISTRY_HOST}/ambient-code/vteam_claude_runner:latest
podman push --tls-verify=false ${REGISTRY_HOST}/ambient-code/vteam_claude_runner:latest

# Restart deployments
oc rollout restart deployment ambient-control-plane -n ambient-code
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

### Issue 3: Control plane token expired or rejected

The `ambient-control-plane-token` secret must hold a **Red Hat SSO** token (`sso.redhat.com/auth/realms/redhat-external`), not an `oc whoami -t` cluster token. Symptoms:

```
"unknown kid" — wrong token type (oc whoami -t was used)
"invalid_grant" — RH SSO token expired
```

Fix: get a fresh token via `ocm` and re-create the secret:

```bash
ocm login  # re-authenticate if needed
oc delete secret ambient-control-plane-token -n ambient-code
oc create secret generic ambient-control-plane-token -n ambient-code \
  --from-literal=token="$(ocm token)"
oc rollout restart deployment/ambient-control-plane -n ambient-code
```

Verify:

```bash
oc logs deployment/ambient-control-plane -n ambient-code --tail=10
# Expected: "project watch stream established", "session watch stream established"
```

### Issue 4: API server HTTPS configuration

```bash
# Check if HTTPS is properly configured in the deployment
oc get deployment ambient-api-server -n ambient-code -o yaml | grep -A5 -B5 enable-https

# Verify TLS certificate is mounted
oc describe deployment ambient-api-server -n ambient-code | grep -A10 -B5 tls
```

**Note:** The gRPC TLS for control plane communication provides end-to-end encryption for session monitoring.

## Cross-Namespace Image Access

The operator creates runner pods in dynamically-created project namespaces (e.g. `fleet-NNNNN`). Those pods need to pull images from the `ambient-code` namespace. Grant all service accounts pull access:

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

### Verify Control Plane gRPC Connectivity

```bash
# Check control plane is connecting via TLS gRPC
oc logs deployment/ambient-control-plane -n ambient-code --tail=20 | grep -i "grpc\|session\|connect"

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
- **Frontend**: `https://ambient-code.apps.<cluster-domain>/`
- **Backend API**: `https://backend-route-ambient-code.apps.<cluster-domain>/`
- **Public API**: `https://public-api-route-ambient-code.apps.<cluster-domain>/`
- **Ambient API Server**: `https://ambient-api-server-ambient-code.apps.<cluster-domain>/`
- **Ambient API Server gRPC**: `https://ambient-api-server-grpc-ambient-code.apps.<cluster-domain>/`

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

The control plane authenticates to the API server using a bearer token stored in the `ambient-control-plane-token` secret. The API server validates tokens via Red Hat SSO JWKS — `oc whoami -t` cluster tokens are **not accepted**.

Use `ocm token` to get a valid RH SSO access token:

```bash
ocm login
oc delete secret ambient-control-plane-token -n ambient-code
oc create secret generic ambient-control-plane-token -n ambient-code \
  --from-literal=token="$(ocm token)"
oc rollout restart deployment/ambient-control-plane -n ambient-code
```

Note: `ocm token` tokens expire in ~15 minutes. For a longer-lived token, use a service account registered in RH SSO.

### Vertex AI Integration (Optional)

The `deploy.sh` script reads `ANTHROPIC_VERTEX_PROJECT_ID` from your environment and sets `CLAUDE_CODE_USE_VERTEX=1` in the operator configmap. The operator then **requires** the `ambient-vertex` secret to exist in `ambient-code`.

**Create this secret before running `make deploy-openshift` if using Vertex AI:**

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
- ✅ **Deploys ambient-control-plane** from locally-built image in internal registry
- ✅ **Deploys runner** from locally-built image in internal registry
- ✅ **Configures OAuth** integration automatically (with cluster-admin)
- ✅ **Creates all routes** for external access
- ✅ **Database migrations** run automatically with proper permissions

## Troubleshooting

### Missing public-api-route

```bash
# Check if public-api is deployed
oc get route public-api-route -n $AMBIENT_PROJECT

# If missing, re-apply production manifests:
kubectl kustomize components/manifests/overlays/production/ | kubectl apply --validate=false -f -
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

### Control plane not connecting to gRPC

```bash
# Check control plane logs for gRPC errors
oc logs deployment/ambient-control-plane -n ambient-code --tail=50 | grep -i "grpc\|error\|connect"

# Verify the gRPC route exists
oc get route ambient-api-server-grpc -n ambient-code

# Check control plane environment
oc get deployment ambient-control-plane -n ambient-code -o jsonpath='{.spec.template.spec.containers[0].env}' | python3 -m json.tool
```

## Next Steps

1. Access the frontend URL (from `oc get route -n ambient-code`)
2. Configure ANTHROPIC_API_KEY in project settings
3. Test SDKs using the commands above
4. Create your first AgenticSession via UI or SDK
5. Monitor with: `oc get pods -n ambient-code -w`
6. Monitor control plane session reconciliation: `oc logs -f deployment/ambient-control-plane -n ambient-code`
