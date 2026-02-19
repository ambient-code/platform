# Data Model Comparison: Backend → ambient-api-server Unification

> **Goal**: Replace the Kubernetes-backed `backend` component with `ambient-api-server` (Postgres-backed) as the single source of truth. The **OpenAPI spec** (`openapi/openapi.yaml`) is the canonical contract; the SDK wraps it into language-friendly clients; the control-plane reconciler and frontend consume the REST API.

---

## 1. Component Map (Current State)

| Component | Storage | Role | Fate |
|-----------|---------|------|------|
| **backend** | etcd (K8s CRDs + Namespaces + Secrets) | REST API for frontend. 88 endpoints. Gin + K8s dynamic client. | **Replace entirely** |
| **public-api** | None (stateless gateway) | Thin proxy: translates `/v1/sessions` ↔ backend's `/api/projects/:p/agentic-sessions/:s`. DTO transformation. | **Remove** — ambient-api-server serves the SDK contract directly |
| **ambient-api-server** | PostgreSQL (gorm) | REST API with 4 Kinds (Session, User, Project, ProjectSettings). rh-trex-ai framework. Session fully expanded (~30 fields). | **Expand** to cover all backend functionality |
| **operator** | Watches K8s CRDs | Creates Jobs/Pods from AgenticSession CRs. | **Keep** — unchanged. Reads CRs, not Postgres. |
| **control-plane** (planned) | Reads Postgres, writes K8s | Reconciler: Postgres rows → K8s resources (Session CR, Namespace, RoleBindings). Syncs CR status back to Postgres. | **Build** |
| **SDK** (Go + Python) | None | Client library. Wraps the OpenAPI spec into language-friendly clients. | **Expand** — derives types from OpenAPI spec |

---

## 2. Kind-by-Kind Gap Analysis

### 2.1 Kinds That Exist in Both (field comparison)

#### Session

Session fields have been expanded from the original 6-field stub to ~30 fields. All data and runtime status fields are now implemented.

 < /dev/null |  Field | Backend (CRD spec) | API Server (Postgres) | Status |
|-------|--------------------|-----------------------|--------|
| `name` / `displayName` | `spec.displayName` | `Name string` | ✅ |
| `prompt` / `initialPrompt` | `spec.initialPrompt` | `Prompt *string` | ✅ |
| `repos` | `spec.repos[]` | `Repos *string` (jsonb) | ✅ |
| `interactive` | `spec.interactive` (bool) | `Interactive *bool` | ✅ |
| `timeout` | `spec.timeout` (int, seconds) | `Timeout *int32` | ✅ |
| `llm_model` | `spec.llmSettings.model` | `LlmModel *string` | ✅ |
| `llm_temperature` | `spec.llmSettings.temperature` | `LlmTemperature *float64` | ✅ |
| `llm_max_tokens` | `spec.llmSettings.maxTokens` | `LlmMaxTokens *int32` | ✅ |
| `workflow_id` | `spec.activeWorkflow` | `WorkflowId *string` | ✅ |
| `created_by_user_id` | `spec.userContext.userId` | `CreatedByUserId *string` | ✅ |
| `assigned_user_id` | — | `AssignedUserId *string` | ✅ |
| `parent_session_id` | parent session ref | `ParentSessionId *string` | ✅ |
| `bot_account_name` | `spec.botAccount.name` | `BotAccountName *string` | ✅ |
| `resource_overrides` | `spec.resourceOverrides` | `ResourceOverrides *string` (jsonb) | ✅ |
| `environment_variables` | `spec.environmentVariables` | `EnvironmentVariables *string` (jsonb) | ✅ |
| `labels` | `metadata.labels` | `SessionLabels *string` (jsonb) | ✅ |
| `annotations` | `metadata.annotations` | `SessionAnnotations *string` (jsonb) | ✅ |
| `project_id` | derived from namespace | `ProjectId *string` | ✅ |
| **Status / Runtime fields** | | | |
| `phase` | `status.phase` | `Phase *string` | ✅ |
| `start_time` | `status.startTime` | `StartTime *time.Time` | ✅ |
| `completion_time` | `status.completionTime` | `CompletionTime *time.Time` | ✅ |
| `sdk_session_id` | `status.sdkSessionId` | `SdkSessionId *string` | ✅ |
| `sdk_restart_count` | `status.sdkRestartCount` | `SdkRestartCount *int32` | ✅ |
| `conditions` | `status.conditions[]` | `Conditions *string` (jsonb) | ✅ |
| `reconciled_repos` | `status.reconciledRepos[]` | `ReconciledRepos *string` (jsonb) | ✅ |
| `reconciled_workflow` | `status.reconciledWorkflow` | `ReconciledWorkflow *string` (jsonb) | ✅ |
| `kube_cr_name` | derived from `metadata.name` | `KubeCrName *string` | ✅ |
| `kube_cr_uid` | `metadata.uid` | `KubeCrUid *string` | ✅ |
| `kube_namespace` | `metadata.namespace` | `KubeNamespace *string` | ✅ |

**Session action endpoints:**

| Backend Endpoint | Purpose | API Server |
|-----------------|---------|------------|
| `POST .../clone` | Clone session to another project | **MISSING** |
| `POST .../start` | Trigger session start | ✅ Implemented |
| `POST .../stop` | Trigger session stop | ✅ Implemented |
| `PUT .../displayname` | Update display name | Covered by PATCH |
| `POST .../repos` | Add repo to session | **MISSING** |
| `DELETE .../repos/:name` | Remove repo from session | **MISSING** |
| `GET .../repos/status` | Get repos reconciliation status | **MISSING** |
| `POST .../workflow` | Select workflow for session | **MISSING** (PATCH covers `workflow_id`, but no git resolution) |
| `GET .../workspace`, `GET/PUT/DELETE .../workspace/*path` | File operations in running session | **MISSING** — proxied to running pod |
| `GET .../git/status`, `POST .../git/configure-remote`, `GET .../git/list-branches` | Git operations in running session | **MISSING** — proxied to running pod |
| `GET .../k8s-resources` | K8s resources for session | **MISSING** — reads from K8s |
| `GET .../workflow/metadata` | Workflow metadata from repo | **MISSING** — reads from git |
| `GET .../export` | Export session data (AG-UI events) | **MISSING** |
| `GET .../credentials/{provider}` | Runtime credential refresh | **MISSING** — reads from K8s secrets |

#### Workflow

| Field | Backend (CRD/types) | API Server (Postgres) | Gap |
|-------|--------------------|-----------------------|-----|
| `name` | — (workflows are git references, not named CRDs) | `Name string` | ✅ |
| `repo_url` | `WorkflowSelection.gitUrl` | `RepoUrl *string` | ✅ (naming differs) |
| `prompt` | — | `Prompt *string` | ✅ API-only |
| `agent_id` | — | `AgentId *string` | ✅ API-only |
| `branch` | `WorkflowSelection.branch` | — | **MISSING** |
| `path` | `WorkflowSelection.path` | — | **MISSING** |

#### User

| Field | Backend (types) | API Server (Postgres) | Gap |
|-------|----------------|-----------------------|-----|
| `username` | `UserContext.userId` | `Username string` | ✅ (naming differs) |
| `name` | `UserContext.displayName` | `Name string` | ✅ (naming differs) |
| `groups` | `UserContext.groups` ([]string) | — | **MISSING** — text[] or jsonb |

#### Agent

| Field | Backend | API Server (Postgres) | Gap |
|-------|---------|----------------------|-----|
| `name` | — | `Name string` | ✅ |
| `repo_url` | — | `RepoUrl *string` | ✅ |
| `prompt` | — | `Prompt *string` | ✅ |
| `project_id` | — | — | **ADD** — FK → projects, multi-tenant scoping |

#### Skill

| Field | Backend | API Server (Postgres) | Gap |
|-------|---------|----------------------|-----|
| `name` | — | `Name string` | ✅ |
| `repo_url` | — | `RepoUrl *string` | ✅ |
| `prompt` | — | `Prompt *string` | ✅ |
| `project_id` | — | — | **ADD** — FK → projects, multi-tenant scoping |

#### Task

| Field | Backend | API Server (Postgres) | Gap |
|-------|---------|----------------------|-----|
| `name` | — | `Name string` | ✅ |
| `repo_url` | — | `RepoUrl *string` | ✅ |
| `prompt` | — | `Prompt *string` | ✅ |
| `project_id` | — | — | **ADD** — FK → projects, multi-tenant scoping |

#### WorkflowSkill, WorkflowTask

Join tables. No field gaps — complete as designed. Inherit project scoping through their parent Workflow.

### 2.2 Kinds That Exist Only in Backend (must add to API server)

| Kind | Backend Storage | Backend Endpoints | Fields | Priority | Status |
|------|----------------|-------------------|--------|----------|--------|
| **Project** | K8s Namespace | 5 CRUD endpoints | `name`, `display_name`, `description`, `labels`, `annotations`, `status` | **HIGH** | **DONE** |
| **ProjectSettings** | K8s CRD (singleton) | Implicit via permissions/secrets endpoints | `project_id`, `group_access` (jsonb), `repositories` (jsonb) | **HIGH** | **DONE** |
| **Permission** | K8s RoleBindings | 3 endpoints (GET/POST/DELETE) | `subject_type`, `subject_name`, `role`, `project_id` | **MEDIUM** | **DONE** |
| **RepositoryRef** | None (bookmark) | 5 CRUD endpoints | `name`, `url`, `branch`, `provider`, `owner`, `repo_name`, `project_id` | **MEDIUM** | **DONE** |

### 2.3 Backend Capabilities with No Direct Kind (functional endpoints)

These are **not Kinds** in the CRUD sense but are operational endpoints the backend provides:

| Capability Group | Backend Endpoints | Implementation Strategy |
|-----------------|-------------------|------------------------|
| **AG-UI Protocol** (6 endpoints) | `POST .../agui/run`, `POST .../agui/interrupt`, `POST .../agui/feedback`, `GET .../agui/events` (SSE), `GET .../agui/history`, `GET .../agui/runs` | **Not a simple proxy.** Backend implements: (a) SSE fan-out with per-run and per-thread subscriber management, (b) `MessageCompactor` that replays append-only event logs into conversation history (handles streaming deltas, tool call lifecycle, dual casing), (c) compact-on-read for reconnection (completed runs → MESSAGES_SNAPSHOT, active runs → raw replay), (d) background goroutine consuming runner SSE with exponential backoff retry (15 retries, 500ms→5s), (e) 15s keepalive pings. Event persistence currently JSONL on disk — must move to Postgres `agui_events` table. Runner service discovery via K8s Service DNS (`session-{name}.{project}.svc.cluster.local:8001`). |
| **Auth Integrations** (17 endpoints) | GitHub App, GitHub PAT, Google OAuth, Jira, GitLab — connect/status/disconnect per provider | Store credentials in Postgres (encrypted). Credentials are **user-scoped** (not project-scoped) — backend uses K8s Secrets keyed by `{provider}-credentials-{sanitizedUserID}` at cluster scope. Migration needs a `user_credentials` table with `user_id` FK, not `project_id`. |
| **Repository Operations** (5 endpoints) | `GET .../repo/tree`, `GET .../repo/blob`, `GET .../repo/branches`, `GET/POST .../repo/seed` | Proxy to git provider APIs. Stateless — no Postgres storage needed. |
| **Permissions** (3 endpoints) | `GET/POST/DELETE .../permissions` | Map to ProjectSettings `group_access`. CRUD on the jsonb array. |
| **Project Keys** (3 endpoints) | `GET/POST/DELETE .../keys` | New Kind or sub-resource of Project. API keys for SDK auth. |
| **Secrets Management** (5 endpoints) | namespace secrets, runner secrets, integration secrets | **OUT OF SCOPE** — secrets remain in K8s Secrets API, managed by existing backend. Never stored in Postgres. |
| **OOTB Workflows** (1 endpoint) | `GET /api/workflows/ootb` | Static config or dedicated table. |
| **MCP Status** (1 endpoint) | `GET .../mcp/status` | Proxy to running pod. |
| **OAuth Callbacks** (2 endpoints) | `/oauth2callback`, `/oauth2callback/status` | Implement in API server. **Operational note**: callback URIs are registered with external providers (GitHub, Google). Changing the callback URL requires re-registering OAuth apps — this is an operational migration, not just code. Backend uses HMAC-SHA256 signed state params (5-min expiry) to prevent CSRF, with a single callback endpoint dispatching to session-scoped or cluster-scoped flows based on state contents. |
| **Session Workspace** (4 endpoints) | File CRUD in running pod workspace | Proxy to running pod via K8s exec or content-service sidecar. |
| **Session Git Ops** (3 endpoints) | Git status/remote/branches in running pod | Proxy to running pod. |
| **Session K8s Resources** (1 endpoint) | List K8s resources created by session | Query K8s API with session label selector. |
| **Content Service** (7 endpoints) | File/git operations via sidecar | **Not API routes to relocate.** Backend runs as a sidecar binary inside runner pods via `CONTENT_SERVICE_MODE=true` — a reduced Gin server (no K8s client, no auth handlers) serving file CRUD + git ops within the pod's workspace. Path traversal prevention via `pathutil.IsPathWithinBase()`. The API server does not need this mode — it remains a sidecar concern. API server only needs to _proxy_ to the sidecar's HTTP endpoints on the runner pod. |

---

## 3. Data Classification: Postgres vs. Kubernetes

> **Principle**: Postgres owns all data. Session is the only Kind that produces a Kubernetes CRD — because it drives runtime (spawns Jobs, Pods, PVCs, Secrets). Everything else is data.

| Kind | Classification | Lives in Postgres | Produces K8s Resources | Why |
|------|---------------|:-:|:-:|-----|
| **Session** | Runtime | ✅ source of truth | ✅ thin AgenticSession CR | Drives Job/Pod lifecycle. OwnerReferences cascade cleanup. |
| **Project** | Data | ✅ | ✅ Namespace (reconciler creates) | Metadata wrapper. Namespace is a side-effect. |
| **ProjectSettings** | Data | ✅ | ✅ RoleBindings (reconciler creates) | Config that drives RBAC. Settings themselves are data. |
| Agent | Data | ✅ | ❌ | Definition. Resolved at session creation. |
| Workflow | Data | ✅ | ❌ | Composition template. Resolved to git coordinates when building Session CR. |
| Skill | Data | ✅ | ❌ | Prompt template. Passed to runner via config. |
| Task | Data | ✅ | ❌ | Prompt template. Passed to runner via config. |
| User | Data | ✅ | ❌ | Identity record. Cross-project scope. |
| WorkflowSkill | Data | ✅ | ❌ | Join table. Pure relational. |
| WorkflowTask | Data | ✅ | ❌ | Join table. Pure relational. |

---

## 4. Target Schema: All Kinds in Postgres

### Base columns (every table via `api.Meta`)

| Column | Type | Notes |
|--------|------|-------|
| `id` | text, PK | KSUID via `api.NewID()` |
| `created_at` | timestamptz | Immutable |
| `updated_at` | timestamptz | gorm-maintained |
| `deleted_at` | timestamptz, nullable | Soft delete index |

### sessions (~30 fields, fully implemented)

**Data fields (API-mutable, set at creation or via PATCH):**

| Column | Type | Notes | Gap Status |
|--------|------|-------|------------|
| `name` | text, not null | Display name. Maps to CRD `spec.displayName`. | ✅ exists |
| `prompt` | text | Initial prompt. Maps to CRD `spec.initialPrompt`. | ✅ exists |
| `repo_url` | text | **DEPRECATE** — replaced by `repos` jsonb | ✅ exists, deprecate |
| `repos` | jsonb | `[{url, branch, auto_push}]` — multi-repo support | ✅ done |
| `interactive` | boolean, default true | Chat vs batch mode | ✅ done |
| `timeout` | int, default 300 | Session timeout in seconds | ✅ done |
| `llm_model` | text | LLM model selection. Default: "sonnet". | ✅ done |
| `llm_temperature` | float, default 0.7 | | ✅ done |
| `llm_max_tokens` | int, default 4000 | | ✅ done |
| `workflow_id` | text, FK → workflows | Resolved to git coordinates when building CR | ✅ exists |
| `created_by_user_id` | text, FK → users | Who created it | ✅ exists |
| `assigned_user_id` | text, FK → users | Who is assigned | ✅ exists |
| `parent_session_id` | text, FK → sessions | Self-referencing for chaining | ✅ done |
| `bot_account_name` | text | Bot account for git operations | ✅ done |
| `resource_overrides` | jsonb | `{cpu, memory, storage_class, priority_class}` | ✅ done |
| `environment_variables` | jsonb | Key-value map passed to runner | ✅ done |
| `labels` | jsonb | K8s label pass-through | ✅ done |
| `annotations` | jsonb | K8s annotation pass-through | ✅ done |
| `project_id` | text, FK → projects | Multi-tenant scoping. Reconciler resolves `project.name` from this FK for CRD `spec.project` field. | ✅ done |

**Runtime status fields (synced from K8s CR by reconciler, read-only via API):**

| Column | Type | Notes | Gap Status |
|--------|------|-------|------------|
| `phase` | text | Pending, Creating, Running, Stopping, Stopped, Completed, Failed | ✅ done |
| `start_time` | timestamptz | When runner started executing | ✅ done |
| `completion_time` | timestamptz | When session reached terminal phase | ✅ done |
| `sdk_session_id` | text | SDK session ID for resume | ✅ done |
| `sdk_restart_count` | int, default 0 | Number of SDK restarts | ✅ done |
| `conditions` | jsonb | `[{type, status, reason, message, last_transition_time}]` | ✅ done |
| `reconciled_repos` | jsonb | `[{url, branch, name, status, cloned_at}]` | ✅ done |
| `reconciled_workflow` | jsonb | `{git_url, branch, path, status, applied_at}` | ✅ done |
| `kube_cr_name` | text | DNS-safe name of the Session CR in K8s | ✅ done |
| `kube_cr_uid` | text | K8s UID after CR creation | ✅ done |
| `kube_namespace` | text | Namespace where CR lives | ✅ done |

### projects (new Kind)

| Column | Type | Notes |
|--------|------|-------|
| `name` | text, unique, not null | DNS-1123 safe. Maps to K8s Namespace name. |
| `display_name` | text | Human-readable |
| `description` | text | Optional |
| `labels` | jsonb | Pass-through to K8s Namespace labels |
| `annotations` | jsonb | Pass-through to K8s Namespace annotations |
| `status` | text | Active, Terminating |

No `project_id` FK on this table (it *is* the project).

### project_settings (new Kind)

| Column | Type | Notes |
|--------|------|-------|
| `project_id` | text, FK → projects, unique | One-to-one with Project |
| `group_access` | jsonb | `[{group_name, role}]` — drives RoleBinding creation |
| ~~`runner_secrets`~~ | ~~jsonb~~ | **REMOVED** — secrets stay in K8s Secrets API, not Postgres |
| `repositories` | jsonb | `[{url, branch, provider}]` — default repos |

### workflows (add 2 fields)

| Column | Type | Notes | Gap Status |
|--------|------|-------|------------|
| `name` | text, not null | | ✅ exists |
| `repo_url` | text | Git repository URL | ✅ exists |
| `prompt` | text | | ✅ exists |
| `agent_id` | text, FK → agents | | ✅ exists |
| `branch` | text | Git branch | **ADD** |
| `path` | text | In-repo file path | **ADD** |
| `project_id` | text, FK → projects | Multi-tenant scoping | **ADD** |

### users (add 1 field)

| Column | Type | Notes | Gap Status |
|--------|------|-------|------------|
| `username` | text, unique, not null | Login identifier | ✅ exists |
| `name` | text, not null | Display name | ✅ exists |
| `groups` | text[] | RBAC groups for project access control | **ADD** |

### agents (add 1 field)

| Column | Type | Notes | Gap Status |
|--------|------|-------|------------|
| `name` | text, not null | | ✅ exists |
| `repo_url` | text | | ✅ exists |
| `prompt` | text | | ✅ exists |
| `project_id` | text, FK → projects | Multi-tenant scoping | **ADD** |

### skills (add 1 field)

| Column | Type | Notes | Gap Status |
|--------|------|-------|------------|
| `name` | text, not null | | ✅ exists |
| `repo_url` | text | | ✅ exists |
| `prompt` | text | | ✅ exists |
| `project_id` | text, FK → projects | Multi-tenant scoping | **ADD** |

### tasks (add 1 field)

| Column | Type | Notes | Gap Status |
|--------|------|-------|------------|
| `name` | text, not null | | ✅ exists |
| `repo_url` | text | | ✅ exists |
| `prompt` | text | | ✅ exists |
| `project_id` | text, FK → projects | Multi-tenant scoping | **ADD** |

### workflow_skills, workflow_tasks

No field gaps. Inherit project scoping through parent Workflow.

---

## 5. Session CR: Thin Projection

The AgenticSession CRD becomes a thin, denormalized projection. The reconciler resolves all FKs and writes a self-contained CR.

```
Postgres Session row
  + JOIN workflows → spec.activeWorkflow{gitUrl, branch, path}
  + JOIN users     → spec.userContext{userId, displayName, groups}
  + pass-through   → spec.repos, spec.resourceOverrides, spec.environmentVariables
  ═══════════════════════════════════════════════════════
  → AgenticSession CR (operator can act on it without DB access)
```

**What flows back** (CR status → Postgres):
- `status.phase` → `sessions.phase`
- `status.startTime` → `sessions.start_time`
- `status.completionTime` → `sessions.completion_time`
- `status.sdkSessionId` → `sessions.sdk_session_id`
- `status.sdkRestartCount` → `sessions.sdk_restart_count`
- `status.conditions` → `sessions.conditions`
- `status.reconciledRepos` → `sessions.reconciled_repos`
- `status.reconciledWorkflow` → `sessions.reconciled_workflow`

---

## 6. Naming Conventions

The reconciler is the translation layer between Postgres (snake_case REST) and K8s (camelCase).

| Postgres / API | K8s CRD | Who Maps |
|---------------|---------|----------|
| `sessions.name` | `spec.displayName` | Reconciler |
| `sessions.prompt` | `spec.initialPrompt` | Reconciler |
| `workflows.repo_url` | `activeWorkflow.gitUrl` | Reconciler |
| `users.username` | `userContext.userId` | Reconciler |
| `users.name` | `userContext.displayName` | Reconciler |
| `snake_case` timestamps | `camelCase` timestamps | Reconciler |

No other component needs to care about the mapping.

---

## 7. Pagination

| Aspect | Current API Server | Current Backend | Target |
|--------|-------------------|----------------|--------|
| Parameters | `page` + `size` + `search` (TSL) | `limit` + `offset` + `continue` + `search` | Keep `page` + `size` + `search` (framework-native) |
| Response | `{kind, page, size, total, items}` | `{items, totalCount, limit, offset, hasMore}` | Keep `{kind, page, size, total, items}` |
| Default page size | 100 (framework) | 20 | 100 |
| Max page size | 65500 (Postgres WHERE IN limit) | 100 | 65500 |
| `total` meaning | Total matching records across all pages (COUNT query) | `totalCount` same | Total across all pages |
| `size` in response | Actual items returned in this page | — | Actual items in page |

The SDK wraps pagination into iterators. The internal format doesn't leak.

---

## 8. Endpoint Migration Roadmap

### Phase 1: Core Data (expand existing Kinds)

| Priority | What | Backend Endpoints Replaced |
|----------|------|-----------------------------|
| P0 | Add missing Session fields (schema expansion) | — |
| P0 | Add Project Kind | 5 CRUD endpoints |
| P0 | Add ProjectSettings Kind | — (implicit via permissions) |
| P1 | Add Workflow `branch`, `path`, `project_id` fields | — |
| P1 | Add User `groups` field | — |
| P1 | Add `project_id` to Agent, Skill, Task | — |
| P1 | Session start/stop/clone actions | 3 endpoints |

### Phase 2: Operational Endpoints

| Priority | What | Backend Endpoints Replaced |
|----------|------|-----------------------------|
| P1 | Permissions (CRUD on ProjectSettings.group_access) | 3 endpoints |
| ~~P1~~ | ~~Secrets management~~ | ~~5 endpoints~~ — **REMOVED**: secrets stay in K8s, managed by existing backend |
| P1 | Project Keys (API key management) | 3 endpoints |
| P2 | Auth integrations (GitHub, GitLab, Google, Jira) | 17 endpoints |
| P2 | OOTB Workflows listing | 1 endpoint |

### Phase 3: Runtime Proxies

These don't store data in Postgres — they proxy to running pods or git providers.

| Priority | What | Backend Endpoints Replaced |
|----------|------|-----------------------------|
| P2 | AG-UI protocol (SSE proxy to pods) | 6 endpoints |
| P2 | Repository operations (git provider proxy) | 5 endpoints |
| P2 | Session workspace (file ops in pods) | 4 endpoints |
| P2 | Session git operations (git in pods) | 3 endpoints |
| P3 | Runtime credentials refresh | 4 endpoints |
| P3 | Session K8s resources view | 1 endpoint |
| P3 | Session export | 1 endpoint |
| P3 | MCP status | 1 endpoint |
| P3 | Content service operations | 7 endpoints |

### Phase 4: OpenAPI + SDK Alignment

| Priority | What |
|----------|------|
| P1 | OpenAPI spec updated with all expanded schemas |
| P1 | SDK regenerated from updated OpenAPI spec |
| P1 | Public-API gateway removed — SDK talks directly to API server |
| P2 | SSE endpoint for session state streaming (replaces polling) |

---

## 9. CRDs to Keep / Remove

| CRD | Current State | Action |
|-----|--------------|--------|
| `agenticsessions.vteam.ambient-code` | Active | **Keep** — only CRD. Reconciler writes it from Postgres. |
| `projectsettings.vteam.ambient-code` | Active, singleton per namespace | **Remove** — reconciler creates RoleBindings from Postgres directly. |
| `rfeworkflows.vteam.ambient-code` | Referenced in docs, code commented out | **Already removed** — clean up stale references. |

---

## 10. OpenAPI as Canonical Contract

The **OpenAPI spec** (`ambient-api-server/openapi/openapi.yaml`) is the single source of truth. All consumers derive from it:

```
OpenAPI spec (openapi.yaml)
    ├── ambient-api-server: implements the REST API (spec lives here)
    ├── SDK (Go): wraps OpenAPI into idiomatic Go client
    ├── SDK (Python): wraps OpenAPI into idiomatic Python client
    ├── control-plane: consumes the REST API
    ├── frontend: consumes the REST API
    └── public-api: REMOVED (SDK talks directly to API server)
```

**OpenAPI spec update needed** — add schemas for all new/expanded Kinds:
- Expand `Session` schema with all data + status fields from Section 4
- Add `Project`, `ProjectSettings` schemas
- Add `project_id` to Agent, Skill, Task, Workflow schemas
- Add `branch`, `path` to Workflow schema
- Add `groups` to User schema
- Add SSE event types for real-time streaming

The SDK generates or hand-writes language-friendly wrappers around these OpenAPI types. The spec is authoritative.
