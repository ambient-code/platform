# Session Environment Variables Reference

This document is the authoritative reference for all environment variables injected into session pods by the Ambient Code Platform. A session pod contains four containers — **init-hydrate**, **ambient-content**, **ambient-code-runner**, and **state-sync** — each receiving a subset of these variables.

For development-focused documentation on the runner, see the [Claude Code Runner README](../../components/runners/claude-code-runner/README.md).

---

## Session Identity

Variables that identify the session within the cluster.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `SESSION_ID` | Yes | — | Operator | Session name used by the runner (same as CR `.metadata.name`) |
| `AGENTIC_SESSION_NAME` | Yes | — | Operator | CR name, used for status updates and credential fetching |
| `AGENTIC_SESSION_NAMESPACE` | Yes | — | Operator | Kubernetes namespace the session runs in |
| `PROJECT_NAME` | Yes | — | Operator | Same as namespace; used by the runner for backend API calls |
| `SESSION_NAME` | Yes | — | Operator | Session name passed to init-hydrate and state-sync for S3 paths |
| `NAMESPACE` | Yes | — | Operator | Namespace passed to init-hydrate and state-sync for S3 paths |
| `IS_RESUME` | No | _(unset)_ | Operator | Set to `true` when the session has a prior `status.startTime`; tells the runner to skip `INITIAL_PROMPT` and use `continue_conversation` |
| `PARENT_SESSION_ID` | No | _(unset)_ | Annotation | Set when this session continues a previous one; read from annotation `vteam.ambient-code/parent-session-id`, with fallback to `spec.environmentVariables` |

---

## LLM Configuration

Controls how the Claude model is invoked.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `INITIAL_PROMPT` | Yes | — | CR spec | The task prompt sent to Claude on first run (from `spec.initialPrompt`) |
| `LLM_MODEL` | Yes | — | CR spec | Claude model identifier from `spec.llmSettings.model` (e.g., `claude-sonnet-4-5`, `claude-opus-4-6`) |
| `LLM_TEMPERATURE` | No | `0.00` | CR spec | Sampling temperature from `spec.llmSettings.temperature` (formatted as `%.2f`) |
| `LLM_MAX_TOKENS` | No | `0` | CR spec | Maximum output tokens from `spec.llmSettings.maxTokens` |
| `TIMEOUT` | Yes | — | CR spec | Session timeout in seconds; typically set via `ProjectSettings.defaultTimeout` |
| `INITIAL_PROMPT_DELAY_SECONDS` | No | `1` | Runner default | Delay before sending initial prompt in non-interactive mode; not injected by operator |

---

## Repository Configuration

Defines which repositories the session operates on.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `REPOS_JSON` | No | _(unset)_ | CR spec | JSON array of repository objects with `input` (url, branch, ref) and optional `output` (fork, targetBranch) configuration |
| `MAIN_REPO_NAME` | No | _(unset)_ | CR spec | Name of the primary repository (from `spec.mainRepoName`) |
| `MAIN_REPO_INDEX` | No | `0` | CR spec | Index into `REPOS_JSON` indicating the Claude working directory |

**`REPOS_JSON` schema example:**

```json
[
  {
    "input": {
      "url": "https://github.com/org/repo",
      "branch": "main"
    },
    "output": {
      "targetBranch": "feature-branch"
    }
  }
]
```

---

## Workflow Configuration

Active workflow metadata for structured agent workflows.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `ACTIVE_WORKFLOW_GIT_URL` | No | _(unset)_ | CR spec | Git URL of the workflow repository |
| `ACTIVE_WORKFLOW_BRANCH` | No | _(unset)_ | CR spec | Branch containing the workflow definition |
| `ACTIVE_WORKFLOW_PATH` | No | _(unset)_ | CR spec | Path within the repository to the workflow file |

---

## Authentication & Credentials

Tokens and API keys for external service access.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `ANTHROPIC_API_KEY` | Conditional | — | Secret (`ambient-runner-secrets`) | Anthropic API key; injected via `envFrom` only when Vertex AI is **disabled** |
| `BOT_TOKEN` | Yes | — | Secret (`ambient-runner-token-{name}`) | Kubernetes ServiceAccount token for CR status updates and backend API calls |
| `GITHUB_TOKEN` | No | _(unset)_ | Secret (`ambient-non-vertex-integrations`) / Runtime | GitHub PAT or App token; also refreshed at runtime from backend API |
| `GITLAB_TOKEN` | No | _(unset)_ | Secret (`ambient-non-vertex-integrations`) / Runtime | GitLab access token; also refreshed at runtime from backend API |

!!! note
    `GITHUB_TOKEN` and `GITLAB_TOKEN` are injected from the integration secret at pod start, but the runner **also fetches fresh tokens at runtime** from the backend API before each SDK run. The runtime-fetched values take precedence.

---

## Vertex AI Configuration

Required only when using Google Cloud Vertex AI instead of the Anthropic API. Mutually exclusive with `ANTHROPIC_API_KEY`.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `CLAUDE_CODE_USE_VERTEX` | Yes | `0` | Operator | Set to `1` to enable Vertex AI; set to `0` when disabled |
| `CLOUD_ML_REGION` | Conditional | — | Operator env | Google Cloud region (e.g., `us-east5`); required when Vertex enabled |
| `ANTHROPIC_VERTEX_PROJECT_ID` | Conditional | — | Operator env | Google Cloud project ID; required when Vertex enabled |
| `GOOGLE_APPLICATION_CREDENTIALS` | Conditional | — | Operator env | Path to service account key JSON file; required when Vertex enabled |

---

## Backend Integration

Variables for runner-to-backend communication.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `BACKEND_API_URL` | Yes | — | Operator | Full URL to the backend API; templated as `http://backend-service.{backendNamespace}.svc.cluster.local:8080/api` |
| `USER_ID` | No | _(unset)_ | CR spec | User identifier from `spec.userContext.userId`; used in Langfuse traces and audit logs |
| `USER_NAME` | No | _(unset)_ | CR spec | Display name from `spec.userContext.displayName`; used in logs |

---

## AG-UI Server

Configuration for the AG-UI protocol server running inside the runner container.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `USE_AGUI` | Yes | `true` | Hardcoded | Enables AG-UI server mode (always `true` for current architecture) |
| `AGUI_PORT` | Yes | `8001` | Hardcoded | Port the AG-UI FastAPI server listens on; must match the container port and Service definition |
| `AGUI_HOST` | No | `0.0.0.0` | Runner default | Bind address for the AG-UI server; not injected by operator |

---

## Langfuse Observability

Platform-wide LLM observability configuration. All keys are sourced from the `ambient-admin-langfuse-secret` Secret and are optional — missing keys will not prevent pod startup.

For setup and deployment details, see [Observability & Langfuse](../observability-langfuse.md).

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `LANGFUSE_ENABLED` | No | _(unset)_ | Secret (`ambient-admin-langfuse-secret`) | Set to `true` to enable Langfuse tracing |
| `LANGFUSE_PUBLIC_KEY` | No | _(unset)_ | Secret (`ambient-admin-langfuse-secret`) | Langfuse project public key |
| `LANGFUSE_SECRET_KEY` | No | _(unset)_ | Secret (`ambient-admin-langfuse-secret`) | Langfuse project secret key |
| `LANGFUSE_HOST` | No | _(unset)_ | Secret (`ambient-admin-langfuse-secret`) | Langfuse server URL (e.g., `http://langfuse-web.langfuse.svc.cluster.local:3000`) |
| `LANGFUSE_MASK_MESSAGES` | No | `true` | Runner default | When `true` (default), redacts user prompts and assistant responses in traces; set to `false` only in dev/testing |
| `LANGFUSE_FLUSH_TIMEOUT` | No | `30.0` | Runner default | Timeout in seconds for flushing Langfuse events at session end |

!!! warning
    Setting `LANGFUSE_MASK_MESSAGES=false` exposes full user message content in Langfuse. Only use this in development or testing environments.

---

## S3 State Persistence

Configuration for the init-hydrate and state-sync containers that persist session state (`.claude/`, `artifacts/`, `uploads/`) to S3-compatible storage.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `S3_ENDPOINT` | No | _(empty)_ | Operator | S3-compatible endpoint URL; empty disables S3 operations |
| `S3_BUCKET` | No | _(empty)_ | Operator | S3 bucket name for session state |
| `AWS_ACCESS_KEY_ID` | No | _(empty)_ | Operator | S3 access key |
| `AWS_SECRET_ACCESS_KEY` | No | _(empty)_ | Operator | S3 secret key |
| `SYNC_INTERVAL` | No | `60` | Hardcoded | Seconds between state-sync cycles |
| `MAX_SYNC_SIZE` | No | `1073741824` | Hardcoded | Maximum sync payload size in bytes (default: 1 GB) |

---

## MCP & Google Workspace Integration

Configuration for Model Context Protocol servers and Google Workspace access.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `MCP_CONFIG_FILE` | No | _(unset)_ | Operator env | Path to custom MCP configuration file (e.g., minimal webfetch-only config for e2e tests) |
| `GOOGLE_MCP_CREDENTIALS_DIR` | No | `/workspace/.google_workspace_mcp/credentials` | Hardcoded | Directory where Google OAuth credentials are stored for workspace-mcp |
| `GOOGLE_OAUTH_CLIENT_ID` | No | _(unset)_ | Operator env | Google OAuth client ID for workspace-mcp |
| `GOOGLE_OAUTH_CLIENT_SECRET` | No | _(unset)_ | Operator env | Google OAuth client secret for workspace-mcp |

---

## Jira Integration

Jira credentials are fetched at runtime from the backend API and set as environment variables by the runner.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `JIRA_URL` | No | _(unset)_ | Runtime | Jira instance URL (e.g., `https://issues.redhat.com`) |
| `JIRA_API_TOKEN` | No | _(unset)_ | Runtime | Jira API token for authentication |
| `JIRA_EMAIL` | No | _(unset)_ | Runtime | Email associated with the Jira API token |

---

## Runtime & Debug

General runtime configuration for the runner container.

| Variable | Required | Default | Source | Description |
|----------|----------|---------|--------|-------------|
| `DEBUG` | No | `true` | Hardcoded | Enables debug-level logging in the runner |
| `INTERACTIVE` | Yes | `false` | CR spec | `true` for interactive chat sessions, `false` for batch mode |
| `WORKSPACE_PATH` | Yes | `/workspace` | Hardcoded | Root directory for cloned repositories and artifacts |
| `ARTIFACTS_DIR` | No | `artifacts` | Hardcoded | Subdirectory under workspace for session artifacts |
| `CONTENT_SERVICE_MODE` | Yes | `true` | Hardcoded | Enables content service mode in the ambient-content container |
| `STATE_BASE_DIR` | Yes | `/workspace` | Hardcoded | Base directory for the content service state |

---

## Runtime-Derived Variables

These variables are **not injected by the operator**. They are set by the runner at startup based on credentials fetched from the backend API.

| Variable | Set By | Description |
|----------|--------|-------------|
| `GIT_USER_NAME` | `auth.py` | Git committer name, derived from GitHub/GitLab credentials; falls back to `Ambient Code Bot` |
| `GIT_USER_EMAIL` | `auth.py` | Git committer email, derived from GitHub/GitLab credentials; falls back to `bot@ambient-code.local` |
| `USER_GOOGLE_EMAIL` | `auth.py` | Google Workspace email from OAuth credentials; used by workspace-mcp for impersonation |

---

## Custom Variables

The AgenticSession CR supports injecting arbitrary environment variables via `spec.environmentVariables`. These are applied **last** and can override any base variable.

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: my-session
spec:
  prompt: "..."
  environmentVariables:
    MY_CUSTOM_VAR: "custom-value"
    DEBUG: "false"  # Overrides the default "true"
```

!!! note
    Variables injected via `secretKeyRef` (like `BOT_TOKEN` and `LANGFUSE_*`) cannot be overridden this way — Kubernetes gives `ValueFrom` precedence over `Value`. Only plain-value variables (like `DEBUG`, `BACKEND_API_URL`, `LLM_MODEL`) can be overridden.

---

## Secrets Reference

Kubernetes Secrets that provide environment variables to session pods.

| Secret Name | Scope | Variables | Injection Method |
|-------------|-------|-----------|------------------|
| `ambient-runner-secrets` | Per-project | `ANTHROPIC_API_KEY` | `envFrom` on runner (Vertex disabled only) |
| `ambient-non-vertex-integrations` | Per-project | `GITHUB_TOKEN`, `GITLAB_TOKEN`, custom keys | `envFrom` on runner and ambient-content |
| `ambient-admin-langfuse-secret` | Platform-wide | `LANGFUSE_ENABLED`, `LANGFUSE_PUBLIC_KEY`, `LANGFUSE_SECRET_KEY`, `LANGFUSE_HOST` | `secretKeyRef` on runner (all keys optional) |
| `ambient-runner-token-{name}` | Per-session | `BOT_TOKEN` (key: `k8s-token`) | `secretKeyRef` on runner and init-hydrate |

!!! note
    The `ambient-admin-langfuse-secret` is copied from the operator namespace to the session namespace at pod creation time. All its keys use `optional: true` to prevent pod startup failures if individual keys are missing.

---

## Container Cross-Reference

Shows which containers receive each variable. Containers: **I** = init-hydrate, **C** = ambient-content, **R** = runner, **S** = state-sync.

### Operator-Injected Variables

These are set by the operator when constructing the pod spec.

| Variable | I | C | R | S | Notes |
|----------|---|---|---|---|-------|
| `SESSION_NAME` | x | | | x | S3 path key |
| `NAMESPACE` | x | | | x | S3 path key |
| `SESSION_ID` | | | x | | Same value as `SESSION_NAME` |
| `AGENTIC_SESSION_NAME` | | | x | | CR name |
| `AGENTIC_SESSION_NAMESPACE` | | | x | | CR namespace |
| `PROJECT_NAME` | | | x | | Backend API calls |
| `IS_RESUME` | | | x | | Conditional |
| `PARENT_SESSION_ID` | | | x | | Conditional |
| `INITIAL_PROMPT` | | | x | | |
| `LLM_MODEL` | | | x | | |
| `LLM_TEMPERATURE` | | | x | | |
| `LLM_MAX_TOKENS` | | | x | | |
| `TIMEOUT` | | | x | | |
| `REPOS_JSON` | x | | x | | Injected in both init-hydrate and runner |
| `MAIN_REPO_NAME` | | | x | | |
| `MAIN_REPO_INDEX` | | | x | | |
| `ACTIVE_WORKFLOW_GIT_URL` | x | | x | | Injected in both |
| `ACTIVE_WORKFLOW_BRANCH` | x | | x | | Injected in both |
| `ACTIVE_WORKFLOW_PATH` | x | | x | | Injected in both |
| `ANTHROPIC_API_KEY` | | | x | | Via `envFrom` (Vertex disabled) |
| `BOT_TOKEN` | x | | x | | Via `secretKeyRef` |
| `GITHUB_TOKEN` | | x | x | | Via `envFrom` + runtime refresh |
| `GITLAB_TOKEN` | | x | x | | Via `envFrom` + runtime refresh |
| `CLAUDE_CODE_USE_VERTEX` | | | x | | |
| `CLOUD_ML_REGION` | | | x | | Conditional (Vertex) |
| `ANTHROPIC_VERTEX_PROJECT_ID` | | | x | | Conditional (Vertex) |
| `GOOGLE_APPLICATION_CREDENTIALS` | | | x | | Conditional (Vertex) |
| `BACKEND_API_URL` | | | x | | |
| `USER_ID` | | | x | | Conditional |
| `USER_NAME` | | | x | | Conditional |
| `USE_AGUI` | | | x | | |
| `AGUI_PORT` | | | x | | |
| `LANGFUSE_ENABLED` | | | x | | Via `secretKeyRef` |
| `LANGFUSE_PUBLIC_KEY` | | | x | | Via `secretKeyRef` |
| `LANGFUSE_SECRET_KEY` | | | x | | Via `secretKeyRef` |
| `LANGFUSE_HOST` | | | x | | Via `secretKeyRef` |
| `S3_ENDPOINT` | x | | | x | |
| `S3_BUCKET` | x | | | x | |
| `AWS_ACCESS_KEY_ID` | x | | | x | |
| `AWS_SECRET_ACCESS_KEY` | x | | | x | |
| `SYNC_INTERVAL` | | | | x | |
| `MAX_SYNC_SIZE` | | | | x | |
| `MCP_CONFIG_FILE` | | | x | | Conditional |
| `GOOGLE_MCP_CREDENTIALS_DIR` | | | x | | |
| `GOOGLE_OAUTH_CLIENT_ID` | | | x | | |
| `GOOGLE_OAUTH_CLIENT_SECRET` | | | x | | |
| `DEBUG` | | | x | | |
| `INTERACTIVE` | | | x | | |
| `WORKSPACE_PATH` | | | x | | |
| `ARTIFACTS_DIR` | | | x | | |
| `CONTENT_SERVICE_MODE` | | x | | | |
| `STATE_BASE_DIR` | | x | | | |

### Runner-Default and Runtime-Derived Variables

These are **not injected by the operator**. They are read from the environment by the runner with built-in defaults, or set at runtime by the runner itself. They can be overridden via `spec.environmentVariables`.

| Variable | Container | Notes |
|----------|-----------|-------|
| `AGUI_HOST` | R | Default: `0.0.0.0` |
| `INITIAL_PROMPT_DELAY_SECONDS` | R | Default: `1` |
| `LANGFUSE_MASK_MESSAGES` | R | Default: `true` |
| `LANGFUSE_FLUSH_TIMEOUT` | R | Default: `30.0` |
| `JIRA_URL` | R | Set by `auth.py` from backend API |
| `JIRA_API_TOKEN` | R | Set by `auth.py` from backend API |
| `JIRA_EMAIL` | R | Set by `auth.py` from backend API |
| `GIT_USER_NAME` | R | Set by `auth.py` from GitHub/GitLab credentials |
| `GIT_USER_EMAIL` | R | Set by `auth.py` from GitHub/GitLab credentials |
| `USER_GOOGLE_EMAIL` | R | Set by `auth.py` from Google OAuth credentials |

---

*Source of truth: [`components/operator/internal/handlers/sessions.go`](../../components/operator/internal/handlers/sessions.go) (lines 730–1190) and [`components/runners/claude-code-runner/`](../../components/runners/claude-code-runner/).*
