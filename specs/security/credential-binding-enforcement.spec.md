# Credential Binding Enforcement

## Purpose

Credentials are global resources. Access to a credential's token at session runtime is governed by `scope=credential` RoleBindings that link a credential to a project or a specific agent within a project. The control plane resolves which credentials a session receives by walking these bindings from most-specific to least-specific scope. A credential with no binding covering the session's project and agent is not injected.

This spec defines the resolver algorithm, authorization rules for creating bindings at each level, and the `credential:token-reader` grant lifecycle.

### Dependencies

- **`project:admin` role**: This spec requires a new `project:admin` role (level 2) between `project:owner` (level 1) and `project:editor` (level 3). Credential binding authorization requires `project:admin` or higher. The role definition and hierarchy amendment belong in `rbac-enforcement.spec.md`; this spec assumes the role exists.
- **Global credential binding pattern**: The `scope=credential` binding with both `project_id=NULL` and `agent_id=NULL` is a new pattern. `ambient-model.spec.md` SHALL be amended to document this as valid for credential scope.
- **Session service identity**: The `credential:token-reader` lifecycle uses `user_id` in RoleBindings to represent the session's OIDC service account (e.g., `service-account-ambient-e2e`). This is the same identity the control plane already provisions via OIDC client_credentials grant — no new identity mechanism is required.

## Requirements

### Requirement: Hierarchical Credential Resolution

The control plane SHALL resolve credentials for a session by walking `scope=credential` RoleBindings from most-specific to least-specific scope: **agent → project → global**.

For each credential provider (github, jira, kubeconfig, google, gitlab, vertex):

1. If a `scope=credential` binding exists where `credential_id` references a credential of this provider, `project_id` matches the session's project, AND `agent_id` matches the session's agent — use that credential (**agent-level binding**).
2. Otherwise, if a `scope=credential` binding exists where `credential_id` references a credential of this provider, `project_id` matches the session's project, AND `agent_id` is NULL — use that credential (**project-level binding**).
3. Otherwise, if a `scope=credential` binding exists where `credential_id` references a credential of this provider, `project_id` is NULL, AND `agent_id` is NULL — use that credential (**global binding**).
4. Otherwise, no credential is injected for this provider.

The API server SHOULD reject creation of duplicate bindings at the same scope level for the same provider (same `credential.provider`, same `project_id`, same `agent_id`). If duplicates exist despite this, the binding with the earliest `created_at` timestamp wins.

#### Scenario: Agent-level binding overrides project-level

- GIVEN credential A (provider=github) is bound to project P with `agent_id=NULL`
- AND credential B (provider=github) is bound to project P with `agent_id=agent-1`
- WHEN a session starts for agent-1 in project P
- THEN the session receives credential B (agent-level wins)

#### Scenario: Project-level binding used when no agent-level exists

- GIVEN credential A (provider=github) is bound to project P with `agent_id=NULL`
- AND no agent-level github binding exists for agent-1 in project P
- WHEN a session starts for agent-1 in project P
- THEN the session receives credential A (project-level fallback)

#### Scenario: No binding means no injection

- GIVEN credential A (provider=github) is bound to project P
- AND no github credential is bound to project Q at any level
- WHEN a session starts in project Q
- THEN no github credential is injected into the session

#### Scenario: Multiple providers resolved independently

- GIVEN credential A (provider=github) is bound to project P at project-level
- AND credential B (provider=jira) is bound to project P at agent-level for agent-1
- AND no google credential is bound to project P
- WHEN a session starts for agent-1 in project P
- THEN the session receives credential A (github, project-level) and credential B (jira, agent-level)
- AND no google credential is injected

#### Scenario: Global binding provides default

- GIVEN credential A (provider=github) has a `scope=credential` binding with `project_id=NULL` and `agent_id=NULL`
- AND no project-level or agent-level github binding exists for project P
- WHEN a session starts in project P
- THEN the session receives credential A (global fallback)

#### Scenario: Agent-level binding overrides global

- GIVEN credential A (provider=github) has a global binding (`project_id=NULL`, `agent_id=NULL`)
- AND credential B (provider=github) is bound to project P with `agent_id=agent-1`
- WHEN a session starts for agent-1 in project P
- THEN the session receives credential B (agent-level overrides global)

### Requirement: Credential Binding Authorization

Creating or deleting `scope=credential` RoleBindings SHALL require authorization that depends on the binding's scope level.

**All credential bindings** require the caller to hold `credential:owner` on the target credential.

**Project-level bindings** (`project_id` set, `agent_id` NULL) additionally require the caller to hold `project:admin` or higher (level ≤ 2) on the target project.

**Agent-level bindings** (`project_id` set, `agent_id` set) additionally require:
1. The caller to hold `project:admin` or higher (level ≤ 2) on the project that owns the agent
2. The specified agent to belong to the specified project (validated by the API server)
3. The `project_id` to be non-NULL (agent-credential bindings without a project are invalid)

**Global bindings** (`project_id` NULL, `agent_id` NULL) additionally require the caller to hold `platform:admin`.

#### Scenario: Project admin binds credential to project

- GIVEN user A holds `credential:owner` on credential C
- AND user A holds `project:admin` on project P
- WHEN user A creates a RoleBinding with `scope=credential`, `credential_id=C`, `project_id=P`, `agent_id=NULL`
- THEN the binding is created (201)

#### Scenario: Project owner binds credential to specific agent

- GIVEN user A holds `credential:owner` on credential C
- AND user A holds `project:owner` on project P
- AND agent-1 belongs to project P
- WHEN user A creates a RoleBinding with `scope=credential`, `credential_id=C`, `project_id=P`, `agent_id=agent-1`
- THEN the binding is created (201)

#### Scenario: Project editor cannot bind credentials

- GIVEN user A holds `credential:owner` on credential C
- AND user A holds `project:editor` on project P
- WHEN user A creates a RoleBinding with `scope=credential`, `credential_id=C`, `project_id=P`
- THEN the request returns 403 Forbidden

#### Scenario: Non-project-member cannot bind credential to agent

- GIVEN user A holds `credential:owner` on credential C
- AND user A has no role on project P
- WHEN user A creates a RoleBinding with `scope=credential`, `credential_id=C`, `project_id=P`, `agent_id=agent-1`
- THEN the request returns 403 Forbidden

#### Scenario: Agent-credential binding requires project_id

- GIVEN user A holds `credential:owner` on credential C
- WHEN user A creates a RoleBinding with `scope=credential`, `credential_id=C`, `agent_id=agent-1`, `project_id=NULL`
- THEN the request returns 400 Bad Request
- AND the error indicates that agent-scoped credential bindings require a project_id

#### Scenario: Agent must belong to the specified project

- GIVEN user A holds `credential:owner` on credential C
- AND user A holds `project:owner` on project P
- AND agent-1 belongs to project Q (not P)
- WHEN user A creates a RoleBinding with `scope=credential`, `credential_id=C`, `project_id=P`, `agent_id=agent-1`
- THEN the request returns 400 Bad Request

#### Scenario: Platform admin creates global credential binding

- GIVEN user A holds `platform:admin`
- AND user A holds `credential:owner` on credential C
- WHEN user A creates a RoleBinding with `scope=credential`, `credential_id=C`, `project_id=NULL`, `agent_id=NULL`
- THEN the binding is created (201)

#### Scenario: Non-admin cannot create global credential binding

- GIVEN user A holds `credential:owner` on credential C
- AND user A does NOT hold `platform:admin`
- WHEN user A creates a RoleBinding with `scope=credential`, `credential_id=C`, `project_id=NULL`, `agent_id=NULL`
- THEN the request returns 403 Forbidden

### Requirement: credential:token-reader Grant Lifecycle

The control plane SHALL grant `credential:token-reader` to the session's OIDC service identity for each credential resolved by the hierarchical resolver. This grant SHALL be scoped to the specific credential and SHALL be revoked when the session terminates.

The control plane authenticates as a platform service account and creates these bindings via the standard `POST /role_bindings` API. Because `credential:token-reader` is an internal role, only service callers (not human users) can create these bindings.

#### Scenario: Token-reader granted at session start

- GIVEN credential A is resolved for a session via the hierarchical resolver
- WHEN the control plane provisions the session pod
- THEN a RoleBinding is created with `role=credential:token-reader`, `scope=credential`, `credential_id=A`, `user_id=<session-oidc-service-account>`

#### Scenario: Token-reader revoked at session end

- GIVEN a session was provisioned with `credential:token-reader` for credential A
- WHEN the session terminates (Completed, Failed, or Stopped)
- THEN the `credential:token-reader` RoleBinding for credential A is deleted

#### Scenario: Sidecar can fetch token with granted role

- GIVEN the control plane granted `credential:token-reader` for credential A to the session's service identity
- WHEN the credential sidecar calls `GET /credentials/{A}/token` with the session's bearer token
- THEN the API server returns the decrypted token (200)

#### Scenario: Sidecar cannot fetch unbound credential token

- GIVEN credential B was NOT resolved for this session (no binding)
- AND no `credential:token-reader` was granted for credential B
- WHEN the credential sidecar calls `GET /credentials/{B}/token`
- THEN the API server returns 404

### Requirement: Binding Deletion Does Not Affect Running Sessions

Deleting a `scope=credential` RoleBinding SHALL NOT terminate running sessions that were provisioned with the previously-bound credential. The credential remains available for the session's lifetime via its existing `credential:token-reader` grant. New sessions started after the binding deletion SHALL NOT receive the credential.

#### Scenario: Running session keeps credential after binding deleted

- GIVEN a session is Running with credential A (bound at project-level)
- WHEN the project-level binding for credential A is deleted
- THEN the running session continues to use credential A
- AND the `credential:token-reader` grant for this session is NOT revoked

#### Scenario: New session does not receive deleted binding's credential

- GIVEN the project-level binding for credential A on project P was deleted
- WHEN a new session starts in project P
- THEN credential A is NOT injected (resolver finds no matching binding)

## Migration

### Existing consumers

| Consumer | Current behavior | Required change |
|----------|-----------------|-----------------|
| Control plane `resolveCredentialIDs` | Lists all credentials via `sdk.Credentials().ListAll()`, picks first per provider | Query `scope=credential` RoleBindings filtered by `project_id` and `agent_id`, implement hierarchical resolution |
| RBAC middleware (credential binding creation) | Validates `credential:owner` + `project:owner` for project-level bindings | Add validation for agent-level bindings (verify agent belongs to project, caller has `project:admin`+), global bindings (require `platform:admin`), and reject `agent_id` without `project_id` |
| Credential sidecar entrypoint | Fetches token via bearer token from CP token exchange | No change — consumes `CREDENTIAL_IDS` produced by CP |
| Runner `populate_runtime_credentials` | Fetches tokens from `CREDENTIAL_IDS` env var | No change — consumes `CREDENTIAL_IDS` produced by CP |
| UI binding matrix | Creates RoleBindings with `credential_id` + `project_id` ± `agent_id` | No change — already creates correct binding structure |

### Specs requiring amendment

| Spec | Amendment |
|------|-----------|
| `rbac-enforcement.spec.md` | Add `project:admin` role at level 2; update credential binding authorization to require `project:admin`+ instead of `project:owner` |
| `ambient-model.spec.md` | Document global credential binding pattern (`scope=credential` with `project_id=NULL`, `agent_id=NULL`); add credential binding scope terms (agent-level, project-level, global) |
