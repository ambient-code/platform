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

## Manifest Changes

### Remove
- `components/oauth-proxy/` (kustomization, deployment patch, service patch)
- `overlays/production/frontend-oauth-patch.yaml`
- All overlay `kustomization.yaml` references to `oauth-proxy` component

### Add
- K8s Secret for SSO client credentials (mounted into frontend pod)
- Impersonation ClusterRole + ClusterRoleBinding (above)

### Update
- Frontend Service: route to port 3000 (Next.js) instead of 8443 (OAuth proxy)
- Frontend Deployment: remove OAuth proxy sidecar container
- E2E overlay: test JWT generation instead of SA token

## ADR-0002 Supersedence

ADR-0002 chose "User token for all operations" (raw token passthrough) over impersonation
because the token was a K8s-native opaque token — passthrough was the simplest and most
direct approach. With the move to SSO JWTs, the core assumption changes:

- **ADR-0002 context:** token is K8s-native → passthrough works
- **New context:** token is SSO JWT → passthrough requires cluster OIDC federation

The security contract from ADR-0002 is preserved: user operations use user permissions,
RBAC is enforced by K8s, audit logs reflect the actual user. Only the mechanism changes
from raw token passthrough to impersonation.
