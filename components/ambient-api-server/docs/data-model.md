# Data Model

## Base Model (api.Meta)

All resources inherit from `api.Meta` (from `rh-trex-ai/pkg/api`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string (KSUID) | Auto-generated via `BeforeCreate` hook |
| `created_at` | time.Time | Auto-managed by GORM |
| `updated_at` | time.Time | Auto-managed by GORM |
| `deleted_at` | *time.Time | Soft delete (GORM `DeletedAt`) |

## Resource Types

### Agent

Defines an AI agent that can execute workflows.

| Field | Go Type | DB Column | Nullable | Description |
|-------|---------|-----------|----------|-------------|
| name | string | name | No | Agent display name |
| repo_url | *string | repo_url | Yes | Source repository URL |
| prompt | *string | prompt | Yes | System prompt / description |

### Skill

A capability that can be attached to workflows.

| Field | Go Type | DB Column | Nullable | Description |
|-------|---------|-----------|----------|-------------|
| name | string | name | No | Skill display name |
| repo_url | *string | repo_url | Yes | Source repository URL |
| prompt | *string | prompt | Yes | Skill description / instructions |

### Task

A unit of work within a workflow.

| Field | Go Type | DB Column | Nullable | Description |
|-------|---------|-----------|----------|-------------|
| name | string | name | No | Task display name |
| repo_url | *string | repo_url | Yes | Source repository URL |
| prompt | *string | prompt | Yes | Task instructions |

### Workflow

Orchestrates an agent with skills to perform tasks.

| Field | Go Type | DB Column | Nullable | Description |
|-------|---------|-----------|----------|-------------|
| name | string | name | No | Workflow display name |
| repo_url | *string | repo_url | Yes | Source repository URL |
| prompt | *string | prompt | Yes | Workflow description |
| agent_id | *string | agent_id | Yes | FK → agents.id |

### WorkflowSkill (Join Table)

Associates skills to workflows with ordering.

| Field | Go Type | DB Column | Nullable | Description |
|-------|---------|-----------|----------|-------------|
| workflow_id | string | workflow_id | No | FK → workflows.id |
| skill_id | string | skill_id | No | FK → skills.id |
| position | int | position | No | Order within workflow |

### WorkflowTask (Join Table)

Associates tasks to workflows with ordering.

| Field | Go Type | DB Column | Nullable | Description |
|-------|---------|-----------|----------|-------------|
| workflow_id | string | workflow_id | No | FK → workflows.id |
| task_id | string | task_id | No | FK → tasks.id |
| position | int | position | No | Order within workflow |

### Session

An execution instance of a workflow.

| Field | Go Type | DB Column | Nullable | Description |
|-------|---------|-----------|----------|-------------|
| name | string | name | No | Session display name |
| repo_url | *string | repo_url | Yes | Target repository URL |
| prompt | *string | prompt | Yes | User prompt for the session |
| created_by_user_id | *string | created_by_user_id | Yes | FK → users.id (creator) |
| assigned_user_id | *string | assigned_user_id | Yes | FK → users.id (assignee) |
| workflow_id | *string | workflow_id | Yes | FK → workflows.id |

### User

Platform user identity.

| Field | Go Type | DB Column | Nullable | Description |
|-------|---------|-----------|----------|-------------|
| username | string | username | No | Unique username |
| name | string | name | No | Display name |

## Relationship Diagram

```
User ──created_by──→ Session ←──workflow──→ Workflow ←──agent──→ Agent
User ──assigned_to─→ Session                  │
                                              ├──→ WorkflowSkill ──→ Skill
                                              └──→ WorkflowTask  ──→ Task
```

## Workflow Pattern: "AS agent WITH skills DO tasks"

The domain model encodes this pattern:
1. **AS** → `Workflow.agent_id` links to an Agent
2. **WITH** → `WorkflowSkill` join table links Skills (ordered by `position`)
3. **DO** → `WorkflowTask` join table links Tasks (ordered by `position`)
4. **Session** instantiates a Workflow for execution

## Database Conventions

- **Table names**: Plural lowercase (e.g., `agents`, `workflow_skills`)
- **Soft deletes**: All tables use `deleted_at` column
- **IDs**: KSUIDs (sortable, globally unique) — generated in `BeforeCreate` GORM hook
- **Migrations**: One per Kind in `plugins/{kinds}/migration.go`, timestamp-based IDs
- **Advisory locks**: Used in `Replace()` operations to prevent concurrent update conflicts
