# RBAC Enforcement Specification

## Purpose

The ambient-api-server SHALL enforce scope-aware authorization on all API endpoints
(HTTP and gRPC) using the database-backed Role and RoleBinding model defined in the
[Ambient Data Model Spec](../api/ambient-model.spec.md). Every request that passes
authentication SHALL be evaluated against the caller's role bindings, restricting
access to the specific resources identified by each binding's scope. Users start with
zero permissions and gain access by creating projects (self-service) or receiving
grants from existing owners.

All user authentication is via JWT (SSO or local Keycloak). The
[SSO Authentication Spec](sso-authentication.spec.md) governs how JWTs are obtained
and validated; this spec governs what happens after authentication succeeds.

## Requirements

### Requirement: Scope-Aware Permission Evaluation

The authorization middleware SHALL evaluate permissions against the binding's scope
context, not just the permission string. A binding with `scope=project` and
`project_id=A` SHALL only authorize access to resources within Project A.

The middleware SHALL extract the resource scope context from the request URL. For
project-scoped routes (`/projects/{id}/...`), the project ID is the path parameter.
For top-level routes (`/sessions/{id}`), the middleware SHALL resolve the owning
project from the database when scope filtering is required.

Bindings at broader scopes grant access to all resources within that scope:

- `global` grants access to all resources on the platform
- `project` grants access to all resources within one project (agents, sessions, inbox)
- `agent` grants access to one agent and its sessions
- `session` grants access to one session run
- `credential` governs access to one credential record

Effective permissions = union of all bindings that match the request context. No deny
rules.

#### Scenario: Project-scoped binding restricts access

- GIVEN user A has `project:editor` bound with `scope=project`, `project_id=proj-1`
- WHEN user A calls `GET /projects/proj-2/agents`
- THEN the middleware returns 403 Forbidden
- AND user A's binding for proj-1 is not considered because proj-2 is requested

#### Scenario: Global binding grants cross-project access

- GIVEN user A has `platform:admin` bound with `scope=global`
- WHEN user A calls `GET /projects/proj-2/agents`
- THEN the request is authorized
- AND all agents in proj-2 are returned

#### Scenario: Agent-scoped binding restricts to one agent

- GIVEN user A has `agent:operator` bound with `scope=agent`, `agent_id=agent-1`
- WHEN user A calls `PATCH /projects/proj-1/agents/agent-2`
- THEN the middleware returns 403 Forbidden
- AND user A's binding for agent-1 does not grant access to agent-2

#### Scenario: Scope hierarchy inheritance

- GIVEN user A has `project:owner` bound with `scope=project`, `project_id=proj-1`
- WHEN user A calls `GET /projects/proj-1/agents/agent-1`
- THEN the request is authorized
- AND the project-scoped binding covers all agents within that project

#### Scenario: Multiple bindings evaluated as union

- GIVEN user A has `project:viewer` on proj-1 AND `project:editor` on proj-2
- WHEN user A calls `POST /projects/proj-2/agents`
- THEN the request is authorized via the proj-2 editor binding
- AND the proj-1 viewer binding does not interfere

### Requirement: Resource List Filtering

List endpoints SHALL return only resources the caller has access to, based on their
bindings. A user with bindings for Projects A and B SHALL see only resources from those
two projects, not the full table.

The middleware SHALL NOT return 403 for list endpoints when the caller has zero matching
resources — it SHALL return an empty list. List access is implicit: if a user holds any
binding that grants `read` or `list` on a resource type, they can call the list endpoint.
The response is filtered to resources within their authorized scope.

#### Scenario: Session list filtered by project bindings

- GIVEN user A has `project:viewer` on proj-1 only
- AND sessions exist in proj-1 and proj-2
- WHEN user A calls `GET /sessions`
- THEN only sessions belonging to proj-1 are returned
- AND sessions in proj-2 are omitted

#### Scenario: Project list filtered by bindings

- GIVEN user A has bindings for proj-1 and proj-3
- AND proj-1, proj-2, proj-3 exist
- WHEN user A calls `GET /projects`
- THEN only proj-1 and proj-3 are returned

#### Scenario: Platform viewer sees all

- GIVEN user A has `platform:viewer` with `scope=global`
- WHEN user A calls `GET /sessions`
- THEN all sessions across all projects are returned

#### Scenario: No bindings returns empty list

- GIVEN user A has no project bindings
- WHEN user A calls `GET /sessions`
- THEN an empty list is returned with HTTP 200
- AND the response is not 403

### Requirement: User Auto-Provisioning

The system SHALL automatically create a User record when a JWT-authenticated caller
is seen for the first time. The User record SHALL be populated from standard OIDC claims
(`sub`, `email`, `preferred_username`). No explicit `POST /users` is required for
bootstrap.

Auto-provisioning SHALL NOT grant any role bindings. The new user starts with zero
permissions and gains access by creating a project or receiving a grant from an
existing owner.

Auto-provisioning SHALL only apply to JWT-authenticated human users. Service callers
authenticating with the platform service token SHALL NOT trigger user auto-provisioning.

#### Scenario: First-time user auto-provisioned

- GIVEN a user authenticates via SSO for the first time
- AND no User record exists for their identity
- WHEN any authenticated API request is processed
- THEN a User record is created from the JWT claims
- AND no role bindings are created
- AND the request proceeds to authorization evaluation

#### Scenario: Existing user not duplicated

- GIVEN a user has an existing User record
- WHEN the user authenticates again
- THEN no duplicate User record is created
- AND the existing record is used

#### Scenario: Concurrent first-time requests are idempotent

- GIVEN a user authenticates for the first time
- WHEN two requests arrive simultaneously before either commits a User record
- THEN exactly one User record is created (upsert semantics)
- AND both requests proceed normally

#### Scenario: Service caller does not trigger auto-provisioning

- GIVEN a request authenticates with the platform service token
- WHEN the request is processed
- THEN no User record is created
- AND the request bypasses RBAC as a service caller

### Requirement: Bootstrap via Project Creation

Any authenticated user SHALL be able to create a project. `POST /projects` SHALL be
exempt from authorization checks — only authentication (valid JWT) is required. On
successful project creation, the system SHALL automatically create a `project:owner`
RoleBinding for the authenticated user, scoped to the new project.

This is the platform's bootstrap mechanism. Users start with zero bindings and gain
access by creating a project.

#### Scenario: New user creates their first project

- GIVEN a user authenticates via SSO for the first time
- AND the user has zero role bindings
- WHEN the user calls `POST /projects` with `{"name": "my-project"}`
- THEN the project is created
- AND a RoleBinding is created: `role=project:owner`, `scope=project`, `project_id=my-project`, `user_id=<caller>`
- AND the user can immediately manage the project

#### Scenario: Project owner binding is atomic with creation

- GIVEN a user creates a project
- WHEN the project is persisted
- THEN the RoleBinding is created in the same database transaction
- AND if the RoleBinding creation fails, the project creation is rolled back

#### Scenario: New user cannot list other projects

- GIVEN a new user with zero bindings
- WHEN the user calls `GET /projects`
- THEN an empty list is returned
- AND no other users' projects are visible

#### Scenario: New user cannot access existing resources

- GIVEN a new user with zero bindings
- WHEN the user calls `GET /sessions` or `GET /projects/other-project`
- THEN the sessions list is empty
- AND the project get returns 404 (existence not disclosed)

### Requirement: Credential Self-Service Bootstrap

Any authenticated user SHALL be able to create a credential. `POST /credentials` SHALL
be exempt from authorization checks — only authentication is required. On successful
credential creation, the system SHALL automatically create a `credential:owner`
RoleBinding for the authenticated user, scoped to the new credential.

Binding a credential to a project (`POST /role_bindings` with `scope=credential`)
SHALL require the caller to hold **both** `credential:owner` on the credential being
bound AND `project:owner` on the target project.

#### Scenario: User creates a credential

- GIVEN an authenticated user
- WHEN the user calls `POST /credentials` with a valid payload
- THEN the credential is created
- AND a RoleBinding is created: `role=credential:owner`, `scope=credential`, `credential_id=<new-id>`, `user_id=<caller>`

#### Scenario: Credential owner binding is atomic with creation

- GIVEN a user calls `POST /credentials`
- WHEN the credential is persisted
- THEN the `credential:owner` RoleBinding is created in the same database transaction
- AND if the RoleBinding creation fails, the credential creation is rolled back

#### Scenario: Credential owner binds to their project

- GIVEN user A owns credential C and holds `project:owner` on proj-1
- WHEN user A calls `POST /role_bindings` with `scope=credential`, `credential_id=C`, `project_id=proj-1`
- THEN the binding is created
- AND runners in proj-1 can access credential C

#### Scenario: Non-project-owner cannot bind credential to project

- GIVEN user B does NOT hold `project:owner` on proj-1
- WHEN user B calls `POST /role_bindings` with `scope=credential`, `credential_id=C`, `project_id=proj-1`
- THEN the request returns 403 Forbidden

#### Scenario: Non-credential-owner cannot bind credential to project

- GIVEN user B holds `project:owner` on proj-1 but does NOT hold `credential:owner` on credential C
- WHEN user B calls `POST /role_bindings` with `scope=credential`, `credential_id=C`, `project_id=proj-1`
- THEN the request returns 403 Forbidden

#### Scenario: Credential list filtered by ownership

- GIVEN user A owns credentials C1 and C2
- AND user B owns credential C3
- WHEN user A calls `GET /credentials`
- THEN only C1 and C2 are returned

### Requirement: Platform Admin Seeding

The first `platform:admin` binding SHALL be created via a CLI command or database
migration, not through the API. This breaks the bootstrap chicken-and-egg: RBAC
endpoints for role binding mutation are themselves RBAC-gated, so the first admin
cannot grant themselves access through the API.

The platform SHALL provide a CLI command to seed the initial admin binding. Subsequent
admins can be granted access through the API by existing admins.

#### Scenario: Seed first admin via CLI

- GIVEN a fresh deployment with no role bindings
- WHEN an operator runs the admin seeding CLI command with a username
- THEN a RoleBinding is created: `role=platform:admin`, `scope=global`, `user_id=<username>`
- AND the admin can now manage all platform resources via the API

#### Scenario: Existing admin grants new admin

- GIVEN user A has `platform:admin`
- WHEN user A calls `POST /role_bindings` with `role_id=<platform:admin role>`, `scope=global`, `user_id=B`
- THEN user B receives platform:admin access

#### Scenario: Non-admin cannot create global bindings

- GIVEN user A has `project:owner` on proj-1 (but not platform:admin)
- WHEN user A calls `POST /role_bindings` with `scope=global`
- THEN the request returns 403 Forbidden

### Requirement: RoleBinding Mutation Authorization

Creating, updating, and deleting role bindings SHALL be authorized based on the
caller's existing bindings. A user SHALL only be able to grant roles **strictly
below** their own level in the role hierarchy. This prevents privilege escalation —
no user can mint a peer at their own tier. The sole exception is `platform:admin`,
which MAY grant `platform:admin` to others (since there is no higher role).

The role hierarchy for escalation checks (higher number = lower privilege):

| Level | Roles |
|-------|-------|
| 0 | `platform:admin` (may grant at own level) |
| 1 | `project:owner`, `credential:owner` |
| 2 | `project:editor`, `agent:operator`, `credential:viewer` |
| 3 | `project:viewer`, `agent:observer` |

For credential-scoped role bindings (`scope=credential`), the caller SHALL hold
`credential:owner` on the target credential in addition to satisfying the level
hierarchy check. This prevents users with unrelated project ownership from granting
credential access on credentials they do not own.

Platform-internal roles (`agent:runner`, `credential:token-reader`) SHALL NOT be
grantable via `POST /role_bindings`. These roles are managed exclusively by the
platform (e.g., the operator grants `agent:runner` to session service accounts at
session start, and `credential:token-reader` to runner pods). Attempting to grant
a platform-internal role via the API SHALL return 403 Forbidden.

#### Scenario: Project owner grants project editor

- GIVEN user A has `project:owner` on proj-1
- WHEN user A calls `POST /role_bindings` with `role=project:editor`, `scope=project`, `project_id=proj-1`, `user_id=B`
- THEN the binding is created
- AND user B gains editor access to proj-1

#### Scenario: Project owner cannot grant project owner

- GIVEN user A has `project:owner` on proj-1
- WHEN user A calls `POST /role_bindings` with `role=project:owner`, `scope=project`, `project_id=proj-1`, `user_id=B`
- THEN the request returns 403 Forbidden
- AND owners cannot mint peers at their own level

#### Scenario: Platform admin grants platform admin

- GIVEN user A has `platform:admin`
- WHEN user A calls `POST /role_bindings` with `role=platform:admin`, `scope=global`, `user_id=B`
- THEN the binding is created
- AND this is the sole exception to the "strictly below" rule

#### Scenario: Project owner cannot grant on other projects

- GIVEN user A has `project:owner` on proj-1 only
- WHEN user A calls `POST /role_bindings` with `scope=project`, `project_id=proj-2`
- THEN the request returns 403 Forbidden

#### Scenario: Project editor cannot grant project owner

- GIVEN user A has `project:editor` on proj-1
- WHEN user A calls `POST /role_bindings` with `role=project:owner`, `scope=project`, `project_id=proj-1`
- THEN the request returns 403 Forbidden
- AND editors cannot escalate to owner

#### Scenario: Non-credential-owner cannot grant credential-scoped roles

- GIVEN user B holds `project:owner` on proj-1 but does NOT hold `credential:owner` on credential C
- WHEN user B calls `POST /role_bindings` with `role=credential:viewer`, `credential_id=C`, `user_id=X`
- THEN the request returns 403 Forbidden
- AND project ownership does not substitute for credential ownership

#### Scenario: Granting platform-internal role rejected

- GIVEN user A has `platform:admin`
- WHEN user A calls `POST /role_bindings` with `role=agent:runner`
- THEN the request returns 403 Forbidden
- AND platform-internal roles are not user-grantable

#### Scenario: Project owner can revoke bindings in their project

- GIVEN user A has `project:owner` on proj-1
- AND user B has a `project:viewer` binding on proj-1
- WHEN user A calls `DELETE /role_bindings/{binding-id}`
- THEN the binding is deleted
- AND user B loses viewer access to proj-1

#### Scenario: Cannot delete own last owner binding on a project

- GIVEN user A is the sole `project:owner` on proj-1
- WHEN user A calls `DELETE /role_bindings/{own-owner-binding}`
- THEN the request returns 409 Conflict
- AND the system prevents orphaned projects with no owner

#### Scenario: Cannot delete sole credential owner binding

- GIVEN user A is the sole `credential:owner` on credential C
- WHEN user A calls `DELETE /role_bindings/{own-credential-owner-binding}`
- THEN the request returns 409 Conflict
- AND the system prevents orphaned credentials with no owner

### Requirement: Auth-Exempt Endpoints

The following endpoints SHALL require only authentication (valid JWT), not authorization.
They are necessary for system operation and bootstrap.

| Endpoint | Reason |
|----------|--------|
| `POST /projects` | Bootstrap — users gain access by creating a project |
| `POST /credentials` | Self-service — users manage their own credentials |
| `GET /roles` | Discovery — users need to see available roles |
| `GET /roles/{id}` | Discovery — read a specific role's permissions |

Health, metrics, and version endpoints are already bypassed at the authentication
layer and do not reach the authorization middleware.

All other endpoints SHALL require both authentication and authorization.

### Requirement: gRPC Authorization

gRPC handlers SHALL enforce the same authorization rules as HTTP handlers. The gRPC
authorization interceptor SHALL extract the caller identity from the request metadata
and evaluate permissions using the same scope-aware logic as the HTTP middleware.

#### Scenario: gRPC session watch authorized

- GIVEN user A has `project:viewer` on proj-1
- WHEN user A opens a gRPC `WatchSessions` stream
- THEN only session events for proj-1 are streamed
- AND events for other projects are filtered out

#### Scenario: gRPC session watch unauthorized

- GIVEN user A has no bindings
- WHEN user A opens a gRPC `WatchSessions` stream
- THEN no session events are streamed
- AND the stream remains open but idle (no error for watches)

#### Scenario: Idle watch stream resource limit

- GIVEN a caller opens multiple gRPC watch streams with no matching bindings
- WHEN the streams have been idle (no events delivered) beyond the server's idle timeout
- THEN the server SHALL close idle streams to prevent connection exhaustion
- AND the client MAY reconnect

#### Scenario: gRPC inbox push authorized

- GIVEN user A has `project:editor` on proj-1
- WHEN user A sends a gRPC `PushInboxMessage` for an agent in proj-1
- THEN the message is accepted

#### Scenario: gRPC inbox push unauthorized

- GIVEN user A has `project:viewer` on proj-1 (read-only)
- WHEN user A sends a gRPC `PushInboxMessage` for an agent in proj-1
- THEN the request returns a permission denied error

### Requirement: Service Caller Bypass

Requests originating from internal platform services (control plane, operator) SHALL
bypass RBAC authorization. Service callers are identified by authenticating with
the platform service token. Service callers are trusted infrastructure — they
reconcile state on behalf of the platform, not on behalf of any individual user.

The service caller bypass SHALL apply to both HTTP and gRPC request paths.

The service token endpoint SHALL only be reachable from within the cluster.
Exfiltration of the service token MUST NOT grant external access. See
[Security Specification — Proxy Authentication](security.spec.md#requirement-proxy-authentication)
for cluster-internal caller validation requirements.

#### Scenario: Control plane updates session status

- GIVEN the control plane authenticates with the service token
- WHEN the CP calls `PATCH /sessions/{id}` to update session status
- THEN the request is authorized regardless of RBAC bindings

#### Scenario: Service token not available to external callers

- GIVEN an external caller with a valid user JWT
- WHEN the caller's request is evaluated
- THEN the caller is not identified as a service caller
- AND RBAC is enforced normally

#### Scenario: Service token rejected from outside the cluster

- GIVEN an external caller who has obtained the service token
- WHEN the caller sends a request with the service token from outside the cluster
- THEN the request is rejected
- AND the service token bypass does not apply to non-cluster-internal traffic

### Requirement: Integration Test Coverage

Integration tests SHALL exercise RBAC enforcement. The test environment SHALL NOT
disable the authorization middleware. Tests SHALL create roles and role bindings
explicitly and verify that authorization is enforced.

Each plugin's integration test suite SHALL include at least:
- A test that verifies access is granted with the correct binding
- A test that verifies access is denied without a binding
- A test that verifies scope isolation (binding for resource A does not grant access to resource B)

#### Scenario: Test creates binding and verifies access

- GIVEN the integration test environment with RBAC enabled
- WHEN a test creates a user, a project, and a `project:editor` binding
- THEN the user can create agents in that project
- AND a second user without a binding receives 403

#### Scenario: Auth-exempt endpoints work without bindings

- GIVEN a test user with zero bindings
- WHEN the user calls `POST /projects`
- THEN the project is created successfully
- AND a `project:owner` binding exists for the user

### Requirement: Production Rollout

RBAC enforcement SHALL be gated behind a configuration flag. The production
environment SHALL explicitly disable enforcement initially, then enable it after:

1. The admin seeding CLI command has been run to create the first admin
2. Auth-exempt endpoints are verified in staging
3. Existing users have been granted appropriate bindings (manually or via migration)

The rollout SHALL NOT require downtime.

The `project:owner` and `credential:owner` RoleBindings SHALL be created on
`POST /projects` and `POST /credentials` regardless of whether RBAC enforcement
is enabled. Binding creation is not gated by the enforcement flag — only
authorization evaluation is. This prevents projects and credentials created during
the rollout window from becoming orphaned when enforcement is enabled.

#### Scenario: Enforcement disabled — all authenticated requests pass

- GIVEN enforcement is disabled via configuration
- WHEN an authenticated user calls any endpoint
- THEN RBAC is not evaluated
- AND the request proceeds

#### Scenario: Bindings created even when enforcement is disabled

- GIVEN enforcement is disabled via configuration
- WHEN a user calls `POST /projects` with `{"name": "new-proj"}`
- THEN the project is created
- AND a `project:owner` RoleBinding is created for the caller
- AND when enforcement is later enabled, the project has an owner

#### Scenario: Enforcement enabled

- GIVEN enforcement is enabled
- AND the admin has been seeded
- WHEN an authenticated user calls `GET /projects/proj-1`
- THEN the middleware checks the user's bindings for proj-1
- AND returns 403 if no matching binding exists

### Requirement: Error Response Opacity

The authorization middleware SHALL NOT disclose which permissions are missing or which
bindings were evaluated. This prevents authorization probing attacks.

For singleton resource endpoints (`GET /projects/{id}`, `GET /sessions/{id}`, etc.),
the middleware SHALL return 404 when the caller has no binding that covers the
requested resource — regardless of whether the resource actually exists. Returning
403 on a singleton GET leaks resource existence and enables ID enumeration.

For list endpoints, the middleware SHALL return 200 with an empty items array when
the caller has no matching resources.

#### Scenario: Singleton GET returns 404, not 403

- GIVEN user A has no binding covering proj-1
- WHEN user A calls `GET /projects/proj-1`
- THEN the response is 404
- AND no information about whether proj-1 exists is disclosed

#### Scenario: Forbidden response on mutation is opaque

- GIVEN user A lacks permission to mutate a resource they can see
- WHEN user A calls `PATCH /projects/proj-1` or `DELETE /projects/proj-1`
- THEN the response is 403 with a generic error body
- AND no details about required permissions or existing bindings are included

#### Scenario: List endpoint with no access returns empty

- GIVEN user A has no bindings matching a list query
- WHEN user A calls a list endpoint
- THEN the response is 200 with an empty items array
- AND no 403 is returned

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Project creation is auth-exempt (bootstrap entry point) | Users gain their first RoleBinding by creating a project. No auto-provisioning of permissions, no admin approval required. Self-service from day one. Alternative (auto-grant on first login) was rejected — it grants access to resources the user didn't ask for and complicates revocation. |
| User auto-provisioned from JWT, not via explicit registration | Users should not need to call a separate registration endpoint. The User record is a side-effect of authentication, not a privileged operation. This eliminates a bootstrap step without granting any permissions. |
| Credential creation is auth-exempt | Same self-service pattern as projects. Users own what they create. Binding credentials to projects requires both `credential:owner` and `project:owner`, preventing unauthorized sharing in either direction. |
| Admin seeded via CLI, not API | Breaks the chicken-and-egg. The RBAC endpoints are themselves gated, so the first admin cannot bootstrap through the API. A CLI command or migration is the standard pattern for initial admin seeding. |
| Scope-aware evaluation, not flat permission check | A flat check ("does this user have `project:read` anywhere?") leaks access across projects. Scope-aware evaluation checks "does this user have `project:read` on *this specific project*?" — the fundamental invariant for multi-tenancy. |
| List endpoints return filtered results, not 403 | Returning 403 on list endpoints breaks pagination and discoverability. An empty list is the correct response when the user has no access. This matches K8s behavior (RBAC-filtered list responses). |
| Service callers bypass RBAC | The control plane and operator are trusted infrastructure. They reconcile state on behalf of the platform. Requiring them to hold user-level bindings would create circular dependencies and make reconciliation fragile. |
| No deny rules | Union-only permission model. Simpler mental model, simpler evaluation. If a user should not have access, don't grant a binding. Deny rules create ordering problems and make debugging harder. |
| Cannot delete last owner binding (projects and credentials) | Prevents orphaned resources. An ownerless project or credential cannot be administered — no one can grant access, update, or delete it. The system enforces at least one owner per project and per credential. Platform admins can always intervene as a recovery path. |
| Strictly-below escalation with platform:admin exception | Users can only grant roles strictly below their own level. This prevents peer-minting (an owner creating another owner). `platform:admin` is the sole exception — since there is no higher role, admins must be able to grant admin to others. |
| Platform-internal roles not user-grantable | `agent:runner` and `credential:token-reader` are managed by the platform (operator grants them at session start). Allowing users to grant these roles would bypass the intended runtime-only access model and create security gaps. |
| Idle gRPC watch streams are closed by the server | An unauthenticated or unauthorized caller could open unlimited idle watch streams to exhaust server connections. The server closes streams that have been idle beyond a timeout to bound resource usage. |
| 404 on unauthorized singleton GETs, not 403 | Returning 403 on `GET /projects/{id}` confirms the resource exists to an unauthorized caller. Returning 404 prevents ID enumeration — the caller learns nothing about whether the resource exists. List endpoints correctly return 200+empty, which is safe because they don't confirm specific IDs. |
| Credential-scoped grants require credential:owner | Without this check, any Level 1+ user (e.g., a project:owner) could grant credential:viewer on credentials they don't own. The level hierarchy alone is insufficient — credential-scoped grants must also verify ownership of the target credential. |
| Owner bindings created regardless of enforcement flag | Projects and credentials created during the enforcement-disabled rollout window would become orphaned the moment enforcement is enabled. Creating bindings unconditionally means every resource has an owner from day one. |
| User auto-provisioning is idempotent (upsert) | Two concurrent first-time requests from the same user could race to create the User record. Upsert semantics (keyed on identity claim) prevent duplicate records and unhandled constraint violations. |
| Service token restricted to cluster-internal traffic | A stolen service token grants full RBAC bypass. Restricting acceptance to cluster-internal callers limits the blast radius of token exfiltration. |
| gRPC uses same evaluation as HTTP | One authorization model, not two. The gRPC interceptor uses the same evaluation logic as the HTTP middleware. Prevents divergence and bypass via protocol switching. |
| Tests exercise real RBAC | Disabling the middleware in tests means RBAC bugs ship to production undetected. Tests should create bindings explicitly and verify enforcement. The test helper should make this ergonomic, not skip it. |
| Configuration flag for rollout | Gradual enablement. Operators can seed admins and verify behavior in staging before enabling in production. No big-bang cutover. |
| Proxy routes out of scope | Routes forwarded by the proxy plugin to external backends are outside the scope of ambient-api-server RBAC. Those backends handle their own authorization. |

## References

- [Ambient Data Model Spec](../api/ambient-model.spec.md) — Role, RoleBinding schemas, built-in roles, permission matrix
- [Security Specification](security.spec.md) — identity boundaries, credential authorization model
- [SSO Authentication Spec](sso-authentication.spec.md) — JWT validation, identity claims
- [Security Standards](../standards/security/security.spec.md) — token handling, RBAC patterns
