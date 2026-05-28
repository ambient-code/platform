# SSO Authentication Specification

## Purpose

The platform SHALL authenticate all human users via OpenID Connect (OIDC) with Red Hat
SSO and represent user identity as signed JWTs throughout the stack. This
replaces the current model where an OpenShift OAuth proxy sidecar produces opaque tokens
that are forwarded to backends.

The migration unifies the authentication model: every component that needs to know "who
is this user?" validates a JWT against the SSO issuer's JWKS endpoint — no component
relies on opaque tokens, OAuth proxy headers, or Kubernetes TokenReview for human user
identity.

## Identity Flow

```
Browser ──OIDC session cookie──▸ Next.js (BFF) ──JWT──▸ Backend / API Server
                                      │                        │
                                      │                        ├─ Validate JWT (JWKS)
                                      │                        ├─ Extract identity (claims)
                                      │                        └─ K8s client: SA token
                                      │                             + Impersonate-User
                                      │                             + Impersonate-Group
                                      ▼                              │
                                 Red Hat SSO                    K8s API Server
                             (confidential client)          (RBAC as impersonated user)
```

## Requirements

### Requirement: BFF OIDC Session Model

The frontend SHALL act as an OIDC confidential client using the Authorization Code Flow.
The browser SHALL receive an opaque, httpOnly, secure, SameSite OIDC session cookie —
never a raw JWT. The frontend server SHALL exchange the OIDC session for a JWT when
proxying requests to backend services.

The OIDC callback route SHALL coexist with existing integration auth routes under
`/api/auth/` (GitHub, GitLab, Jira, Google, Gerrit, CodeRabbit). The OIDC callback
MUST NOT conflict with or disrupt those routes.

#### Scenario: User login

- GIVEN a user navigates to the platform
- WHEN they are not authenticated
- THEN the frontend redirects to the SSO authorization endpoint
- AND the SSO login page is displayed

#### Scenario: OIDC callback

- GIVEN the user completes SSO authentication
- WHEN SSO redirects to the frontend OIDC callback route
- THEN the frontend exchanges the authorization code for tokens
- AND stores the OIDC session server-side
- AND sets an httpOnly, secure, SameSite cookie on the browser

#### Scenario: OIDC routes coexist with integration auth routes

- GIVEN existing integration auth routes at `/api/auth/{provider}/connect`, `/api/auth/{provider}/status`, etc.
- WHEN the OIDC callback route is added
- THEN integration auth routes continue to function unchanged
- AND the OIDC route does not shadow or intercept integration auth requests

#### Scenario: Authenticated API request

- GIVEN a user with a valid OIDC session cookie
- WHEN the browser makes an API request to the frontend
- THEN the frontend extracts the JWT from the server-side OIDC session
- AND forwards it as `Authorization: Bearer <jwt>` to the upstream backend

#### Scenario: Token refresh

- GIVEN a user's access token has expired but the refresh token is valid
- WHEN the user makes a request
- THEN the frontend refreshes the access token using the refresh token
- AND the OIDC session is updated transparently

#### Scenario: Logout

- GIVEN a user clicks logout
- WHEN the logout request is processed
- THEN the frontend destroys the server-side OIDC session
- AND clears the OIDC session cookie
- AND redirects to the SSO logout endpoint for single sign-out

### Requirement: JWT Validation

Every backend service that receives a user request SHALL validate the JWT before
processing. Validation SHALL verify: signature against the SSO issuer's JWKS endpoint,
`exp` (expiration), `iss` (issuer), and `aud` (audience). Services MUST reject tokens
that fail any check with HTTP 401.

#### Scenario: Valid JWT accepted

- GIVEN a request with a valid, unexpired JWT signed by the SSO issuer
- WHEN the backend receives the request
- THEN the request is processed normally
- AND user identity is extracted from standard OIDC claims (`sub`, `email`, `preferred_username`, `groups`)

#### Scenario: Expired JWT rejected

- GIVEN a request with an expired JWT
- WHEN the backend receives the request
- THEN the backend returns 401 Unauthorized

#### Scenario: Wrong audience rejected

- GIVEN a JWT with an `aud` claim that does not match the service's expected audience
- WHEN the backend receives the request
- THEN the backend returns 401 Unauthorized

#### Scenario: Tampered JWT rejected

- GIVEN a JWT with a modified payload but original signature
- WHEN the backend receives the request
- THEN signature verification fails
- AND the backend returns 401 Unauthorized

#### Scenario: JWKS key rotation

- GIVEN the SSO issuer rotates its signing keys
- WHEN a JWT signed with the new key is received
- THEN the backend fetches the updated JWKS
- AND validates the JWT against the new key

### Requirement: K8s Authorization via Impersonation

The legacy backend SHALL use its own ServiceAccount token for all Kubernetes API calls
and SHALL set impersonation headers to represent the authenticated user's identity.
K8s RBAC SHALL evaluate permissions as the impersonated user, preserving all existing
per-user RoleBindings and SelfSubjectAccessReview checks.

The backend ServiceAccount SHALL have a ClusterRole granting the `impersonate` verb on
`users`, `groups`, and `serviceaccounts` resources. The `serviceaccounts` resource is
required because API key tokens represent K8s ServiceAccount identities.

#### Scenario: List resources respects user RBAC

- GIVEN a user with access to Project A but not Project B
- WHEN the user lists AgenticSessions
- THEN the backend sets `Impersonate-User` to the user's identity from JWT claims
- AND K8s returns only AgenticSessions in Project A
- AND AgenticSessions in Project B are not visible

#### Scenario: Create resource with RBAC check

- GIVEN a user with `create` permission for AgenticSessions in a Project
- WHEN the user creates an AgenticSession
- THEN the backend validates the JWT
- AND sets impersonation headers on the K8s client
- AND the SSAR succeeds because the user has the required RoleBinding
- AND the backend creates the resource using its SA (existing pattern)

#### Scenario: Unauthorized create rejected

- GIVEN a user without `create` permission for AgenticSessions in a Project
- WHEN the user attempts to create an AgenticSession
- THEN the backend sets impersonation headers on the K8s client
- AND the SSAR fails
- AND the backend returns 403 Forbidden

#### Scenario: Audit trail preserved

- GIVEN a user performs an operation via impersonation
- WHEN K8s audit logging records the API call
- THEN the audit log entry includes the impersonated user identity
- AND the acting ServiceAccount identity

#### Scenario: Impersonation RBAC enforced

- GIVEN the backend ServiceAccount
- WHEN the SA attempts to impersonate a user
- THEN K8s verifies the SA has the `impersonate` verb on the appropriate resource
- AND the impersonation succeeds only if the RBAC binding exists

### Requirement: SSAR Compatibility

SelfSubjectAccessReview (SSAR) calls SHALL work identically under impersonation. The
backend SHALL issue SSARs via K8s clients configured with impersonation headers so that
K8s evaluates the impersonated user's permissions, not the ServiceAccount's permissions.

The SSAR result cache SHALL include the impersonated user identity in the cache key.
Under impersonation, the bearer token is the backend ServiceAccount's token (shared
across all requests), so caching by token alone would cause cross-user authorization
leaks.

#### Scenario: SSAR with impersonation

- GIVEN a user authenticated via JWT with email `user@example.com`
- WHEN the backend performs an SSAR to check if the user can list AgenticSessions in namespace `project-a`
- THEN the K8s client is configured with `Impersonate-User: user@example.com`
- AND K8s evaluates the SSAR against `user@example.com`'s RoleBindings
- AND the result reflects the user's actual permissions

#### Scenario: SSAR cache isolation

- GIVEN user A and user B both make requests
- WHEN the backend caches SSAR results
- THEN user A's cached result is NOT returned for user B
- AND cache keys include the impersonated identity

### Requirement: API Key Authentication

API keys (K8s ServiceAccount tokens) SHALL continue to be accepted as an alternative
to SSO JWTs. When the backend receives a bearer token that is not a valid JWT (fails
JWT parsing), it SHALL fall back to Kubernetes TokenReview to validate the token as a
ServiceAccount token. API key identity SHALL be resolved from the ServiceAccount's
annotations (existing pattern).

This dual-path authentication is required because API keys are minted as K8s
ServiceAccount tokens and cannot be replaced with SSO JWTs.

#### Scenario: API key accepted

- GIVEN a request with a valid K8s ServiceAccount token (API key)
- WHEN the backend receives the request
- THEN JWT validation fails (token is not a JWT)
- AND the backend falls back to TokenReview
- AND the token is validated as a K8s ServiceAccount
- AND user identity is resolved from the ServiceAccount's annotations

#### Scenario: API key impersonation

- GIVEN a validated API key with a resolved user identity
- WHEN the backend makes K8s API calls
- THEN impersonation headers reflect the API key's associated user
- AND RBAC is enforced for that user

#### Scenario: Invalid token rejected

- GIVEN a token that is neither a valid JWT nor a valid K8s ServiceAccount token
- WHEN the backend receives the request
- THEN both JWT validation and TokenReview fail
- AND the backend returns 401 Unauthorized

### Requirement: Identity Claim Mapping

User identity SHALL be derived from JWT claims. The following standard OIDC claims
SHALL be used:

| Claim | Maps to | Used for |
|-------|---------|----------|
| `sub` | User ID | Unique identifier, RoleBinding subjects |
| `email` | User email | Display, notifications, RoleBinding subjects |
| `preferred_username` | Username | Display, audit logs |
| `groups` | Group membership | Group-based RBAC, impersonation groups |

The platform SHALL support configuring which claim is used for the K8s `Impersonate-User`
value. The default SHALL be `email` to match existing RoleBinding subjects that use
email addresses.

#### Scenario: Identity extracted from JWT

- GIVEN a JWT with claims `{"sub": "f:abc:jsell", "email": "jsell@redhat.com", "preferred_username": "jsell", "groups": ["team-ambient"]}`
- WHEN the backend processes the request
- THEN `Impersonate-User` is set to `jsell@redhat.com`
- AND `Impersonate-Group` is set to `["team-ambient"]`

### Requirement: Runner Token Propagation

The runner SHALL continue to receive the human user's token as `caller_token` via the
`x-caller-token` header on AG-UI interactions. With SSO authentication, `caller_token`
is a JWT. The runner uses `caller_token` only for API server HTTP calls (credential
fetches, feedback), never for direct K8s API calls. The runner's own K8s access SHALL
continue to use its per-session ServiceAccount bot token.

#### Scenario: caller_token is a JWT

- GIVEN a user interacts with a running session via AG-UI
- WHEN the frontend proxies the interaction to the runner
- THEN the `x-caller-token` header contains the user's SSO JWT
- AND the runner uses it for credential fetch calls
- AND the runner falls back to `BOT_TOKEN` if the caller token is expired

### Requirement: CLI Authentication

The CLI SHALL authenticate via OIDC Authorization Code Flow with PKCE against the SSO
issuer. The CLI SHALL store the refresh token for automatic token renewal. The CLI
is a public client (it cannot hold a client secret).

#### Scenario: CLI login

- GIVEN a user runs the CLI login command
- WHEN the CLI initiates the OIDC flow
- THEN it opens the user's browser to the SSO authorization endpoint with PKCE challenge
- AND listens for the callback on a local port
- AND exchanges the authorization code for tokens
- AND persists the access token and refresh token

#### Scenario: CLI token refresh

- GIVEN a user's CLI access token has expired
- WHEN the user runs any CLI command
- THEN the CLI refreshes the token using the stored refresh token
- AND updates the stored tokens

### Requirement: Local Development Authentication

The Kind and local-dev environments SHALL include a Keycloak instance as part of the
dev stack, providing a real OIDC flow without requiring VPN access to Red Hat SSO.
This replaces the static JWKS ConfigMap, `DISABLE_AUTH=true` mock mode, and
`OC_TOKEN` / `ENABLE_OC_WHOAMI` env vars as the primary local auth mechanism.

Keycloak SHALL start with a pre-configured realm requiring no manual setup.
The realm configuration SHALL be version-controlled in the repository as a
Keycloak realm export (JSON).

The pre-configured realm SHALL include:

- A confidential client for the frontend BFF (redirect URI to localhost)
- A public client for the CLI (PKCE, redirect to localhost callback)
- A default dev user with admin-level project access and standard OIDC claims
  (`email`, `preferred_username`, `groups`)

The backend and API server SHALL validate JWTs against the local Keycloak's JWKS
endpoint using the same code path as production. No special dev-only validation
logic SHALL exist — the only difference is which JWKS endpoint is configured.

Mock identity mode (`DISABLE_AUTH=true`) MAY be retained as a lightweight fallback
for rapid iteration when the full OIDC flow is not needed. Mock identity mode
MUST NOT be available in production deployments.

#### Scenario: Kind cluster bootstrap includes Keycloak

- GIVEN a developer runs the Kind cluster bootstrap
- WHEN the cluster is ready
- THEN a Keycloak instance is running with the pre-configured realm
- AND the frontend, backend, and API server are configured to use it
- AND no manual Keycloak setup is required

#### Scenario: Developer login via local Keycloak

- GIVEN a running Kind cluster with Keycloak
- WHEN a developer navigates to the frontend
- THEN they are redirected to the local Keycloak login page
- AND they can log in with the pre-configured dev credentials
- AND the frontend receives a real JWT and establishes an OIDC session cookie

#### Scenario: Backend validates local Keycloak JWTs

- GIVEN a Kind cluster with Keycloak
- WHEN the backend receives a JWT signed by the local Keycloak
- THEN it validates the JWT against Keycloak's JWKS endpoint
- AND extracts identity from standard OIDC claims
- AND impersonation works with the dev user's identity
- AND the validation code path is identical to production

#### Scenario: CLI authenticates against local Keycloak

- GIVEN a running Kind cluster with Keycloak
- WHEN a developer runs the CLI login command targeting the local environment
- THEN the CLI performs OIDC auth code + PKCE against the local Keycloak
- AND receives a valid JWT

#### Scenario: Realm config is version-controlled

- GIVEN the Keycloak realm export JSON is stored in the repository
- WHEN a developer modifies the realm config (adds a client, changes roles)
- THEN the change is reviewed via normal pull request process
- AND all developers get the updated config on their next cluster bootstrap

#### Scenario: Mock identity fallback

- GIVEN `DISABLE_AUTH=true` is set in a local dev environment
- WHEN a request arrives without a JWT
- THEN the backend uses a configurable mock identity
- AND impersonation is set to the mock user
- AND this mode MUST NOT be available in production deployments

### Requirement: E2E Test Authentication

End-to-end tests SHALL authenticate without requiring interactive SSO login. The
platform SHALL support a non-interactive authentication path for test automation.
In Kind environments, E2E tests SHALL use the local Keycloak instance.

#### Scenario: E2E test with client_credentials grant

- GIVEN an E2E test environment with a Keycloak client_credentials client
- WHEN the test suite starts
- THEN it obtains a JWT via the client_credentials grant against Keycloak
- AND uses the JWT for all API requests during the test run

#### Scenario: E2E test against local Keycloak

- GIVEN a Kind cluster with the local Keycloak running
- WHEN the E2E test suite starts
- THEN it authenticates against the local Keycloak using pre-configured test credentials
- AND the backend validates the resulting JWT normally

#### Scenario: E2E token not exposed to browser

- GIVEN the E2E test authentication token
- WHEN the test framework injects the token
- THEN the token SHALL be injected server-side (via cookie or API route)
- AND SHALL NOT be exposed as a browser-accessible environment variable

### Requirement: Feature-Flagged Migration

The transition from OAuth proxy to SSO authentication SHALL be gated behind a feature
flag (`sso-authentication` in Unleash). During migration, the platform SHALL support
both authentication modes simultaneously. The feature flag SHALL control which
authentication path is active per deployment.

This is an infrastructure flag, not a user-facing feature toggle. It is not visible
in workspace settings and is not user-configurable. The ops team enables it
per-environment as part of the SSO rollout.

#### Scenario: Legacy mode (flag off)

- GIVEN the SSO auth feature flag is disabled
- WHEN a request arrives with an OAuth proxy header
- THEN the backend uses the existing OAuth proxy flow
- AND K8s calls use the opaque token directly as a bearer token

#### Scenario: SSO mode (flag on)

- GIVEN the SSO auth feature flag is enabled
- WHEN a request arrives with `Authorization: Bearer <jwt>`
- THEN the backend validates the JWT against the JWKS endpoint
- AND K8s calls use impersonation

#### Scenario: Flag removal

- GIVEN the SSO auth migration is complete across all environments
- WHEN the feature flag is removed
- THEN all OAuth proxy code paths, forwarded header handling, and opaque token
  support SHALL be removed
- AND the OAuth proxy sidecar manifests SHALL be deleted

### Requirement: Manifest Changes

The deployment manifests SHALL be updated to support the new authentication model.

#### Scenario: OAuth proxy sidecar removed

- GIVEN a production deployment with SSO auth enabled
- WHEN the frontend is deployed
- THEN no OAuth proxy sidecar container is present
- AND the frontend Service routes traffic directly to the Next.js container port

#### Scenario: SSO client credentials provisioned

- GIVEN a deployment with SSO auth enabled
- WHEN the frontend pod starts
- THEN a K8s Secret containing `SSO_CLIENT_ID`, `SSO_CLIENT_SECRET`, and `SSO_ISSUER_URL`
  is mounted into the frontend container

### Requirement: SSO Client Configuration

Each deployed environment SHALL have its own OIDC confidential client registered in
Red Hat SSO. The client SHALL be configured with:

- Client authentication enabled (confidential)
- Authorization Code grant type
- Valid redirect URI pointing to the frontend OIDC callback route
- Valid post-logout redirect URI pointing to the frontend root
- Web origins matching the frontend host (for CORS on the token endpoint)

Local development environments (Kind, local-dev) SHALL use a local Keycloak instance
with pre-configured clients instead of registering clients in Red Hat SSO.

In deployed environments where the platform operates its own Keycloak instance, that
instance MAY be federated to Red Hat SSO via Identity Brokering — Keycloak delegates
login to RH SSO but issues its own tokens. This provides full client management
autonomy without requiring RH SSO realm admin access.

#### Scenario: One client per environment

- GIVEN stage and production deployments
- WHEN SSO clients are registered
- THEN each environment has its own client with its own secret
- AND a compromised secret in one environment does not affect others

#### Scenario: Audience isolation

- GIVEN separate clients for stage and production
- WHEN a JWT is minted for the stage client
- THEN the `aud` claim contains the stage client ID
- AND the production backend rejects it because the audience does not match

#### Scenario: Backend impersonation RBAC provisioned

- GIVEN a deployment with SSO auth enabled
- WHEN the backend pod starts
- THEN the backend ServiceAccount has a ClusterRoleBinding granting `impersonate` verb
  on `users`, `groups`, and `serviceaccounts` resources

## Roadmap

This spec covers **Phase 1** of a broader IAM consolidation. The full roadmap, informed
by the [IAM consolidation proposal](../../docs/internal/proposals/iam-consolidation-plan.md)
(PR #1466), is:

| Phase | Scope | Depends on |
|-------|-------|------------|
| **1. SSO user auth + impersonation** (this spec) | Frontend BFF, backend JWT validation, K8s impersonation. API keys and runner auth unchanged. | SSO confidential client registration |
| **2. API keys → SSO service accounts** | Replace K8s SA-based API keys with Keycloak confidential clients. Eliminates TokenReview fallback, K8s SA creation, and `last-used-at` annotation patching. | Keycloak Admin API access (`manage-clients` realm role) |
| **3. Runner auth → OIDC token exchange** | Replace RSA keypair exchange with RFC 8693 token exchange. Runner exchanges projected K8s SA token for an SSO-issued JWT. Eliminates CP token server, RSA bootstrap, and operator 45-min refresh loop. | SSO token exchange enabled; SSO trusts cluster JWKS as identity provider |
| **4. DB RBAC reconciler** | DB `role_bindings` table becomes single write plane. Reconciler syncs K8s RoleBindings from DB state. Eliminates dual-grant problem (K8s RBAC + DB RBAC). | Phases 1-2 complete |
| **5. Credential consolidation** | Move per-user OAuth integration tokens (GitLab, Google, Jira, Gerrit, CodeRabbit) from K8s Secrets to the `credentials` table. Single audit trail and access control. | Phase 4 (DB RBAC) |

Phase 1 is designed to be independently shippable. Each subsequent phase removes a
category of K8s-managed identity state and moves it to SSO or the database, converging
toward a single IAM plane.

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| BFF with confidential client (not public client in browser) | IETF recommendation for web apps. Tokens never reach the browser, eliminating XSS-based token theft. Next.js already acts as a proxy, making BFF natural. |
| K8s impersonation (not cluster OIDC federation) | Platform MUST work on any K8s cluster (Kind, ROSA classic, ROSA HCP) without cluster-level OIDC configuration. Impersonation is a standard K8s feature available everywhere. |
| `email` claim as default impersonation identity | Existing RoleBindings use email addresses as subject names. Using `email` preserves all existing RBAC bindings without migration. |
| Feature-flagged migration (not big-bang cutover) | Enables incremental rollout, environment-by-environment. Legacy OAuth proxy path remains available as fallback. |
| Supersede ADR-0002 (not amend) | ADR-0002's core assumption — the auth token is a K8s-native opaque token — is no longer true. The security contract (user operations use user permissions) is preserved; only the mechanism changes. |
| CLI remains a public client with PKCE | CLIs cannot securely store client secrets. PKCE provides equivalent security for native apps per RFC 7636. |
| Dual-path auth (JWT + TokenReview) | API keys are K8s ServiceAccount tokens that cannot be replaced with SSO JWTs. The backend tries JWT first, falls back to TokenReview, preserving both authentication paths. |
| SSAR cache includes impersonated identity | Under impersonation, the bearer token is shared (backend SA). Caching by token alone would leak authorization decisions across users. |
| E2E tokens injected server-side | Browser-exposed test tokens (via `NEXT_PUBLIC_*` env vars) are an XSS risk. Server-side injection via cookies or API routes prevents accidental token exposure. |
| Local Keycloak for dev (not mock mode or static JWKS) | Real OIDC flow in dev catches integration issues early. Same validation code path as production — no dev-only auth logic to maintain. Replaces ad-hoc static JWKS ConfigMap, `DISABLE_AUTH`, and `OC_TOKEN` env vars. |
| Keycloak Identity Brokering for deployed environments | Federating to RH SSO provides full client management autonomy without requiring realm admin access. Only one client registration needed in RH SSO (the Keycloak instance itself). |

## References

- [Security Specification](security.spec.md) — identity boundaries, token propagation
- [K8s Client Usage Patterns](../standards/backend/k8s-client.spec.md) — user-scoped vs. SA client patterns
- [Security Standards](../standards/security/security.spec.md) — token handling, RBAC enforcement
- [ADR-0002](../../docs/internal/adr/0002-user-token-authentication.md) — superseded by this spec
- [OAuth 2.0 for Browser-Based Applications](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps) — BFF recommendation
- [K8s User Impersonation](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation)
- Migration workflow: `workflows/security/sso-migration.workflow.md`
- [IAM consolidation proposal](../../docs/internal/proposals/iam-consolidation-plan.md) (PR #1466) — full IAM audit and long-term consolidation plan
