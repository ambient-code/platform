# SSO Authentication Migration Workflow

**Spec:** `specs/security/sso-authentication.spec.md`

## Consumer Migration Map

Every component that touches user authentication and what changes for each.

| Consumer | Current behavior | New behavior | Key files |
|----------|-----------------|--------------|-----------|
| Frontend OAuth proxy sidecar | Injects `X-Forwarded-Access-Token`, `X-Forwarded-User`, etc. | Removed; Next.js handles OIDC directly | `manifests/components/oauth-proxy/` |
| Frontend `buildForwardHeadersAsync` | Reads `X-Forwarded-Access-Token` from request, forwards to upstream | Reads JWT from OIDC session, sets `Authorization: Bearer` | `src/lib/auth.ts` |
| Frontend logout | Redirects to `/oauth/sign_out` (OAuth proxy endpoint) | Redirects to Next.js signout → SSO logout | `src/components/navigation.tsx`, `src/app/projects/[name]/layout.tsx` |
| Backend `forwardedIdentityMiddleware` | Reads `X-Forwarded-User/Email/Groups` headers | Reads identity from validated JWT claims | `server/server.go` |
| Backend `GetK8sClientsForRequest` | Uses raw token as `cfg.BearerToken` | Validates JWT, uses SA token + impersonation | `handlers/middleware.go`, `handlers/k8s_clients_for_request_prod.go` |
| Backend SSAR cache | Keyed by `SHA256(token)` | Keyed by `SHA256(token) + impersonated-user` | `handlers/ssar_cache.go` |
| Backend API key auth | TokenReview on SA token | Unchanged — TokenReview is the fallback when JWT parsing fails | `handlers/middleware.go` |
| API server `forwarded_token.go` | Converts `X-Forwarded-Access-Token` to `Authorization` header | Passthrough — JWT arrives in `Authorization` already | `pkg/middleware/forwarded_token.go` |
| Public API `extractToken` | Falls back to `X-Forwarded-Access-Token` | `Authorization: Bearer` only | `handlers/middleware.go` |
| CLI `acpctl login` | OIDC auth code + PKCE against SSO, client ID `ocm-cli` | Same flow, dedicated client ID | `cmd/acpctl/login/cmd.go` |
| SDK (Go, Python) | Accepts token string, sets `Authorization: Bearer` | No change — token format is opaque to SDK | None |
| Runner `caller_token` | Receives opaque token or JWT via `x-caller-token` | Receives JWT via `x-caller-token` | No change — runner treats it as opaque bearer |
| Runner K8s access | Per-session SA bot token | Per-session SA bot token (unchanged) | None |
| E2E tests | Inject SA token via `NEXT_PUBLIC_E2E_TOKEN` (browser-exposed) | Inject test JWT server-side (cookie or API route) | `e2e/cypress/support/commands.ts`, `src/services/api/client.ts` |
| Per-user RoleBindings | `subjects[].name = "user@email.com"` | Same — impersonation uses same email string | None |

## RBAC Changes

### Backend ServiceAccount — new impersonation ClusterRole

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: backend-api-impersonator
rules:
  - apiGroups: [""]
    resources: ["users", "groups", "serviceaccounts"]
    verbs: ["impersonate"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: backend-api-impersonator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: backend-api-impersonator
subjects:
  - kind: ServiceAccount
    name: backend-api
    namespace: ambient-code
```

## Backend Implementation Notes

### Dual-path auth flow

```
Token received
  │
  ├─ Try JWT validation (JWKS)
  │   ├─ Success → extract claims → impersonate user
  │   └─ Fail (not a JWT) ─┐
  │                         │
  └─────────────────────────┤
                            │
                     Try K8s TokenReview
                       ├─ Success → resolve SA identity → impersonate
                       └─ Fail → 401 Unauthorized
```

### SSAR cache key change

Current: `SHA256(token)[:8]:namespace:verb:group:resource`

With impersonation, `token` is always the backend SA token (same for all requests).
New key must include impersonated identity:

`SHA256(token)[:8]:impersonated-user:namespace:verb:group:resource`

### GetK8sClientsForRequest — impersonation config

The function signature stays the same: `(c *gin.Context) → (kubernetes.Interface, dynamic.Interface)`.
Internally, instead of `cfg.BearerToken = userToken`, use:

```go
cfg.BearerToken = backendSAToken
cfg.Impersonate = rest.ImpersonationConfig{
    UserName: emailFromJWT,
    Groups:   groupsFromJWT,
}
```

All 142+ callers are unaffected — they receive a K8s client and don't know how it was built.

### Dual-client pattern preserved

Some handlers use both user-scoped client (RBAC check) and backend SA client (writes):
- User-scoped: SA token + impersonation (RBAC checked by K8s as impersonated user)
- Backend SA: SA token without impersonation (elevated for writes after RBAC validation)

The nil-check on `GetK8sClientsForRequest` changes semantics: the SA client never
returns nil (unlike user token clients that return nil on invalid tokens). JWT validation
failures should return 401 before reaching the client construction.

## Frontend Implementation Notes

### OIDC session layer

The frontend needs an OIDC client library that supports:
- Authorization Code Flow with confidential client
- Server-side session storage
- Token refresh
- JWKS validation
- Single sign-out

`buildForwardHeadersAsync` changes from reading `X-Forwarded-Access-Token` to
extracting the JWT from the OIDC session. The function signature and all 97+ consumers
are unaffected — they call `buildForwardHeadersAsync(request)` and get back headers.

### Environment variables

Remove: `OC_TOKEN`, `OC_USER`, `OC_EMAIL`, `ENABLE_OC_WHOAMI`
Add: `SSO_CLIENT_ID`, `SSO_CLIENT_SECRET`, `SSO_ISSUER_URL`
Keep: `DISABLE_AUTH` (mock mode for local dev)

## Local Keycloak Dev Setup

### Kind overlay additions

Add a Keycloak Deployment to the Kind overlay (`overlays/kind/`):

- Image: `quay.io/keycloak/keycloak` (`start-dev` mode)
- Single replica, H2 in-memory (no persistence needed for dev)
- Realm import via `--import-realm` flag with ConfigMap-mounted JSON
- Service: `keycloak-service` on port 8080
- NodePort or Ingress for browser access from developer workstation

### Realm export JSON

Store at `components/manifests/overlays/kind/keycloak-realm.json`:

```json
{
  "realm": "ambient",
  "enabled": true,
  "clients": [
    {
      "clientId": "ambient-frontend",
      "enabled": true,
      "protocol": "openid-connect",
      "publicClient": false,
      "secret": "dev-secret-do-not-use-in-prod",
      "redirectUris": ["http://localhost:3000/api/auth/callback/keycloak"],
      "postLogoutRedirectUris": ["http://localhost:3000"],
      "webOrigins": ["http://localhost:3000"],
      "standardFlowEnabled": true,
      "directAccessGrantsEnabled": true,
      "serviceAccountsEnabled": true
    },
    {
      "clientId": "ambient-cli",
      "enabled": true,
      "protocol": "openid-connect",
      "publicClient": true,
      "redirectUris": ["http://localhost:8400/callback"],
      "standardFlowEnabled": true
    }
  ],
  "users": [
    {
      "username": "developer",
      "email": "developer@local.dev",
      "enabled": true,
      "credentials": [{"type": "password", "value": "developer", "temporary": false}],
      "groups": ["ambient-admin"]
    }
  ],
  "groups": [
    {"name": "ambient-admin"}
  ]
}
```

### What it replaces in the Kind overlay

| Current file | Replaced by |
|-------------|-------------|
| `ambient-api-server-jwks-patch.yaml` (static JWKS ConfigMap) | Keycloak's live JWKS endpoint |
| `api-server-no-jwt-patch.yaml` (`--enable-jwt=false`) | `--enable-jwt=true --jwk-cert-url=http://keycloak-service:8080/realms/ambient/protocol/openid-connect/certs` |
| `test-user.yaml` (K8s SA with cluster-admin) | Keycloak dev user + `client_credentials` client for E2E |
| `DISABLE_AUTH=true` in frontend | Frontend configured with `SSO_ISSUER_URL=http://keycloak-service:8080/realms/ambient` |

### Environment variables for Kind

```
# Frontend
SSO_CLIENT_ID=ambient-frontend
SSO_CLIENT_SECRET=dev-secret-do-not-use-in-prod
SSO_ISSUER_URL=http://keycloak-service:8080/realms/ambient

# Backend / API server
JWKS_URL=http://keycloak-service:8080/realms/ambient/protocol/openid-connect/certs
JWT_AUDIENCE=ambient-frontend

# CLI (developer workstation, outside cluster)
ISSUER_URL=http://localhost:<keycloak-nodeport>/realms/ambient
```

### Deployed environments with Identity Brokering

For openshift-dev, mpp, and production, run a Keycloak instance with Identity Brokering:

1. Add an "OpenID Connect v1.0" Identity Provider in your Keycloak pointing to RH SSO
2. Register your Keycloak as a client in RH SSO (one-time ask to realm admin)
3. Create frontend/CLI clients in your Keycloak (full admin control)
4. Backends validate JWTs against your Keycloak's JWKS — same as Kind, different URL

This means the same Keycloak realm config (clients, roles) works across all environments.
The only difference is whether Keycloak authenticates users directly (Kind — local
credentials) or delegates to RH SSO (deployed — Identity Brokering).

## Manifest Changes

### Remove
- `components/oauth-proxy/` (kustomization, deployment patch, service patch)
- `overlays/production/frontend-oauth-patch.yaml`
- All overlay `kustomization.yaml` references to `oauth-proxy` component

### Add
- K8s Secret for SSO client credentials (mounted into frontend pod)
- Impersonation ClusterRole + ClusterRoleBinding (above)
- Kind overlay: Keycloak Deployment, Service, ConfigMap (realm JSON), NodePort/Ingress
- Kind overlay: frontend/backend env patches pointing to local Keycloak

### Update
- Frontend Service: route to port 3000 (Next.js) instead of 8443 (OAuth proxy)
- Frontend Deployment: remove OAuth proxy sidecar container
- Kind overlay: API server patch → `--enable-jwt=true` with Keycloak JWKS URL (replaces `--enable-jwt=false`)
- E2E overlay: use Keycloak `client_credentials` grant instead of K8s SA token

### Remove (Kind overlay)
- `ambient-api-server-jwks-patch.yaml` (static JWKS ConfigMap — Keycloak serves live JWKS)
- `api-server-no-jwt-patch.yaml` (JWT is now enabled with Keycloak as issuer)
- `test-user.yaml` (K8s SA test user — replaced by Keycloak dev user)

## Future Phases (from IAM Consolidation Proposal)

This workflow covers **Phase 1** only. The following phases are defined in
`docs/internal/proposals/iam-consolidation-plan.md` (PR #1466) and should be specced
separately when ready.

### Phase 2: API keys → SSO service accounts

Replace `CreateProjectKey()` (which creates K8s SAs + TokenRequest) with Keycloak Admin
API calls to create confidential clients. Users receive `client_id`/`client_secret`
instead of a K8s SA JWT.

**What goes away:**
- `ambient-key-*` ServiceAccount creation in `handlers/permissions.go`
- `ambient-key-*` RoleBinding creation
- TokenRequest minting for access keys
- `updateAccessKeyLastUsedAnnotation()` (SA annotation patching)
- TokenReview fallback in the auth middleware (all tokens become SSO JWTs)

**What's new:**
- Keycloak Admin API client in the backend
- `SSO_ADMIN_CLIENT_ID` / `SSO_ADMIN_CLIENT_SECRET` credentials
- Keycloak client roles mapping to `project:admin/edit/view`

**Prerequisite:** Keycloak Admin API access with `manage-clients` realm role.

### Phase 3: Runner auth → OIDC token exchange (RFC 8693)

Replace the RSA keypair exchange between runner and control plane with standard OIDC
token exchange. The runner exchanges its projected K8s SA token for an SSO-issued JWT.

**What goes away:**
- Operator: SA creation for `ambient-session-*`, TokenRequest minting, 45-min refresh loop
- Operator: Secret `ambient-runner-token-*` creation
- Control plane: entire `internal/tokenserver/` and `internal/keypair/` packages
- Control plane: Secret `ambient-cp-token-keypair`

**What's new:**
- Runner: OIDC token exchange on startup (exchange K8s SA token → SSO JWT)
- SSO: `ambient-runner-exchange` client with token exchange permission
- SSO: cluster JWKS registered as identity provider (so SSO can validate K8s SA tokens)

**Prerequisite:** SSO token exchange enabled; SSO trusts cluster JWKS.

### Phase 4: DB RBAC reconciler

Make the DB `role_bindings` table the single write plane for permissions. A reconciler
in the control plane watches DB changes and syncs K8s RoleBindings.

**Role mapping:** `project:owner` → `ambient-project-admin`, `project:editor` →
`ambient-project-edit`, `project:viewer` → `ambient-project-view`.

Fine-grained permissions (`credential:token-reader`, etc.) remain DB-only — K8s RBAC
enforces the coarse gate (project access), DB RBAC enforces fine-grained actions.

### Phase 5: Credential consolidation

Move per-user OAuth integration tokens from K8s Secrets to the `credentials` table.
Add `user_id` and `scope` columns. New routes: `GET/POST/DELETE /users/me/credentials`.

## ADR-0002 Supersedence

ADR-0002 chose "User token for all operations" (raw token passthrough) over impersonation
because the token was a K8s-native opaque token — passthrough was the simplest and most
direct approach. With the move to SSO JWTs, the core assumption changes:

- **ADR-0002 context:** token is K8s-native → passthrough works
- **New context:** token is SSO JWT → passthrough requires cluster OIDC federation

The security contract from ADR-0002 is preserved: user operations use user permissions,
RBAC is enforced by K8s, audit logs reflect the actual user. Only the mechanism changes
from raw token passthrough to impersonation.
