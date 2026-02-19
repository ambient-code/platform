# Ambient Platform Python SDK

Simple HTTP client library for the Ambient Code Platform - Create and manage AI agent sessions without Kubernetes complexity.

## Installation

```bash
pip install ambient-platform-sdk
```

## Quick Start

```python
import os
from ambient_platform import AmbientClient, CreateSessionRequest, RepoHTTP

# Create HTTP client
client = AmbientClient(
    base_url="https://your-platform.example.com",
    token=os.getenv("AMBIENT_TOKEN"),      # Bearer token
    project=os.getenv("AMBIENT_PROJECT"),  # Project namespace
)

# Create a session
request = CreateSessionRequest(
    task="Analyze the repository structure and provide a summary",
    model="claude-3.5-sonnet",
    repos=[
        RepoHTTP(
            url="https://github.com/ambient-code/platform",
            branch="main"
        )
    ]
)

response = client.create_session(request)
print(f"Created session: {response.id}")

# Get session details
session = client.get_session(response.id)
print(f"Status: {session.status}")

# List all sessions
sessions = client.list_sessions()
print(f"Found {len(sessions.items)} sessions")

# Close client when done
client.close()
```

## Authentication & Authorization

The SDK uses Bearer token authentication with project-scoped authorization:

### Token Requirements

- **Bearer Token**: Must be a valid authentication token (OpenShift, JWT, or GitHub format)  
- **Project Header**: `X-Ambient-Project` specifies the target Kubernetes namespace
- **RBAC**: User must have appropriate permissions in the target namespace

### Supported Token Formats

- **OpenShift**: `sha256~...` format tokens from `oc whoami -t`
- **JWT**: Standard JSON Web Tokens with 3 base64 parts
- **GitHub**: Tokens starting with `ghp_`, `gho_`, `ghu_`, or `ghs_`

### Required Permissions

Your user account must have these Kubernetes RBAC permissions in the target project/namespace:

```yaml
# Minimum required permissions
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions"] 
  verbs: ["get", "list", "create"]

- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get"]
```

### Common Permission Errors

**403 Forbidden**:
```bash
# Check your permissions
oc auth can-i create agenticsessions.vteam.ambient-code -n your-project
oc auth can-i list agenticsessions.vteam.ambient-code -n your-project  
```

**401 Unauthorized**:
```bash
# Check token validity
oc whoami  # Should return your username
oc whoami -t  # Should return a token starting with sha256~
```

**400 Bad Request - Project required**:
- Ensure `AMBIENT_PROJECT` environment variable is set
- Project must be a valid Kubernetes namespace name  
- User must have access to the specified project

```bash
# Set environment variables
export AMBIENT_TOKEN="your-bearer-token"      # Required
export AMBIENT_PROJECT="your-project-name"    # Required  
export AMBIENT_API_URL="https://your-api.com" # Optional
```

**OpenShift Users:**
```bash
# Use your OpenShift token
export AMBIENT_TOKEN="$(oc whoami -t)"
export AMBIENT_PROJECT="$(oc project -q)"
```

## Core Operations

### Create Session

```python
from ambient_platform import CreateSessionRequest, RepoHTTP

request = CreateSessionRequest(
    task="Review this code for security issues",
    model="claude-3.5-sonnet",  # Optional, uses platform default if omitted
    repos=[
        RepoHTTP(url="https://github.com/user/repo", branch="main")
    ]
)

response = client.create_session(request)
print(f"Session ID: {response.id}")
```

### Get Session Details

```python
from ambient_platform import StatusCompleted

session = client.get_session("session-1234567")
print(f"Status: {session.status}")

if session.status == StatusCompleted:
    print(f"Result: {session.result}")
```

### List Sessions

```python
sessions = client.list_sessions()
for session in sessions.items:
    print(f"- {session.id} ({session.status}): {session.task}")
```

### Monitor Session Completion

```python
# Wait for session to complete (with timeout)
try:
    completed = client.wait_for_completion(
        session_id="session-1234567",
        poll_interval=5.0,  # Check every 5 seconds
        timeout=300.0       # 5 minute timeout
    )
    
    if completed.status == StatusCompleted:
        print(f"Success: {completed.result}")
    else:
        print(f"Failed: {completed.error}")
        
except TimeoutError:
    print("Session monitoring timed out")
```

## Session Status Values

```python
from ambient_platform import StatusPending, StatusRunning, StatusCompleted, StatusFailed

# Status constants
StatusPending   = "pending"   # Session created, waiting to start
StatusRunning   = "running"   # AI agent actively working
StatusCompleted = "completed" # Task finished successfully
StatusFailed    = "failed"    # Task failed with error
```

## Configuration Options

### Environment Variables

Create client from environment variables:

```python
# Automatically reads AMBIENT_API_URL, AMBIENT_TOKEN, AMBIENT_PROJECT
client = AmbientClient.from_env()
```

### Context Manager

Use client as context manager for automatic cleanup:

```python
with AmbientClient.from_env() as client:
    response = client.create_session(request)
    session = client.get_session(response.id)
    # Client automatically closed when exiting context
```

### Custom Timeout

```python
client = AmbientClient(
    base_url="https://api.example.com",
    token="your-token",
    project="your-project",
    timeout=60.0  # 60 second timeout
)
```

## Error Handling

```python
from ambient_platform.exceptions import (
    AmbientAPIError,
    AuthenticationError,
    SessionNotFoundError,
    AmbientConnectionError,
)

try:
    session = client.get_session("invalid-id")
except SessionNotFoundError as e:
    print(f"Session not found: {e}")
except AuthenticationError as e:
    print(f"Auth failed: {e}")
except AmbientConnectionError as e:
    print(f"Connection failed: {e}")
except AmbientAPIError as e:
    print(f"API error: {e}")
```

## Examples

See the `examples/` directory for complete working examples:

- **`main.py`** - Complete session lifecycle demonstration

Run the example:
```bash
cd examples/
export AMBIENT_TOKEN="your-token"
export AMBIENT_PROJECT="your-project"
python main.py
```

## API Reference

### AmbientClient

```python
class AmbientClient:
    def __init__(self, base_url: str, token: str, project: str, timeout: float = 30.0)
    
    def create_session(self, request: CreateSessionRequest) -> CreateSessionResponse
    def get_session(self, session_id: str) -> SessionResponse
    def list_sessions(self) -> SessionListResponse
    def wait_for_completion(self, session_id: str, poll_interval: float = 5.0, timeout: Optional[float] = None) -> SessionResponse
    
    @classmethod
    def from_env(cls, **kwargs) -> "AmbientClient"
    
    def close(self)  # Close HTTP client
```

### Data Classes

```python
@dataclass
class CreateSessionRequest:
    task: str
    model: Optional[str] = None
    repos: Optional[List[RepoHTTP]] = None

@dataclass
class RepoHTTP:
    url: str
    branch: Optional[str] = None

@dataclass
class SessionResponse:
    id: str
    status: str  # "pending", "running", "completed", "failed"
    task: str
    model: Optional[str] = None
    created_at: Optional[str] = None
    completed_at: Optional[str] = None
    result: Optional[str] = None
    error: Optional[str] = None

@dataclass
class SessionListResponse:
    items: List[SessionResponse]
    total: int

@dataclass
class CreateSessionResponse:
    id: str
    message: str

@dataclass
class ErrorResponse:
    error: str
    message: Optional[str] = None
```

## Architecture

### Design Principles

- **HTTP-First**: Pure REST API client with no Kubernetes dependencies
- **Minimal Dependencies**: Uses only `httpx` for HTTP requests
- **Simple Integration**: Easy to embed in any Python application
- **Type Safety**: Dataclasses with type hints for all requests/responses
- **Clear Separation**: Public SDK vs internal platform implementation

### HTTP vs Kubernetes

This SDK provides a **simplified HTTP interface** to the Ambient Platform:

| Aspect | HTTP SDK (This Package) | Internal Platform |
|--------|------------------------|-------------------|
| **API** | Simple REST endpoints (`/v1/sessions`) | Complex Kubernetes CRDs |
| **Auth** | Bearer token + project header | RBAC + service accounts |
| **Types** | Simple dataclasses | Full K8s metadata/spec/status |
| **Usage** | Any HTTP client, any environment | Kubernetes cluster access required |
| **Dependencies** | Only `httpx` | `kubernetes`, `pydantic`, etc. |
| **Target** | External integrators, simple automation | Internal platform components |

## Troubleshooting

### Authentication Issues
```
❌ AMBIENT_TOKEN environment variable is required
```
**Solution**: Set your Bearer token: `export AMBIENT_TOKEN="your-token"`

### Project Header Missing  
```
API error (400): Project required. Set X-Ambient-Project header
```
**Solution**: Set project name: `export AMBIENT_PROJECT="your-project"`

### Connection Errors
```
Failed to connect to API: [connection error details]
```
**Solution**: Verify API endpoint and network connectivity

### Session Not Found
```
session not found: session-xyz
```
**Solution**: Verify session ID and check if you have access to the project

## Development

### Setup Development Environment

```bash
# Clone repository
git clone https://github.com/ambient-code/platform.git
cd platform/components/ambient-sdk/python-sdk

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install in development mode
pip install -e ".[dev]"

# Run tests
pytest

# Format code
black ambient_platform examples
isort ambient_platform examples

# Type checking
mypy ambient_platform
```

### Running Tests

```bash
# Run all tests
pytest

# Run integration tests (requires running API)
pytest -m integration

# Run with coverage
pytest --cov=ambient_platform --cov-report=html
```

### Testing Against Real API

```bash
# Set environment variables
export AMBIENT_TOKEN="your-token"
export AMBIENT_PROJECT="your-project"
export AMBIENT_API_URL="https://your-api.example.com"

# Run example
python examples/main.py

# Run integration tests
pytest -m integration
```

## OpenAPI Specification

This SDK is built to match the canonical OpenAPI specification owned by the API server at `../../ambient-api-server/openapi/openapi.yaml`. The SDK does not maintain its own spec copy — types and client behavior derive from the API server's definitions.

## Terminal Usage Guide

### Quick Setup

```bash
# Navigate to python-sdk directory
cd /path/to/platform/components/ambient-sdk/python-sdk

# Create and activate virtual environment
python -m venv venv
source venv/bin/activate

# Install SDK
pip install -e .

# Set environment variables
export AMBIENT_TOKEN="your-bearer-token"
export AMBIENT_PROJECT="your-project-name"
export AMBIENT_API_URL="https://your-api-endpoint.com"  # Optional

# Run example
python examples/main.py
```

### Interactive Python Session

```bash
# Start Python REPL
python

# Use the SDK interactively
>>> from ambient_platform import AmbientClient, CreateSessionRequest, RepoHTTP
>>> client = AmbientClient.from_env()
>>> request = CreateSessionRequest(task="Hello world", model="claude-3.5-sonnet")
>>> response = client.create_session(request)
>>> print(f"Session ID: {response.id}")
```

## Contributing

1. **SDK Changes**: Modify code in `ambient_platform/` directory
2. **API Changes**: Update `../openapi.yaml` specification first
3. **Examples**: Add working examples to `examples/` directory
4. **Testing**: Ensure all changes work with real API endpoints

For complete platform documentation, see the main [platform repository](https://github.com/ambient-code/platform).