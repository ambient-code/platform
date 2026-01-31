# Claude Code Runner

The Claude Code Runner is a Python-based component that wraps the Claude Code SDK to provide agentic coding capabilities within the Ambient platform.

## Architecture

The runner consists of several key components:

- **`adapter.py`** - Core adapter that wraps the Claude Code SDK and produces AG-UI protocol events
- **`main.py`** - FastAPI server that handles run requests via SSE (Server-Sent Events)
- **`observability.py`** - Langfuse integration for tracking usage and performance
- **`security_utils.py`** - Security utilities for sanitizing secrets and timeouts
- **`context.py`** - Runner context for session and workspace management

## System Prompt Configuration

The Claude Code Runner uses a hybrid system prompt approach that combines:

1. **Base Claude Code Prompt** - The built-in `claude_code` system prompt from the Claude Agent SDK
2. **Workspace Context** - Custom workspace-specific information appended to the base prompt

### Implementation

In `adapter.py` (lines 557-561), the system prompt uses `SystemPromptPreset` format:

```python
system_prompt_config = {
    "type": "preset",
    "preset": "claude_code",
    "append": workspace_prompt,
}
```

This configuration ensures that:
- Claude receives the standard Claude Code instructions and capabilities
- Additional workspace context is provided, including:
  - Repository structure and locations
  - Active workflow information
  - Artifacts and file upload locations
  - Git branch and push instructions for auto-push repos
  - MCP integration setup instructions
  - Workflow-specific instructions from `ambient.json`

### Workspace Context

The workspace context prompt is built by `_build_workspace_context_prompt()` (lines 1500-1575) and includes:

- **Working Directory**: Current workflow or repository location
- **Artifacts Path**: Where to create output files
- **Uploaded Files**: Files uploaded by the user
- **Repositories**: List of repos available in the session
- **Working Branch**: Feature branch for all repos (e.g., `ambient/<session-id>`)
- **Git Push Instructions**: Auto-push configuration for specific repos
- **MCP Integrations**: Instructions for setting up Google Drive and Jira access
- **Workflow Instructions**: Custom system prompt from workflow's `ambient.json`

## Environment Variables

### Authentication

- `ANTHROPIC_API_KEY` - Anthropic API key for Claude access
- `CLAUDE_CODE_USE_VERTEX` - Set to `1` to use Vertex AI instead of Anthropic API
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to GCP service account key (for Vertex AI)
- `ANTHROPIC_VERTEX_PROJECT_ID` - GCP project ID (for Vertex AI)
- `CLOUD_ML_REGION` - GCP region (for Vertex AI)

### Model Configuration

- `LLM_MODEL` - Model to use (e.g., `claude-sonnet-4-5`)
- `LLM_MAX_TOKENS` / `MAX_TOKENS` - Maximum tokens per response
- `LLM_TEMPERATURE` / `TEMPERATURE` - Temperature for sampling

### Session Configuration

- `AGENTIC_SESSION_NAME` - Session name/ID
- `AGENTIC_SESSION_NAMESPACE` - K8s namespace for the session
- `IS_RESUME` - Set to `true` when resuming a session
- `INITIAL_PROMPT` - Initial user prompt

### Repository Configuration

- `REPOS_JSON` - JSON array of repository configurations
  ```json
  [
    {
      "url": "https://github.com/owner/repo",
      "name": "repo-name",
      "branch": "ambient/session-id",
      "autoPush": true
    }
  ]
  ```
- `MAIN_REPO_NAME` - Name of the main repository (CWD)
- `MAIN_REPO_INDEX` - Index of main repo (if name not specified)

### Workflow Configuration

- `ACTIVE_WORKFLOW_GIT_URL` - URL of active workflow repository

### Observability

- `LANGFUSE_PUBLIC_KEY` - Langfuse public key
- `LANGFUSE_SECRET_KEY` - Langfuse secret key
- `LANGFUSE_HOST` - Langfuse host URL

### Backend Integration

- `BACKEND_API_URL` - URL of the backend API
- `PROJECT_NAME` - Project name
- `BOT_TOKEN` - Authentication token for backend API calls
- `USER_ID` - User ID for observability
- `USER_NAME` - User name for observability

### MCP Configuration

- `MCP_CONFIG_FILE` - Path to MCP servers config (default: `/app/claude-runner/.mcp.json`)

### Google Workspace Integration

- `USER_GOOGLE_EMAIL` - User's Google email for authentication (set by operator)
- `GOOGLE_MCP_CREDENTIALS_DIR` - Path to Google OAuth credentials directory (default: `/workspace/.google_workspace_mcp/credentials`)

## MCP Servers

The runner supports MCP (Model Context Protocol) servers for extending Claude's capabilities:

- **webfetch** - Fetch content from URLs
- **mcp-atlassian** - Jira integration for issue management
- **google-workspace** - Google Drive integration for file access
- **session** - Session control tools (restart_session)

MCP servers are configured in `.mcp.json` and loaded at runtime.

### Google Workspace MCP Server

The google-workspace MCP server (workspace-mcp) requires OAuth credentials for Google Drive access:

**Credentials Flow:**
1. User authenticates via OAuth in the frontend (`/integrations` page)
2. Backend stores credentials in a cluster-level K8s secret
3. Operator creates a session-specific secret with the user's credentials
4. Secret is mounted read-only at `/app/.google_workspace_mcp/credentials/`
5. A postStart lifecycle hook copies credentials to writable `/workspace/.google_workspace_mcp/credentials/`
6. The MCP server uses the writable path for token refresh

**Credentials Format** (flat JSON structure):
```json
{
  "token": "<access_token>",
  "refresh_token": "<refresh_token>",
  "token_uri": "https://oauth2.googleapis.com/token",
  "client_id": "<oauth_client_id>",
  "client_secret": "<oauth_client_secret>",
  "scopes": ["https://www.googleapis.com/auth/drive.readonly"],
  "expiry": "2026-01-30T12:00:00"
}
```

**Why writable credentials directory?**
The workspace-mcp server refreshes OAuth tokens automatically when they expire. This requires writing updated tokens to the credentials file. K8s secrets are mounted read-only, so the postStart hook copies credentials to a writable location.

## Development

### Running Tests

```bash
# Install dependencies
uv pip install -e ".[dev]"

# Run all tests
pytest -v

# Run with coverage
pytest --cov=. --cov-report=term-missing
```

See [tests/README.md](tests/README.md) for detailed testing documentation.

### Local Development

```bash
# Install in development mode
uv pip install -e .

# Run the server
python main.py
```

## API

The runner exposes a FastAPI server with the following endpoints:

- `POST /run` - Execute a run with Claude Code SDK
  - Request: `RunAgentInput` (thread_id, run_id, messages)
  - Response: Server-Sent Events (SSE) stream of AG-UI protocol events

- `POST /interrupt` - Interrupt the active execution

- `GET /health` - Health check endpoint

## AG-UI Protocol Events

The runner emits AG-UI protocol events via SSE:

- `RUN_STARTED` - Run has started
- `TEXT_MESSAGE_START` - Message started (user or assistant)
- `TEXT_MESSAGE_CONTENT` - Message content chunk
- `TEXT_MESSAGE_END` - Message completed
- `TOOL_CALL_START` - Tool invocation started
- `TOOL_CALL_ARGS` - Tool arguments
- `TOOL_CALL_END` - Tool invocation completed
- `STEP_STARTED` - Processing step started
- `STEP_FINISHED` - Processing step completed
- `STATE_DELTA` - State update (e.g., result payload)
- `RAW` - Custom events (thinking blocks, system logs, etc.)
- `RUN_FINISHED` - Run completed
- `RUN_ERROR` - Error occurred

## Workspace Structure

The runner operates within a workspace at `/workspace/` with the following structure:

```
/workspace/
├── repos/                # Cloned repositories
│   └── {repo-name}/     # Individual repository
├── workflows/            # Workflow repositories
│   └── {workflow-name}/ # Individual workflow
├── artifacts/            # Output files created by Claude
├── file-uploads/         # User-uploaded files
└── .google_workspace_mcp/ # Google OAuth credentials (writable copy)
    └── credentials/
        └── credentials.json

/app/
└── .claude/              # Claude SDK state (conversation history)
```

### Directory Permissions

The runner container runs as uid=1001 (non-root). The init container (`hydrate.sh`) sets up directory permissions:

- `/workspace/artifacts`, `/workspace/file-uploads`, `/workspace/repos` - **777 permissions**
  - Required because MCP servers spawn as subprocesses and need write access to their working directory
  - `chown 1001:0` is attempted first but may fail on SELinux-restricted hosts
- `/app/.claude` - **777 permissions**
  - Claude SDK requires write access for conversation state
- `/workspace/.google_workspace_mcp/credentials/` - **777 permissions**
  - Created by postStart hook for writable OAuth token storage

## Security

The runner implements several security measures:

- **Secret Sanitization**: API keys and tokens are redacted from logs
- **Timeout Protection**: Operations have configurable timeouts
- **User Context Validation**: User IDs and names are sanitized
- **Read-only Workflow Directories**: Workflows are read-only, outputs go to artifacts
- **OAuth Credentials Isolation**: Google credentials are stored in session-specific secrets, copied to writable storage only within the container

See `security_utils.py` for implementation details.

## Recent Changes

### System Prompt Configuration (2026-01-29)

Changed the system prompt configuration to use the `SystemPromptPreset` format:

**Before (incorrect - caused SDK initialization error):**
```python
system_prompt_config = [
    "claude_code",
    {"type": "text", "text": workspace_prompt}
]
```

**After (correct SystemPromptPreset format):**
```python
system_prompt_config = {
    "type": "preset",
    "preset": "claude_code",
    "append": workspace_prompt,
}
```

**Rationale:**
- `ClaudeAgentOptions.system_prompt` expects `str | SystemPromptPreset | None`
- The list format was invalid and caused `'list' object has no attribute 'get'` error
- `SystemPromptPreset` uses `type="preset"`, `preset="claude_code"`, and optional `append`
- This leverages the built-in Claude Code system prompt with appended workspace context

**Impact:**
- Fixes SDK initialization failure
- Claude receives comprehensive instructions from both sources
- Better alignment with Claude Agent SDK type definitions

**Files Changed:**
- `components/runners/claude-code-runner/adapter.py` (lines 557-561)
