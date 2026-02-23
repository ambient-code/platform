# UI Model Comparison: Current Frontend vs. New Data Model

> How the existing Ambient Code Platform frontend maps onto the new
> ambient-api-server OpenAPI Kinds, what stays the same, what changes, and what
> gaps exist in the current OpenAPI schemas.

---

## 1. New Data Model Summary

Eight Kinds with a relational Postgres backend. Only Session produces a K8s CRD.

| Kind | Key Fields | Relationships |
|------|-----------|---------------|
| **User** | `username`, `name` | Creates and gets assigned Sessions |
| **Agent** | `name`, `repo_url`, `prompt` | Executes Workflows (via `workflow.agent_id`) |
| **Skill** | `name`, `repo_url`, `prompt` | Attached to Workflows via WorkflowSkill (ordered) |
| **Task** | `name`, `repo_url`, `prompt` | Attached to Workflows via WorkflowTask (ordered) |
| **Workflow** | `name`, `repo_url`, `prompt`, `agent_id` | "AS agent WITH skills DO tasks" |
| **Session** | `name`, `repo_url`, `prompt`, `workflow_id`, `created_by_user_id`, `assigned_user_id` | Runtime execution instance |
| **WorkflowSkill** | `workflow_id`, `skill_id`, `position` | Junction table (ordered) |
| **WorkflowTask** | `workflow_id`, `task_id`, `position` | Junction table (ordered) |

All entities share an `ObjectReference` base: `id` (UUID), `kind`, `href`, `created_at`, `updated_at`.

---

## 2. Current Frontend Architecture

### Pages

| Route | Purpose |
|-------|---------|
| `/projects` | Workspace list (paginated table with search, create, delete) |
| `/projects/[name]` | Workspace detail with tabbed sidebar: Sessions (default), Sharing, Settings |
| `/projects/[name]/sessions/[sessionName]` | Session detail: split-panel with sidebar (workflows, repos, artifacts, MCP, integrations, file explorer) + main chat area (AG-UI streaming) |
| `/projects/[name]/keys` | API key management |
| `/projects/[name]/settings` | Project settings |
| `/projects/[name]/permissions` | RBAC management |
| `/integrations` | Integration hub (GitHub, GitLab, Google, Jira) |

### Data Flow

```
React Component
  -> React Query Hook (useCreateSession, useSessionsPaginated, etc.)
  -> API Service (src/services/api/sessions.ts)
  -> NextJS API Route (/api/projects/[name]/agentic-sessions/...)
  -> Go Backend -> K8s CRD (AgenticSession)
```

Chat uses AG-UI protocol over SSE, independent of the data model.

### Current Session Type Structure (K8s CR)

```
AgenticSession {
  metadata: { name, namespace, uid, creationTimestamp, labels, annotations }
  spec: {
    initialPrompt, displayName, interactive, timeout,
    llmSettings: { model, temperature, maxTokens },
    repos: [{ url, branch, autoPush }],
    activeWorkflow: { gitUrl, branch, path }
  }
  status: {
    phase, startTime, completionTime, sdkSessionId, sdkRestartCount,
    conditions, reconciledRepos, reconciledWorkflow
  }
}
```

---

## 3. Field Mapping: Current UI Types -> New API Kinds

### Session

| Current (K8s CR shape) | New (Postgres/REST) | Notes |
|------------------------|--------------------|----|
| `metadata.name` | `id` (UUID) | URL routing changes: `/sessions/{name}` -> `/sessions/{id}` |
| `metadata.namespace` | Derived from `project_id` | Scoping shifts from K8s namespace to Postgres FK |
| `metadata.uid` | `id` | UUID is now the primary identifier |
| `metadata.creationTimestamp` | `created_at` | Same data, snake_case |
| `spec.displayName` | `name` | Direct field, no spec wrapper |
| `spec.initialPrompt` | `prompt` | Direct field |
| `spec.interactive` | `interactive` | **Not in current OpenAPI** (in Postgres schema) |
| `spec.timeout` | `timeout` | **Not in current OpenAPI** (in Postgres schema) |
| `spec.llmSettings.model` | `llm_model` | **Not in current OpenAPI** (in Postgres schema) |
| `spec.llmSettings.temperature` | `llm_temperature` | **Not in current OpenAPI** (in Postgres schema) |
| `spec.llmSettings.maxTokens` | `llm_max_tokens` | **Not in current OpenAPI** (in Postgres schema) |
| `spec.repos[]` | `repos` (jsonb) | **Not in current OpenAPI** (in Postgres schema) |
| `spec.activeWorkflow.{gitUrl,branch,path}` | `workflow_id` FK -> resolved by reconciler | UI fetches workflow details separately |
| `status.phase` | `phase` | **Not in current OpenAPI** (in Postgres schema) |
| `status.startTime` | `start_time` | **Not in current OpenAPI** (in Postgres schema) |
| `status.completionTime` | `completion_time` | **Not in current OpenAPI** (in Postgres schema) |
| `status.sdkSessionId` | `sdk_session_id` | **Not in current OpenAPI** (in Postgres schema) |
| `status.conditions[]` | `conditions` (jsonb) | **Not in current OpenAPI** (in Postgres schema) |
| `status.reconciledRepos[]` | `reconciled_repos` (jsonb) | **Not in current OpenAPI** (in Postgres schema) |
| `status.reconciledWorkflow` | `reconciled_workflow` (jsonb) | **Not in current OpenAPI** (in Postgres schema) |
| *(new)* | `created_by_user_id` FK | Currently derived from auth context |
| *(new)* | `assigned_user_id` FK | Currently not a concept in the UI |
| *(new)* | `parent_session_id` FK | Currently not a concept (clone is a copy, not a chain) |

### Workflow

| Current | New | Notes |
|---------|-----|-------|
| Static OOTB list from `/api/workflows/ootb` | CRUD resource at `/v1/workflows` | Full lifecycle management |
| `WorkflowSelection.gitUrl` | `repo_url` | Field rename |
| `WorkflowSelection.branch` | `branch` | **Not in current OpenAPI** (in Postgres schema) |
| `WorkflowSelection.path` | `path` | **Not in current OpenAPI** (in Postgres schema) |
| *(not a concept)* | `agent_id` FK | Workflows now explicitly reference an Agent |
| *(not a concept)* | WorkflowSkill junction | Skills composed into workflows with ordering |
| *(not a concept)* | WorkflowTask junction | Tasks composed into workflows with ordering |

### Agent, Skill, Task

No equivalents exist in the current UI. These are entirely new CRUD surfaces.

### User

| Current | New | Notes |
|---------|-----|-------|
| Auth identity from `/api/me` | `User` Kind at `/v1/users` | First-class CRUD resource |
| `X-Forwarded-User` header | `username` field | |
| Display name from auth | `name` field | |
| *(not exposed)* | `groups` | **Not in current OpenAPI** (in Postgres schema) |

---

## 4. What Stays the Same

These UI surfaces are unaffected or require only minor data-fetching changes:

1. **Session detail + chat interface** -- The entire AG-UI streaming system, message rendering, tool call hierarchy, thinking blocks, and chat input are independent of the data model. They talk to the runner, not the API server. The sidebar panels (repos, artifacts, MCP, integrations, file explorer) also stay the same.

2. **Session lifecycle controls** -- Start, stop, continue, clone, delete. The actions are the same; only the API endpoint paths change.

3. **Session list table** -- Columns (name, status, mode, model, created, cost, actions) remain identical. The data source changes from K8s list to Postgres list.

4. **Workspace/project pages** -- The concept of project-scoped views persists. The data source shifts from K8s namespaces to Postgres projects.

5. **Integrations hub** -- GitHub, GitLab, Google, Jira integration setup is orthogonal to the data model.

6. **BFF proxy pattern** -- NextJS API routes proxying to the backend is unchanged. The target backend changes from K8s-backed Go service to Postgres-backed Go service.

---

## 5. What Changes

### 5.1 Session Creation Dialog

**Current**: Minimal form with session name + model picker + integration status cards. Creates an `AgenticSession` CR with hardcoded defaults.

**New**: Should include:
- Session name (same)
- Model picker (same, once `llm_model` is in OpenAPI)
- **Workflow picker** -- select from `/v1/workflows` list. Selecting a workflow auto-resolves the agent, skills, and tasks. This replaces the "welcome experience" workflow cards
- **User assignment** -- optional `assigned_user_id` picker for collaboration
- Integration status (same)

### 5.2 Workflow Management (New CRUD Surface)

**Current**: Workflows are static OOTB definitions fetched once. The session sidebar has a "Workflows" accordion for selecting/activating one.

**New**: Full workflow management:
- **Workflow list page** -- table of workflows within a project
- **Workflow detail/editor** -- the "AS agent WITH skills DO tasks" composer:
  - Agent selector (dropdown from `/v1/agents`)
  - Skills list with drag-and-drop ordering (via WorkflowSkill `position`)
  - Tasks list with drag-and-drop ordering (via WorkflowTask `position`)
  - Workflow-level prompt editor
  - `repo_url` field for git-backed workflow definitions
- **Workflow selection in session** -- the sidebar accordion switches from static OOTB list to dynamic workflow list

### 5.3 Agent Catalog (New CRUD Surface)

**Current**: Agents are implicit -- the runner pod IS the agent.

**New**: Agent management:
- **Agent list** -- table within project settings or as a top-level section
- **Agent detail** -- name, repo_url (documentation/persona repo), prompt (system prompt/persona definition)
- **Agent selection** -- referenced when composing Workflows

### 5.4 Skill Catalog (New CRUD Surface)

**Current**: No concept of standalone skills.

**New**: Skill management:
- **Skill list** -- reusable prompt templates
- **Skill detail** -- name, repo_url (skill definition repo), prompt (the skill template)
- **Skill composition** -- referenced in Workflows via WorkflowSkill with `position` ordering

### 5.5 Task Catalog (New CRUD Surface)

**Current**: No concept of standalone tasks.

**New**: Task management:
- **Task list** -- atomic work unit definitions
- **Task detail** -- name, repo_url (task definition repo), prompt (the task template)
- **Task sequencing** -- referenced in Workflows via WorkflowTask with `position` ordering

### 5.6 Routing and Identification

**Current**: Resources identified by K8s name (DNS-safe string) within a namespace.

**New**: Resources identified by UUID. URL patterns shift:
- `/projects/{name}/sessions/{sessionName}` -> `/projects/{name}/sessions/{id}`
- All new Kinds use UUID-based routes

### 5.7 Pagination Response Shape

**Current frontend**: Expects `{ items, totalCount, limit, offset, hasMore }`.

**Current OpenAPI**: Returns `{ kind, page, size, total, items }`.

**Recommended** (per DATA_MODEL_COMPARISON.md): `{ kind, items, total, limit, offset }`.

The React Query hooks need their pagination params and response parsing adjusted regardless of which standard is chosen.

---

## 6. Gaps in Current OpenAPI Schemas

The OpenAPI specs define minimal schemas. The Postgres schema in `DATA_MODEL_COMPARISON.md` is the complete picture. These fields are in Postgres but **not yet in the OpenAPI**:

### Session (most gaps)

| Missing Field | Type | UI Impact |
|--------------|------|-----------|
| `interactive` | boolean | Create dialog mode toggle |
| `timeout` | int | Session timeout configuration |
| `llm_model` | text | Model picker in create dialog |
| `llm_temperature` | float | Advanced settings |
| `llm_max_tokens` | int | Advanced settings |
| `repos` | jsonb | Repository management sidebar |
| `phase` | text | Status badge everywhere |
| `start_time` | timestamptz | Session timing display |
| `completion_time` | timestamptz | Session timing display |
| `sdk_session_id` | text | Session resume capability |
| `sdk_restart_count` | int | Status display |
| `conditions` | jsonb | Error/warning display |
| `reconciled_repos` | jsonb | Repo status sidebar panel |
| `reconciled_workflow` | jsonb | Active workflow display |
| `parent_session_id` | UUID FK | Session chaining |
| `bot_account_name` | text | Git operations |
| `resource_overrides` | jsonb | Advanced settings |
| `environment_variables` | jsonb | Advanced settings |
| `labels` | jsonb | K8s pass-through |
| `annotations` | jsonb | K8s pass-through |
| `project_id` | UUID FK | Multi-tenant scoping |

### Workflow

| Missing Field | Type | UI Impact |
|--------------|------|-----------|
| `branch` | text | Workflow git source |
| `path` | text | Workflow file path |
| `project_id` | UUID FK | Multi-tenant scoping |

### User

| Missing Field | Type | UI Impact |
|--------------|------|-----------|
| `groups` | text[] | RBAC display |

### All Entities

| Missing Field | Type | UI Impact |
|--------------|------|-----------|
| `project_id` | UUID FK | Project-scoped queries (`?search=project_id='...'`) |

---

## 7. Recommended Frontend Service Layer Changes

### New Service Files Needed

| File | Purpose |
|------|---------|
| `src/services/api/agents.ts` | Agent CRUD: list, get, create, patch |
| `src/services/api/skills.ts` | Skill CRUD: list, get, create, patch |
| `src/services/api/tasks.ts` | Task CRUD: list, get, create, patch |
| `src/services/api/workflows-crud.ts` | Workflow CRUD (replaces static OOTB) |
| `src/services/api/workflow-skills.ts` | WorkflowSkill junction CRUD |
| `src/services/api/workflow-tasks.ts` | WorkflowTask junction CRUD |
| `src/services/api/users-crud.ts` | User CRUD (supplements auth) |

### New React Query Hooks Needed

| File | Key Hooks |
|------|-----------|
| `src/services/queries/use-agents.ts` | `useAgentsPaginated`, `useAgent`, `useCreateAgent`, `useUpdateAgent` |
| `src/services/queries/use-skills.ts` | `useSkillsPaginated`, `useSkill`, `useCreateSkill`, `useUpdateSkill` |
| `src/services/queries/use-tasks.ts` | `useTasksPaginated`, `useTask`, `useCreateTask`, `useUpdateTask` |
| `src/services/queries/use-workflows-crud.ts` | `useWorkflowsPaginated`, `useWorkflow`, `useCreateWorkflow`, `useUpdateWorkflow` |
| `src/services/queries/use-workflow-composition.ts` | `useWorkflowSkills`, `useWorkflowTasks`, add/remove/reorder mutations |

### New TypeScript Types Needed

| File | Types |
|------|-------|
| `src/types/agent.ts` | `Agent`, `AgentList`, `AgentPatchRequest` |
| `src/types/skill.ts` | `Skill`, `SkillList`, `SkillPatchRequest` |
| `src/types/task.ts` | `Task`, `TaskList`, `TaskPatchRequest` |
| `src/types/workflow.ts` | `Workflow`, `WorkflowList`, `WorkflowPatchRequest` (with full fields) |
| `src/types/workflow-composition.ts` | `WorkflowSkill`, `WorkflowTask`, list and patch variants |
| `src/types/api/common.ts` | Updated `ObjectReference`, `List` base types matching new API shape |

### Modified Types

| File | Change |
|------|--------|
| `src/types/agentic-session.ts` | Flatten from `metadata/spec/status` nesting to flat fields. Add `workflow_id`, `created_by_user_id`, `assigned_user_id`, `parent_session_id`. Remove K8s-specific `ObjectMeta`. |
| `src/types/api/sessions.ts` | Update response shapes to match new List format |

---

## 8. Proposed Navigation Structure

```
Workspaces (/)
  └── {workspace} (/projects/{name})
        ├── Sessions (default tab) -- list, create, detail+chat
        ├── Workflows -- list, create, detail (AS/WITH/DO composer)
        ├── Agents -- list, create, detail
        ├── Skills -- list, create, detail
        ├── Tasks -- list, create, detail
        ├── Sharing -- permissions (existing)
        └── Settings -- project settings, keys (existing)
```

Agent, Skill, and Task catalogs could alternatively be **platform-global** (not project-scoped) if they are shared across projects, depending on whether `project_id` is required or nullable for these Kinds.

---

## 9. Migration Path: Incremental Adoption

The transition can be done incrementally without breaking the current UI:

1. **Phase 1 -- Service layer abstraction**: Introduce an adapter in the API service layer that maps new REST responses to the existing K8s CR type shapes. The React components continue using the current types unchanged. All existing tests pass.

2. **Phase 2 -- New CRUD surfaces**: Add Agent, Skill, Task, and Workflow management pages. These are additive -- no existing pages change. The Workflow page replaces the static OOTB endpoint.

3. **Phase 3 -- Session type migration**: Update the `AgenticSession` type from K8s CR nesting (`metadata/spec/status`) to flat REST fields. Update all components that destructure `session.spec.displayName` to use `session.name`, etc.

4. **Phase 4 -- Routing migration**: Switch session URLs from name-based to UUID-based. Add redirects for backward compatibility.

5. **Phase 5 -- Remove K8s type artifacts**: Drop `ObjectMeta`, `spec/status` wrappers, namespace-based scoping from the frontend codebase entirely.
