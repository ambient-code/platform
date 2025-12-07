# MCP Server for Ambient Code Platform

A Model Context Protocol (MCP) server that provides read-only access to the Ambient Code Platform backend API. This server runs as a sidecar in Claude runner pods and allows Claude to browse projects, sessions, and workspace files through native MCP tools.

## Features

- **13 Read-Only Tools** organized into 4 categories:
  - **Project Management**: List projects, get project details, check access
  - **Session Browsing**: List and inspect agentic sessions
  - **Workspace Access**: Browse and read files from session workspaces
  - **Cluster Info**: Get cluster and workflow information

- **Secure Authentication**: Uses user bearer tokens for RBAC-enforced API access
- **Error Handling**: Informative error messages with status code mapping
- **Path Validation**: Prevents path traversal attacks on file access

## Architecture

**Deployment**: Sidecar container in Claude runner pods
**Communication**: MCP over stdio
**Authentication**: User bearer token from `BOT_TOKEN` environment variable
**Backend API**: Internal Kubernetes service DNS

## Available Tools

### Project Management

- `list_projects()` - List all accessible projects
- `get_project(project_name)` - Get project details
- `check_project_access(project_name)` - Check user permissions

### Session Browsing

- `list_sessions(project_name)` - List agentic sessions in a project
- `get_session(project_name, session_name)` - Get session details
- `get_session_k8s_resources(project_name, session_name)` - Get K8s resources
- `list_session_workspace(project_name, session_name)` - List workspace files

### Workspace File Access

- `get_workspace_file(project_name, session_name, path)` - Get file contents

### Workflow & Cluster Info

- `list_ootb_workflows()` - List out-of-the-box workflows
- `get_workflow_metadata(project_name, session_name)` - Get workflow metadata
- `get_cluster_info()` - Get cluster information
- `get_health()` - Get backend health status

## Installation

```bash
# Install with uv (preferred)
uv pip install .

# Or with pip
pip install .
```

## Usage

### As MCP Server (in runner pods)

The server is automatically started when included in the runner's `.mcp.json` configuration:

```json
{
  "mcpServers": {
    "mcp-ambient": {
      "command": "mcp-ambient-server",
      "args": [],
      "env": {
        "BACKEND_API_URL": "${BACKEND_API_URL}",
        "BOT_TOKEN": "${BOT_TOKEN}"
      }
    }
  }
}
```

### Manual Testing (with port-forward)

```bash
# Port-forward backend API
kubectl port-forward -n ambient-code svc/vteam-backend 8080:8080

# Export credentials
export BACKEND_API_URL=http://localhost:8080/api
export BOT_TOKEN=$(kubectl get secret -n test-project session-xyz-runner-token -o jsonpath='{.data.token}' | base64 -d)

# Run server
mcp-ambient-server
```

## Configuration

The server requires two environment variables:

- `BACKEND_API_URL` - Backend API base URL (default: `http://vteam-backend.ambient-code.svc.cluster.local:8080/api`)
- `BOT_TOKEN` - User bearer token for authentication (required)

## Development

### Setup

```bash
# Create virtual environment
uv venv
source .venv/bin/activate  # or `. .venv/bin/activate`

# Install dependencies
uv pip install -e ".[dev]"
```

### Running Tests

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=mcp_ambient_server --cov-report=html

# Run specific test file
pytest tests/test_client.py
```

### Linting and Formatting

```bash
# Format code
black src/ tests/

# Run linter
ruff check src/ tests/
```

## Security

- **Read-Only Operations**: All tools use GET requests only
- **Token Security**: Tokens are never logged (only token length)
- **Input Validation**: Project/session names and file paths are validated
- **Path Traversal Protection**: Blocks `..` in file paths
- **RBAC Enforcement**: All API calls respect user permissions

## Error Handling

The server maps HTTP status codes to informative error messages:

- `401` - "Authentication failed. BOT_TOKEN may be invalid or expired."
- `403` - "Access denied. User does not have permission for this resource."
- `404` - "Resource not found"
- `500+` - "Backend API error: {message}"
- Network errors - "Cannot reach backend API. Check cluster connectivity."
- Timeouts - "Request timed out after 30s."

## Integration with Runner

This MCP server is designed to run as a sidecar in Claude runner pods. The operator sets the required environment variables (`BACKEND_API_URL`, `BOT_TOKEN`) automatically when creating runner pods.

## License

Part of the Ambient Code Platform project.
