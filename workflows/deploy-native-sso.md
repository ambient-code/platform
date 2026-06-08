# Deploy Ambient Code with Native SSO Support

Complete workflow for deploying Ambient Code to a new namespace with native Keycloak SSO, bypassing the legacy oauth-proxy and operator.

## Architecture

```
User → Keycloak (SSO) → Frontend (NextJS)
                      → acpctl (CLI)
                      ↓
               API Server (REST + gRPC, PostgreSQL-backed)
                      ↓
               Control Plane (watches gRPC streams, creates k8s resources)
                      ↓
               Runner Pod (Claude Agent SDK, authenticates via CP token server)
```

No legacy backend proxy, no agentic-operator, no oauth-proxy.

## Prerequisites

- `oc` CLI authenticated to the target cluster
- `kustomize` installed
- Access to push container images to your registry
- Access to Keycloak admin UI (or ability to create one)
- GCP service account JSON key with Vertex AI permissions

## Variables

Replace these throughout the workflow:

| Variable | Example | Description |
|----------|---------|-------------|
| `$NAMESPACE` | `jsell-ambient-sso-poc` | Target namespace |
| `$CLUSTER_APPS_DOMAIN` | `apps.rosa.hcmais01ue1.s9m2.p3.openshiftapps.com` | Cluster wildcard domain |
| `$KEYCLOAK_URL` | `https://keycloak-ambient-keycloak.$CLUSTER_APPS_DOMAIN` | Keycloak base URL |
| `$KEYCLOAK_REALM` | `ambient-code` | Keycloak realm name |
| `$REGISTRY` | `quay.io/rh-ee-jsell` | Container image registry |
| `$GCP_PROJECT_ID` | `my-gcp-project` | GCP Vertex AI project |
| `$GCP_KEY_FILENAME` | `service-account.json` | Key filename inside ambient-vertex secret |

---

## Step 1: Keycloak Setup

### 1a. Deploy Keycloak (if not already running)

If you need your own Keycloak instance, deploy one to the cluster. Otherwise skip to 1b.

### 1b. Create Realm

1. Log into Keycloak admin UI at `$KEYCLOAK_URL/admin`
2. Create realm: `$KEYCLOAK_REALM`

### 1c. Configure Identity Brokering with RH SSO

1. In the `$KEYCLOAK_REALM` realm → **Identity Providers** → **Add provider** → **OpenID Connect v1.0**
2. Configure:
   - **Alias**: `redhat-sso`
   - **Display Name**: `Red Hat SSO`
   - **Discovery URL**: `https://sso.redhat.com/auth/realms/redhat-external/.well-known/openid-configuration`
   - **Client ID**: your RH SSO client ID
   - **Client Secret**: your RH SSO client secret
   - **Scopes**: leave empty unless the upstream client supports `openid email profile` (most don't — removing them avoids `invalid_scope` errors)
3. Under **Advanced** or **First login flow**: set to a flow that does NOT include "Review Profile" — otherwise users can self-declare their username on first login

### 1d. Auto-redirect to RH SSO

1. **Authentication** → **browser** flow
2. Find **Identity Provider Redirector** execution step
3. Click gear/config → set **Default Identity Provider** = `redhat-sso`

This skips the Keycloak login form and sends users straight to RH SSO.

### 1e. Create Client: `ambient-frontend`

1. **Clients** → **Create client**
2. **Client ID**: `ambient-frontend`
3. **Client type**: OpenID Connect
4. **Client authentication**: ON
5. **Authentication flow**: check **Standard flow** only
6. **Valid redirect URIs**: `https://frontend-route-$NAMESPACE.$CLUSTER_APPS_DOMAIN/api/auth/sso/callback`
7. **Web origins**: `https://frontend-route-$NAMESPACE.$CLUSTER_APPS_DOMAIN`
8. Save → go to **Credentials** tab → copy the **Client secret**

### 1f. Create Client: `ambient-control-plane` (service account)

1. **Clients** → **Create client**
2. **Client ID**: `ambient-control-plane` (or any name — note it as `$CP_CLIENT_ID`)
3. **Client type**: OpenID Connect
4. **Client authentication**: ON
5. **Authentication flow**: check **Service accounts roles** ONLY (uncheck everything else)
6. Leave Root URL and Home URL blank
7. Save → go to **Credentials** tab → copy the **Client secret**

### 1g. Create Client: `ambient-cli` (for acpctl)

1. **Clients** → **Create client**
2. **Client ID**: `ambient-cli`
3. **Client type**: OpenID Connect
4. **Client authentication**: OFF (public client)
5. **Authentication flow**: check **Standard flow** only
6. **Valid redirect URIs**: `http://127.0.0.1/callback`

   Note: `acpctl` uses `http://127.0.0.1:<random-port>/callback`. Keycloak treats `localhost` specially and allows any port. Using the IP `127.0.0.1` without a port works the same way.

---

## Step 2: Build and Push Images

Build the api-server with the `JWK_CERT_URL` env var support:

```bash
cd components/ambient-api-server
podman build -t $REGISTRY/vteam_api_server:latest -f Dockerfile .
podman push $REGISTRY/vteam_api_server:latest
```

Build any other modified components (frontend, backend, etc.) similarly.

### Required code change: `e_production.go`

The api-server's production environment hardcodes the JWKS URL. Ensure `components/ambient-api-server/cmd/ambient-api-server/environments/e_production.go` reads `JWK_CERT_URL` from the environment:

```go
switch {
case os.Getenv("JWK_CERT_URL") != "":
    c.Auth.JwkCertURL = os.Getenv("JWK_CERT_URL")
case c.Auth.JwkCertURL != "" && c.Auth.JwkCertURL != defaultJwkCertURL:
    // CLI flag was explicitly set to a non-default value; keep it.
default:
    c.Auth.JwkCertURL = defaultJwkCertURL
}
```

Without this, the api-server ignores `--jwk-cert-url` and `JWK_CERT_URL`.

---

## Step 3: Create Kubernetes Secrets

These secrets are NOT managed by the overlay and must be created manually before applying.

### 3a. SSO Credentials

Update `sso-credentials.yaml` in the overlay with real values, or create directly:

```bash
oc create secret generic sso-credentials -n $NAMESPACE \
  --from-literal=SSO_ISSUER_URL=$KEYCLOAK_URL/realms/$KEYCLOAK_REALM \
  --from-literal=SSO_FRONTEND_ISSUER_URL=$KEYCLOAK_URL/realms/$KEYCLOAK_REALM \
  --from-literal=SSO_CLIENT_ID=ambient-frontend \
  --from-literal=SSO_CLIENT_SECRET=<secret from step 1e> \
  --from-literal=SSO_AUDIENCE=ambient-frontend \
  --from-literal=SESSION_SECRET=$(openssl rand -hex 32)
```

### 3b. Control Plane OIDC

```bash
oc create secret generic control-plane-oidc -n $NAMESPACE \
  --from-literal=client-id=$CP_CLIENT_ID \
  --from-literal=client-secret=<secret from step 1f>
```

### 3c. Vertex AI Credentials

```bash
oc create secret generic ambient-vertex -n $NAMESPACE \
  --from-file=$GCP_KEY_FILENAME=/path/to/gcp-service-account.json
```

---

## Step 4: Configure the Overlay

### 4a. `kustomization.yaml`

Update:
- `namespace:` → `$NAMESPACE`
- `images:` → your registry (`$REGISTRY`)

### 4b. `sso-credentials.yaml`

If using the file-based secret (instead of step 3a), update all `CHANGE_ME` values.

### 4c. `frontend-sso-patch.yaml`

Update `SSO_REDIRECT_URI`:
```yaml
- name: SSO_REDIRECT_URI
  value: "https://frontend-route-$NAMESPACE.$CLUSTER_APPS_DOMAIN/api/auth/sso/callback"
```

### 4d. `ambient-api-server-env-patch.yaml`

Update:
```yaml
- name: BACKEND_URL
  value: "http://backend-service.$NAMESPACE.svc:8080"
- name: JWK_CERT_URL
  value: "$KEYCLOAK_URL/realms/$KEYCLOAK_REALM/protocol/openid-connect/certs"
```

### 4e. `control-plane-env-patch.yaml`

Update all service URLs to use `$NAMESPACE`:
```yaml
- name: AMBIENT_API_SERVER_URL
  value: "http://ambient-api-server.$NAMESPACE.svc:8000"
- name: AMBIENT_GRPC_SERVER_ADDR
  value: "ambient-api-server.$NAMESPACE.svc:9000"
- name: AMBIENT_GRPC_USE_TLS
  value: "false"
- name: CP_TOKEN_URL
  value: "http://ambient-control-plane.$NAMESPACE.svc:8080/token"
- name: OIDC_TOKEN_URL
  value: "$KEYCLOAK_URL/realms/$KEYCLOAK_REALM/protocol/openid-connect/token"
```

### 4f. `api-server-secret-patch.yaml`

Set `clientId` to match the Keycloak control plane client ID (`$CP_CLIENT_ID`). This tells the api-server's gRPC middleware to recognize tokens from this client as service callers:
```yaml
stringData:
  clientId: $CP_CLIENT_ID
```

### 4g. `clusterrolebindings.yaml`

**Applied separately from kustomize** (kustomize's `namespace:` transformer rewrites all subject namespaces, which would break the `ambient-code` bindings).

Update all subject namespaces to include both `ambient-code` and `$NAMESPACE`. If deploying to `ambient-code` itself, only one subject per binding is needed.

```yaml
subjects:
- kind: ServiceAccount
  name: ambient-control-plane
  namespace: ambient-code
- kind: ServiceAccount
  name: ambient-control-plane
  namespace: $NAMESPACE
```

### 4h. `api-server-route.yaml`

Update the hostname (or remove `host:` to let OpenShift auto-generate):
```yaml
spec:
  host: ambient-api-server-$NAMESPACE.$CLUSTER_APPS_DOMAIN
```

### 4i. `operator-config-openshift.yaml`

Update GCP config:
```yaml
data:
  USE_VERTEX: "1"
  CLOUD_ML_REGION: "global"
  ANTHROPIC_VERTEX_PROJECT_ID: "$GCP_PROJECT_ID"
  GOOGLE_APPLICATION_CREDENTIALS: "/app/vertex/$GCP_KEY_FILENAME"
```

---

## Step 5: Deploy

```bash
cd /path/to/platform

# Apply kustomize first
oc apply -k components/manifests/overlays/hcmais/jsell-sso-poc/

# Then apply CRBs AFTER kustomize to restore the ambient-code subjects
# that kustomize's namespace transformer overwrites
oc apply -f components/manifests/overlays/hcmais/jsell-sso-poc/clusterrolebindings.yaml
```

Verify all deployments come up:
```bash
oc get pods -n $NAMESPACE
```

Expected pods:
- `ambient-api-server` (1/1 Running)
- `ambient-api-server-db` (1/1 Running)
- `ambient-control-plane` (1/1 Running)
- `backend-api` (1/1 Running)
- `frontend` (1/1 Running)
- `minio` (1/1 Running)
- `postgresql` (1/1 Running)
- `public-api` (1/1 Running — legacy, can be removed later)
- `unleash` (1/1 Running)
- `agentic-operator` (1/1 Running — legacy, can be removed later)

---

## Step 6: Verify

### 6a. Control Plane → API Server gRPC

```bash
oc logs deployment/ambient-control-plane -n $NAMESPACE --tail=10
```

Look for:
```
session watch stream established
project watch stream established
project_settings watch stream established
```

If you see `Unauthenticated` or `unknown kid`, the JWKS configuration is wrong (step 4d) or the control plane OIDC client isn't recognized (step 4f).

### 6b. API Server JWKS

```bash
oc logs deployment/ambient-api-server -n $NAMESPACE -c api-server --tail=20 | grep -i 'jwt\|key\|URL'
```

Should show loading keys from your Keycloak URL.

### 6c. Frontend SSO Login

Visit `https://frontend-route-$NAMESPACE.$CLUSTER_APPS_DOMAIN` — should redirect to Keycloak → RH SSO → back to the app.

### 6d. CLI Login

```bash
acpctl login --use-auth-code \
  --url https://ambient-api-server-$NAMESPACE.$CLUSTER_APPS_DOMAIN \
  --issuer-url $KEYCLOAK_URL/realms/$KEYCLOAK_REALM \
  --client-id ambient-cli
```

### 6e. End-to-End Session

```bash
acpctl project create test-project
acpctl session create --project test-project --prompt "say hi"
```

Watch control plane logs — should show session reconciliation and runner pod creation.

---

## Promoting to `ambient-code` Namespace

When moving from a PoC namespace to the production `ambient-code` namespace, change:

| What | From | To |
|------|------|----|
| `kustomization.yaml` namespace | `jsell-ambient-sso-poc` | `ambient-code` |
| All `*.svc` URLs in patches | `*.jsell-ambient-sso-poc.svc` | `*.ambient-code.svc` |
| `control-plane-rbac-patch.yaml` subject namespace | `jsell-ambient-sso-poc` | `ambient-code` |
| Route hostnames | `*-jsell-ambient-sso-poc.$CLUSTER_APPS_DOMAIN` | `*-ambient-code.$CLUSTER_APPS_DOMAIN` (or custom domain) |
| `SSO_REDIRECT_URI` in frontend patch | PoC frontend route | Production frontend route |
| Keycloak `ambient-frontend` client redirect URIs | PoC callback URL | Production callback URL |
| Image registry in kustomization | `quay.io/rh-ee-jsell/*` | Production registry |
| `ambient-vertex` secret | Copy from existing namespace | Already exists in `ambient-code` |
| `sso-credentials` secret | PoC values | Production Keycloak values |
| `control-plane-oidc` secret | PoC client credentials | Production client credentials (or same Keycloak) |

### Conflict with existing deployment

The `ambient-code` namespace already has the legacy stack (oauth-proxy frontend, agentic-operator, legacy backend). To promote:

1. **Scale down the legacy operator** to prevent it from fighting with the control plane over session CRs:
   ```bash
   oc scale deployment/agentic-operator -n ambient-code --replicas=0
   ```

2. **Update the frontend deployment** to remove the oauth-proxy sidecar container (the SSO patch replaces it)

3. **The CRDs are shared** (`AgenticSession`, `ProjectSettings`) — no changes needed, they're cluster-scoped definitions with namespaced instances

4. **ClusterRoleBindings** for `ambient-control-plane` and `backend-api` already reference `ambient-code` namespace in the base manifests — the overlay patches won't conflict

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `unknown key ID` in api-server logs | API server can't verify JWT from Keycloak or control plane | Check `JWK_CERT_URL` points to correct Keycloak JWKS endpoint. For service-to-service tokens from a different issuer, add their keys to the `jwks.json` configmap as a static fallback |
| `invalid_client` in control plane logs | Wrong OIDC client ID or secret | Verify `control-plane-oidc` secret matches Keycloak client credentials |
| `invalid_scope` on Keycloak login | Upstream RH SSO client doesn't support requested scopes | Remove scopes from Identity Provider config in Keycloak |
| `invalid_redirect_uri` for acpctl | Keycloak client missing localhost redirect | Add `http://127.0.0.1/callback` to `ambient-cli` client Valid Redirect URIs |
| `PERMISSION_DENIED` on `WatchSessionMessages` | Runner token not recognized as service caller | Verify `GRPC_SERVICE_ACCOUNT` (from `ambient-api-server` secret `clientId`) matches the Keycloak control plane client ID. The api-server checks for `service-account-<clientId>` prefix |
| `Service account key file not found` | Vertex AI credentials not mounted or wrong filename | Verify `ambient-vertex` secret key name matches `GOOGLE_APPLICATION_CREDENTIALS` path in `operator-config` |
| Control plane gRPC `i/o timeout` | Control plane pointing at wrong api-server | Check `AMBIENT_API_SERVER_URL` and `AMBIENT_GRPC_SERVER_ADDR` in control plane env |
| User sees "Review Profile" on first SSO login | Keycloak first broker login flow prompts for profile | Set first login flow to skip Review Profile step |
| 403 on project access after creation | User identity in SAR doesn't match RoleBinding subject | Ensure the claim used for RoleBinding (`email`) matches what the backend uses for SubjectAccessReview impersonation |
