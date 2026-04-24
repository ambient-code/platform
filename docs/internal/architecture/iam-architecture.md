# Ambient Platform — Full IAM Architecture

## THE BIG PICTURE (End-to-End Flow)

```
╔══════════════════════════════════════════════════════════════════════════════════════╗
║                           IDENTITY ENTRY POINTS                                      ║
╠══════════════════╦════════════════════════╦═════════════════╦════════════════════════╣
║   HUMAN (Browser) ║   CLI / SDK USER        ║  BOT / API KEY  ║  SERVICE (internal)    ║
║                  ║                        ║                 ║                        ║
║  RH SSO / OCP    ║  oc whoami -t          ║  K8s SA JWT     ║  OIDC client creds     ║
║  OAuth login     ║  sha256~... token      ║  (ambient-key-*)║  or AMBIENT_API_TOKEN  ║
╚══════╦═══════════╩═══════════╦════════════╩════════╦════════╩═══════════╦════════════╝
       │                       │                      │                    │
       ▼                       │                      │                    │
╔══════════════╗               │                      │                    │
║  OAuth Proxy ║               │                      │                    │
║  (sidecar)   ║               │                      │                    │
║              ║               │                      │                    │
║  Validates   ║               │                      │                    │
║  OCP token   ║               │                      │                    │
║              ║               │                      │                    │
║  Injects:    ║               │                      │                    │
║  X-Forwarded-║               │                      │                    │
║    User      ║               │                      │                    │
║    Email     ║               │                      │                    │
║    Groups    ║               │                      │                    │
║    Access-   ║               │                      │                    │
║    Token     ║               │                      │                    │
╚══════╦═══════╝               │                      │                    │
       │                       │                      │                    │
       ▼                       ▼                      │                    │
╔══════════════════════════════════════════╗          │                    │
║              NEXT.JS FRONTEND             ║          │                    │
║  (components/frontend)                   ║          │                    │
║                                          ║          │                    │
║  buildForwardHeadersAsync()              ║          │                    │
║  ┌──────────────────────────────────┐    ║          │                    │
║  │ Reads incoming headers           │    ║          │                    │
║  │ Passes through X-Forwarded-*     │    ║          │                    │
║  │ Sets BOTH:                       │    ║          │                    │
║  │   Authorization: Bearer <token>  │    ║          │                    │
║  │   X-Forwarded-Access-Token: ...  │    ║          │                    │
║  └──────────────────────────────────┘    ║          │                    │
║                                          ║          │                    │
║  /api/projects/[name]/* → proxy →        ║          │                    │
╚══════════════════╦═══════════════════════╝          │                    │
                   │                                   │                    │
                   └───────────────────────────────────┘                   │
                                           │                               │
                                           ▼                               ▼
╔══════════════════════════════════════════════════════════════════════════════════════╗
║                           BACKEND API SERVER                                         ║
║                      (components/backend)   SA: backend-api                          ║
║                                                                                      ║
║  MIDDLEWARE CHAIN (every request):                                                   ║
║  1. Logger (redacts tokens from logs)                                                ║
║  2. forwardedIdentityMiddleware()                                                    ║
║     ├─ X-Forwarded-User       → ctx["userID"]                                        ║
║     ├─ X-Forwarded-Email      → ctx["userEmail"]                                     ║
║     ├─ X-Forwarded-Groups     → ctx["userGroups"]                                    ║
║     ├─ X-Forwarded-Access-Token → ctx["forwardedAccessToken"]  ◄── PRIORITY 1        ║
║     ├─ Authorization: Bearer  → ctx["authorizationHeader"]      ◄── PRIORITY 2        ║
║     └─ if no X-Forwarded-User: resolveServiceAccountFromToken() ◄── BOT/API KEY path  ║
║         └─ TokenReview API → extracts SA name → reads annotation for userID          ║
║  3. CORS                                                                             ║
║  4. ValidateProjectContext() (per /api/projects/:name group)                         ║
║     ├─ extractRequestToken()  (X-Fwd-Access-Token > Bearer > ?token=)                ║
║     ├─ GetK8sClientsForRequest() → user-scoped K8s client                            ║
║     ├─ check globalSSARCache (30s TTL, SHA256 keyed)                                 ║
║     └─ SelfSubjectAccessReview: can user LIST agenticsessions in namespace?           ║
║         └─ 403 if denied, cache result                                               ║
║                                                                                      ║
║  TOKEN TYPES VALIDATED:                                                              ║
║  • sha256~...  (OCP tokens)           SSAR hit → K8s validates against cluster       ║
║  • eyJ...      (K8s SA JWT)           SSAR hit → K8s validates against cluster       ║
║  • ghp_...     (GitHub PAT)           SSAR hit → K8s validates against cluster       ║
║  • generic 20+ chars (bearer)         SSAR hit → K8s validates against cluster       ║
║                                                                                      ║
║  TWO K8s CLIENTS:                                                                    ║
║  • K8sClient / DynamicClient  (SA: backend-api) → privileged writes after RBAC check ║
║  • reqK8s / reqDyn            (user token)       → SSAR checks, list operations       ║
╚═══════╦════════════════════════╦═════════════════════════════════╦════════════════════╝
        │                        │                                 │
        │ create/update          │ credentials stored              │ OAuth token exchange
        │ AgenticSession CR      │ in K8s Secrets                  │ GitHub, Google, etc.
        ▼                        ▼                                 ▼
╔══════════════╗    ╔═════════════════════════╗    ╔═══════════════════════════════╗
║   KUBERNETES  ║    ║     K8s SECRETS          ║    ║  EXTERNAL OAuth PROVIDERS    ║
║   API SERVER  ║    ║                          ║    ║                               ║
║               ║    ║  gitlab-user-tokens      ║    ║  GitHub App   → JWT minted   ║
║  Validates    ║    ║  oauth-callbacks         ║    ║  GitHub PAT   → stored       ║
║  every token  ║    ║  {session}-google-oauth  ║    ║  Google Drive → refresh tok  ║
║  against      ║    ║  google-creds-{userID}   ║    ║  GitLab PAT  → stored        ║
║  cluster JWKS ║    ║  jira-creds-{userID}     ║    ║  Jira token  → stored        ║
║               ║    ║  gerrit-creds-{userID}   ║    ║  Gerrit HTTP → stored        ║
║  Enforces     ║    ║  coderabbit-creds-{uid}  ║    ║  Coderabbit  → stored        ║
║  RBAC at K8s  ║    ║  ambient-runner-token-*  ║    ╚═══════════════════════════════╝
║  level        ║    ║  ambient-cp-token-keypair║
╚═══════╦═══════╝    ╚═════════════════════════╝
        │
        │ (operator watches AgenticSession CRs)
        ▼
╔══════════════════════════════════════════════════════════════════════════════════════╗
║                           KUBERNETES OPERATOR                                         ║
║                      (components/operator)   SA: agentic-operator                    ║
║                                                                                      ║
║  On new AgenticSession CR:                                                           ║
║  1. Create ServiceAccount: ambient-session-<session-id>                              ║
║  2. Create Role + RoleBinding (least-privilege for runner)                           ║
║  3. Mint token: K8sClient.ServiceAccounts(ns).CreateToken(sa, ...)  ← JWT ~1hr      ║
║  4. Store in Secret: ambient-runner-token-<session-id>  key: k8s-token              ║
║  5. Create Job/Pod with:                                                             ║
║     • volumeMount: /var/run/secrets/ambient/bot-token (from above secret)           ║
║     • NetworkPolicy: ingress only from backend                                       ║
║  6. Every 45min: regenerate token, update secret (kubelet refreshes mount)          ║
║                                                                                      ║
║  On StopSession: delete Pod, Secret, RoleBinding, SA                                ║
╚═══════════════════════════════════╦════════════════════════════════════════════════════╝
                                    │ /var/run/secrets/ambient/bot-token (JWT)
                                    │ CP_TOKEN_URL env var
                                    ▼
╔══════════════════════════════════════════════════════════════════════════════════════╗
║                           RUNNER POD                                                  ║
║                      SA: ambient-session-<session-id>                                 ║
║                                                                                      ║
║  Two auth paths:                                                                     ║
║                                                                                      ║
║  PATH A: Call Backend API                                                            ║
║    Authorization: Bearer <k8s-token from /var/run/secrets/ambient/bot-token>         ║
║    → Backend validates via SSAR → handler executes                                   ║
║                                                                                      ║
║  PATH B: Call Control Plane for fresh API token                                      ║
║    POST /token                                                                       ║
║    Authorization: Bearer <RSA-encrypted(session-id)>                                 ║
║    → Control plane decrypts with RSA private key                                     ║
║    → Returns OIDC/static API token for downstream calls                              ║
╚═══════════════════════════════════╦════════════════════════════════════════════════════╝
                                    │ (calls for API token)
                                    ▼
╔══════════════════════════════════════════════════════════════════════════════════════╗
║                           AMBIENT CONTROL PLANE                                       ║
║                      (components/ambient-control-plane)  SA: ambient-control-plane   ║
║                                                                                      ║
║  Token Server (RSA-based exchange):                                                  ║
║  • Keypair stored in Secret: ambient-cp-token-keypair                                ║
║  • Receives: Bearer <RSA-encrypted(session-id)>                                      ║
║  • Decrypts session ID → validates → returns API token                               ║
║                                                                                      ║
║  Outbound auth to API server:                                                        ║
║  Either:                                                                             ║
║  • StaticTokenProvider: reads AMBIENT_API_TOKEN env var                              ║
║  • OIDCTokenProvider: client_credentials flow to RH SSO                             ║
║      OIDC_TOKEN_URL = https://sso.redhat.com/auth/realms/redhat-external/...        ║
║      OIDC_CLIENT_ID + OIDC_CLIENT_SECRET                                             ║
║      Caches token with 30s refresh buffer                                            ║
╚═══════════════════════════════════╦════════════════════════════════════════════════════╝
                                    │
                                    ▼
╔══════════════════════════════════════════════════════════════════════════════════════╗
║                      AMBIENT API SERVER  (Database-backed RBAC)                      ║
║                      (components/ambient-api-server)                                  ║
║                                                                                      ║
║  Authentication:                                                                     ║
║  1. ForwardedAccessToken middleware: X-Forwarded-Access-Token → Authorization header  ║
║  2. JWT validation: signature verified against RH SSO JWKS                           ║
║     • Dev:  secrets/kind-jwks.json (local file)                                      ║
║     • Prod: https://sso.redhat.com/.../openid-connect/certs                         ║
║  3. gRPC: AMBIENT_API_TOKEN (static) or GRPC_SERVICE_ACCOUNT (JWT username match)    ║
║                                                                                      ║
║  Authorization (DB RBAC):                                                            ║
║  DBAuthorizationMiddleware → queries PostgreSQL                                      ║
║  role_bindings → roles → permissions (resource:action JSON array)                   ║
║                                                                                      ║
║  PostgreSQL Schema:                                                                  ║
║  users: id, username, name, email                                                    ║
║  roles: id, name, permissions (JSON ["session:read", "credential:token", ...])       ║
║  role_bindings: user_id, role_id, scope (platform|project), scope_id                 ║
║  credentials: id, project_id, name, provider, token*, url, email                     ║
║               *token stored plaintext — no DB-level encryption                       ║
║                                                                                      ║
║  Built-in roles:                                                                     ║
║  platform:admin       ["*:*"]                                                        ║
║  platform:viewer      [read-only subset]                                             ║
║  project:owner        ["project:*", "agent:*", "session:*", ...]                    ║
║  project:editor       [create/update, not delete project]                            ║
║  project:viewer       [read-only]                                                    ║
║  agent:runner         [runtime identity for agent pods]                              ║
║  credential:token-reader  ["credential:token"]                                       ║
╚══════════════════════════════════════════════════════════════════════════════════════╝
```

---

## ALL SERVICE ACCOUNTS

### System Service Accounts (static, in manifests)

| SA Name | Namespace | ClusterRole | Key Permissions |
|---|---|---|---|
| `agentic-operator` | ambient-code | `agentic-operator` | Create/delete pods, jobs, PVCs, SAs, RoleBindings; mint SA tokens (`serviceaccounts/token`) |
| `ambient-control-plane` | ambient-code | `ambient-control-plane` | Manage projects/namespaces/RBAC |
| `backend-api` | ambient-code | `backend-api` | Create/update AgenticSessions, mint tokens, manage access keys |
| `frontend` | ambient-code | `ambient-frontend-auth` | TokenReview, authz checks |
| `ambient-backend` | ambient-system | `ambient-backend-cluster-role` | Legacy/backup backend |

### Dynamic Service Accounts (created at runtime)

| SA Name Pattern | Created By | Purpose | Bound Role |
|---|---|---|---|
| `ambient-session-<session-id>` | Operator | Runner pod identity | Least-privilege project Role (read ConfigMaps, etc.) |
| `ambient-key-<name>-<uid>` | Backend | User API access keys | `ambient-project-admin/edit/view` (user's choice) |

---

## ALL TOKEN TYPES IN THE SYSTEM

### User-Facing Tokens

| Token | Format | Source | Used For | Validated By |
|---|---|---|---|---|
| OCP/RH SSO bearer | `sha256~...` | `oc whoami -t` or browser login | All user API calls | K8s API server (SSAR) |
| K8s SA JWT | `eyJ...` | TokenRequest API | Access keys, runner pods | K8s API server (SSAR) |
| GitHub PAT | `ghp_...` | User creates in GitHub | Git operations | GitHub API |
| Generic bearer | 20+ chars | Various | SDK/CLI access | K8s SSAR |

### System Tokens

| Token | Source | Used For | Lifetime |
|---|---|---|---|
| Runner pod JWT | Operator → TokenRequest on `ambient-session-*` SA | Runner → Backend auth | ~1hr, refreshed every 45min |
| Access key JWT | Backend → TokenRequest on `ambient-key-*` SA | CI/CD → Backend auth | User-specified, max 1yr |
| Control plane OIDC token | RH SSO client_credentials flow | Control plane → API server | Short-lived, auto-refreshed (30s buffer) |
| Control plane static token | `AMBIENT_API_TOKEN` env var | Dev/simple deployments | Static (until rotated) |
| GitHub App installation token | Backend mints via GitHub App JWT | Git clone in sessions | ~1hr (GitHub enforced) |

### OAuth Integration Tokens

| Token | Provider | Stored In | Keyed By |
|---|---|---|---|
| Google access + refresh token | Google OAuth | K8s Secret (backend ns) | userID |
| GitLab PAT | User provides | K8s Secret `gitlab-user-tokens` | userID |
| Jira API token | User provides | K8s Secret (backend ns) | userID |
| Gerrit HTTP/cookie | User provides | K8s Secret (backend ns) | userID |
| CodeRabbit API key | User provides | K8s Secret (backend ns) | userID |
| Session-specific OAuth creds | OAuth callback | K8s Secret `{session}-{provider}-oauth` | session-scoped |

---

## ALL SECRETS IN THE SYSTEM

| Secret Name | Namespace | Contents | Owner |
|---|---|---|---|
| `gitlab-user-tokens` | project | GitLab PATs keyed by userID | Backend writes, runner reads |
| `gitlab-connections` | project | GitLab connection metadata | Backend |
| `oauth-callbacks` | backend | Temporary OAuth state (UUID keyed) | Backend (TTL) |
| `{session}-{provider}-oauth` | project | Session-scoped OAuth creds | Backend (GC via OwnerRef) |
| `google-creds-{hash}` | backend | Google OAuth access+refresh token | Backend |
| `jira-creds-{hash}` | backend | Jira URL + email + token | Backend |
| `gerrit-creds-{hash}` | backend | Gerrit instance credentials | Backend |
| `coderabbit-creds-{hash}` | backend | CodeRabbit API key | Backend |
| `ambient-runner-token-{session}` | project | Runner pod K8s JWT (`k8s-token` key) | Operator creates, pod mounts |
| `ambient-cp-token-keypair` | ambient-code | RSA-4096 pub+priv key for runner↔CP auth | Control plane |
| Access key SA token secrets | project | (managed by K8s) | K8s auto-manages for SA JWTs |

---

## HOW THE TOKEN HEADERS FLOW

```
Browser/User
  │
  │  (browser session cookie, managed by OAuth proxy)
  ▼
OAuth Proxy sidecar
  │
  │  Adds to every proxied request:
  │  X-Forwarded-User: alice
  │  X-Forwarded-Email: alice@example.com
  │  X-Forwarded-Groups: platform-admins,dev-team
  │  X-Forwarded-Access-Token: sha256~<cluster-token>
  ▼
Next.js Frontend API Route
  │
  │  buildForwardHeadersAsync() extracts all X-Forwarded-* headers
  │  Sets BOTH on backend call:
  │    X-Forwarded-Access-Token: sha256~<cluster-token>
  │    Authorization: Bearer sha256~<cluster-token>
  │  Also forwards: X-Forwarded-User, Email, Groups
  ▼
Backend API Server
  │
  │  forwardedIdentityMiddleware():
  │    ctx.userID    ← X-Forwarded-User
  │    ctx.userEmail ← X-Forwarded-Email
  │    ctx.userGroups← X-Forwarded-Groups
  │    ctx.token     ← X-Forwarded-Access-Token (priority 1)
  │                    Authorization: Bearer (priority 2)
  │
  │  ValidateProjectContext():
  │    GetK8sClientsForRequest(token) → user-scoped K8s client
  │    SSAR: can user LIST agenticsessions in project namespace?
  │    (cached 30s)
  │
  │  Handler:
  │    User token for READ operations (SSAR)
  │    Backend SA for WRITE operations (after SSAR validates)
  ▼
Kubernetes API Server
  (validates token signature against cluster JWKS)
```

---

## THE DUAL AUTHORIZATION MODEL

```
Every API request hits BOTH authorization layers:

Layer 1: Kubernetes RBAC (components/backend)
  ┌─────────────────────────────────────────────┐
  │  SelfSubjectAccessReview                     │
  │  "Can THIS TOKEN perform VERB on RESOURCE    │
  │   in NAMESPACE?"                             │
  │                                              │
  │  Enforced by: K8s API server                 │
  │  ClusterRoles: ambient-project-admin/edit/view│
  │  Source of truth: K8s RBAC objects           │
  └─────────────────────────────────────────────┘

Layer 2: Database RBAC (components/ambient-api-server)
  ┌─────────────────────────────────────────────┐
  │  DBAuthorizationMiddleware                   │
  │  "Does this JWT username have a role_binding │
  │   granting RESOURCE:ACTION in this project?" │
  │                                              │
  │  Enforced by: PostgreSQL query               │
  │  Roles: platform:admin, project:owner, etc.  │
  │  Source of truth: PostgreSQL DB              │
  └─────────────────────────────────────────────┘

These are INDEPENDENT systems. A user needs:
• K8s RBAC binding for backend operations
• DB role binding for ambient-api-server operations
```

---

## RH SSO / CLUSTER JWT — HOW THEY RELATE

```
Red Hat SSO (external OIDC provider)
  URL: https://sso.redhat.com/auth/realms/redhat-external
  │
  ├─► Issues user tokens for human login (browser OAuth flow)
  │     → OAuth proxy validates these, injects X-Forwarded-* headers
  │
  ├─► Issues service tokens for control plane (client_credentials flow)
  │     OIDC_CLIENT_ID + OIDC_CLIENT_SECRET → AMBIENT_API_TOKEN equiv
  │
  └─► JWKS endpoint used by ambient-api-server to verify JWT signatures
        https://sso.redhat.com/.../openid-connect/certs

OpenShift / Kubernetes API Server (cluster JWT issuer)
  │
  ├─► Issues user tokens (sha256~...) — what you get from `oc whoami -t`
  │     These are validated when backend does SSAR
  │
  ├─► Issues SA tokens via TokenRequest API
  │     Used for: runner pods, access keys
  │     Operator mints these for runners
  │     Backend mints these for user access keys
  │
  └─► TokenReview API — validates any bearer token against cluster
        Backend uses this to identify which SA called (for BOT_TOKEN path)

Key distinction:
  RH SSO tokens → validated by ambient-api-server (DB RBAC layer)
  OCP/K8s tokens → validated by K8s API server via SSAR (K8s RBAC layer)
  BOTH types accepted at backend — which layer you hit depends on which
  component you're calling.
```
