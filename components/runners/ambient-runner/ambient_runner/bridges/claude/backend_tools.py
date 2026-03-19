"""Backend API tools for Claude Agent SDK.

Provides session management tools as MCP-compatible SDK tools.
"""

import json
import logging
import os
from typing import Any, Callable, Optional

from ambient_runner.tools.backend_api import BackendAPIClient

logger = logging.getLogger(__name__)


def create_backend_mcp_tools(
    sdk_tool_decorator: Callable,
    client: Optional[BackendAPIClient] = None,
) -> list[Any]:
    """Create backend API tools for the Claude Agent SDK.

    Args:
        sdk_tool_decorator: The claude_agent_sdk.tool decorator
        client: Optional BackendAPIClient instance (will create default if not provided)

    Returns:
        List of SDK tool functions
    """
    # Use provided client or create default
    api_client = client or _create_default_client()
    if api_client is None:
        logger.warning(
            "Backend API client not available - backend tools will be skipped"
        )
        return []

    tools = []

    def _tool_response(data: dict) -> dict:
        """Helper to format successful tool response."""
        return {"content": [{"type": "text", "text": json.dumps(data, indent=2)}]}

    def _tool_error(error: Exception) -> dict:
        """Helper to format error tool response."""
        return {
            "content": [
                {
                    "type": "text",
                    "text": json.dumps(
                        {"success": False, "error": str(error)}, indent=2
                    ),
                }
            ],
            "isError": True,
        }

    # Tool 1: List Sessions
    @sdk_tool_decorator(
        "acp_list_sessions",
        (
            "List all active agentic sessions in the current project. "
            "Retrieves running and pending sessions. By default excludes "
            "completed/stopped sessions."
        ),
        {
            "type": "object",
            "properties": {
                "include_completed": {
                    "type": "boolean",
                    "description": "Whether to include stopped/completed sessions",
                }
            },
            "required": [],
        },
    )
    async def acp_list_sessions(args: dict) -> dict:
        """List all active agentic sessions."""
        try:
            include_completed = args.get("include_completed", False)
            sessions = api_client.list_sessions(include_completed=include_completed)
            return _tool_response(
                {
                    "success": True,
                    "sessions": sessions,
                    "count": len(sessions),
                }
            )
        except Exception as e:
            logger.error(f"Error listing sessions: {e}", exc_info=True)
            return _tool_error(e)

    tools.append(acp_list_sessions)

    # Tool 2: Get Session
    @sdk_tool_decorator(
        "acp_get_session",
        (
            "Get detailed information about a specific agentic session. "
            "Retrieves full session object including spec, status, and metadata."
        ),
        {
            "type": "object",
            "properties": {
                "session_name": {
                    "type": "string",
                    "description": "Name of the session to retrieve",
                }
            },
            "required": ["session_name"],
        },
    )
    async def acp_get_session(args: dict) -> dict:
        """Get detailed information about a specific session."""
        try:
            session_name = args["session_name"]
            session = api_client.get_session(session_name)
            return _tool_response({"success": True, "session": session})
        except Exception as e:
            logger.error(f"Error getting session: {e}", exc_info=True)
            return _tool_error(e)

    tools.append(acp_get_session)

    # Tool 3: Create Session
    @sdk_tool_decorator(
        "acp_create_session",
        (
            "Create a new agentic session in the current project. "
            "Creates and starts a new Claude session with the specified configuration."
        ),
        {
            "type": "object",
            "properties": {
                "session_name": {
                    "type": "string",
                    "description": "Unique identifier (DNS-compatible: lowercase, hyphens, no spaces)",
                },
                "initial_prompt": {
                    "type": "string",
                    "description": "Initial message to send to the agent",
                },
                "display_name": {
                    "type": "string",
                    "description": "Human-readable display name",
                },
                "repos": {
                    "type": "string",
                    "description": 'JSON array of repo configs: [{"url": "https://...", "branch": "main"}]',
                },
                "model": {
                    "type": "string",
                    "description": "LLM model override (e.g., claude-sonnet-4-5)",
                },
            },
            "required": ["session_name"],
        },
    )
    async def acp_create_session(args: dict) -> dict:
        """Create a new agentic session."""
        try:
            session_name = args["session_name"]
            initial_prompt = args.get("initial_prompt")
            display_name = args.get("display_name")
            repos_str = args.get("repos")
            model = args.get("model")

            # Parse repos if provided
            repos_list = None
            if repos_str:
                try:
                    repos_list = json.loads(repos_str)
                except json.JSONDecodeError as e:
                    return _tool_error(ValueError(f"Invalid repos JSON: {e}"))

            session = api_client.create_session(
                session_name=session_name,
                initial_prompt=initial_prompt,
                display_name=display_name,
                repos=repos_list,
                model=model,
            )
            return _tool_response(
                {
                    "success": True,
                    "message": f"Session '{session_name}' created successfully",
                    "session": session,
                }
            )
        except Exception as e:
            logger.error(f"Error creating session: {e}", exc_info=True)
            return _tool_error(e)

    tools.append(acp_create_session)

    # Tool 4: Stop Session
    @sdk_tool_decorator(
        "acp_stop_session",
        (
            "Stop a running agentic session. "
            "Gracefully stops the specified session and cleans up resources."
        ),
        {
            "type": "object",
            "properties": {
                "session_name": {
                    "type": "string",
                    "description": "Name of the session to stop",
                }
            },
            "required": ["session_name"],
        },
    )
    async def acp_stop_session(args: dict) -> dict:
        """Stop a running agentic session."""
        try:
            session_name = args["session_name"]
            api_result = api_client.stop_session(session_name)
            return _tool_response(
                {
                    "success": True,
                    "message": f"Session '{session_name}' stop initiated",
                    "result": api_result,
                }
            )
        except Exception as e:
            logger.error(f"Error stopping session: {e}", exc_info=True)
            return _tool_error(e)

    tools.append(acp_stop_session)

    # Tool 5: Send Message
    @sdk_tool_decorator(
        "acp_send_message",
        (
            "Send a message to an agentic session. "
            "Sends a user message to the specified session, triggering a new agent run. "
            "This is asynchronous - the agent will process the message in the background."
        ),
        {
            "type": "object",
            "properties": {
                "session_name": {
                    "type": "string",
                    "description": "Name of the session to send message to",
                },
                "message": {
                    "type": "string",
                    "description": "Message content to send",
                },
                "thread_id": {
                    "type": "string",
                    "description": "Optional thread ID for multi-threaded sessions",
                },
            },
            "required": ["session_name", "message"],
        },
    )
    async def acp_send_message(args: dict) -> dict:
        """Send a message to an agentic session."""
        try:
            session_name = args["session_name"]
            message = args["message"]
            thread_id = args.get("thread_id")

            api_result = api_client.send_message(
                session_name=session_name,
                message=message,
                thread_id=thread_id,
            )
            return _tool_response(
                {
                    "success": True,
                    "message": f"Message sent to session '{session_name}'",
                    "run": api_result,
                }
            )
        except Exception as e:
            logger.error(f"Error sending message: {e}", exc_info=True)
            return _tool_error(e)

    tools.append(acp_send_message)

    # Tool 6: Get API Reference
    @sdk_tool_decorator(
        "acp_get_api_reference",
        (
            "Get comprehensive API reference documentation for the Ambient Code Platform backend. "
            "Returns detailed information about available REST API endpoints, authentication, "
            "request/response formats, and code examples. Use this when you need to generate "
            "code that calls the ACP backend API directly, create HTML dashboards, scripts, "
            "or integrations."
        ),
        {
            "type": "object",
            "properties": {},
            "required": [],
        },
    )
    async def acp_get_api_reference(args: dict) -> dict:
        """Get comprehensive API reference documentation."""
        backend_url = os.getenv("BACKEND_API_URL", "http://backend:8080/api")
        project_name = os.getenv("PROJECT_NAME", "your-project-name")
        bot_token = os.getenv("BOT_TOKEN", "your-bot-token")

        docs = f"""# Ambient Code Platform API Reference

## Base Configuration

**Base URL**: `{backend_url}`
**Project Name**: `{project_name}`
**Authentication**: Bearer token in `Authorization` header

```javascript
const BASE_URL = '{backend_url}';
const PROJECT_NAME = '{project_name}';
const BOT_TOKEN = '{bot_token}';  // From environment or config

const headers = {{
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${{BOT_TOKEN}}`
}};
```

---

## Endpoints

### 1. List Sessions

**GET** `/api/projects/{{projectName}}/agentic-sessions`

List all agentic sessions in the project.

**Response**: `200 OK`
```json
{{
  "sessions": [
    {{
      "name": "session-abc123",
      "displayName": "My Session",
      "phase": "Running",
      "spec": {{
        "initialPrompt": "Build a feature",
        "repos": [...],
        "llmSettings": {{"model": "claude-sonnet-4-5"}}
      }},
      "status": {{...}}
    }}
  ]
}}
```

**Example**:
```javascript
const response = await fetch(
    `${{BASE_URL}}/projects/${{PROJECT_NAME}}/agentic-sessions`,
    {{ headers }}
);
const data = await response.json();
console.log(`Found ${{data.sessions.length}} sessions`);
```

---

### 2. Get Session

**GET** `/api/projects/{{projectName}}/agentic-sessions/{{sessionName}}`

Get detailed information about a specific session.

**Response**: `200 OK`
```json
{{
  "name": "session-abc123",
  "displayName": "My Session",
  "phase": "Running",
  "spec": {{...}},
  "status": {{
    "podName": "session-abc123-pod",
    "conditions": [...]
  }}
}}
```

**Example**:
```javascript
const sessionName = 'my-session';
const response = await fetch(
    `${{BASE_URL}}/projects/${{PROJECT_NAME}}/agentic-sessions/${{sessionName}}`,
    {{ headers }}
);
const session = await response.json();
console.log(`Session ${{session.name}} is ${{session.phase}}`);
```

---

### 3. Create Session

**POST** `/api/projects/{{projectName}}/agentic-sessions`

Create and start a new agentic session.

**Request Body**:
```json
{{
  "sessionName": "my-new-session",
  "displayName": "My New Session",
  "initialPrompt": "Build a dashboard for metrics",
  "repos": [
    {{
      "url": "https://github.com/org/repo",
      "branch": "main"
    }}
  ],
  "llmSettings": {{
    "model": "claude-sonnet-4-5"
  }}
}}
```

**Required Fields**:
- `sessionName` (string): DNS-compatible name (lowercase, hyphens, no spaces)

**Optional Fields**:
- `displayName` (string): Human-readable name
- `initialPrompt` (string): Initial message to send to the agent
- `repos` (array): Repository configurations
- `llmSettings.model` (string): LLM model override

**Response**: `201 Created`
```json
{{
  "name": "my-new-session",
  "phase": "Pending",
  "spec": {{...}}
}}
```

**Example**:
```javascript
const sessionData = {{
    sessionName: 'task-' + Date.now(),
    displayName: 'Automated Task',
    initialPrompt: 'Fix the bug in auth.py',
    repos: [{{
        url: 'https://github.com/myorg/myrepo',
        branch: 'main'
    }}]
}};

const response = await fetch(
    `${{BASE_URL}}/projects/${{PROJECT_NAME}}/agentic-sessions`,
    {{
        method: 'POST',
        headers,
        body: JSON.stringify(sessionData)
    }}
);
const created = await response.json();
console.log(`Created session: ${{created.name}}`);
```

**HTML Button Example**:
```html
<button onclick="createSession()">Create Session</button>

<script>
async function createSession() {{
    const response = await fetch(
        '{backend_url}/projects/{project_name}/agentic-sessions',
        {{
            method: 'POST',
            headers: {{
                'Content-Type': 'application/json',
                'Authorization': 'Bearer {bot_token}'
            }},
            body: JSON.stringify({{
                sessionName: 'web-session-' + Date.now(),
                displayName: 'Web Dashboard Session',
                initialPrompt: 'Analyze the codebase'
            }})
        }}
    );
    const session = await response.json();
    alert('Created: ' + session.name);
}}
</script>
```

---

### 4. Stop Session

**POST** `/api/projects/{{projectName}}/agentic-sessions/{{sessionName}}/stop`

Stop a running session.

**Request Body**: Empty or `{{}}`

**Response**: `200 OK`
```json
{{
  "status": "ok"
}}
```

**Example**:
```javascript
const sessionName = 'my-session';
const response = await fetch(
    `${{BASE_URL}}/projects/${{PROJECT_NAME}}/agentic-sessions/${{sessionName}}/stop`,
    {{
        method: 'POST',
        headers
    }}
);
console.log('Session stopped');
```

---

### 5. Send Message (AG-UI Run)

**POST** `/api/projects/{{projectName}}/agentic-sessions/{{sessionName}}/agui/run`

Send a message to a running session (asynchronous).

**Request Body**:
```json
{{
  "messages": [
    {{
      "role": "user",
      "content": "Please review the recent changes"
    }}
  ],
  "threadId": "optional-thread-id"
}}
```

**Response**: `200 OK`
```json
{{
  "runId": "run-xyz789",
  "threadId": "thread-abc123"
}}
```

**Example**:
```javascript
const response = await fetch(
    `${{BASE_URL}}/projects/${{PROJECT_NAME}}/agentic-sessions/${{sessionName}}/agui/run`,
    {{
        method: 'POST',
        headers,
        body: JSON.stringify({{
            messages: [{{
                role: 'user',
                content: 'What is the current status?'
            }}]
        }})
    }}
);
const run = await response.json();
console.log(`Message sent, runId: ${{run.runId}}`);
```

---

## Authentication Patterns

### Using Environment Variables
```javascript
// Server-side (Node.js, Python, etc.)
const BOT_TOKEN = process.env.BOT_TOKEN;

fetch(url, {{
    headers: {{
        'Authorization': `Bearer ${{BOT_TOKEN}}`
    }}
}});
```

### Frontend with Proxy
```javascript
// Frontend calls your backend proxy (avoids exposing token)
fetch('/api/proxy/sessions', {{
    method: 'POST',
    body: JSON.stringify(sessionData)
}});

// Your backend proxy adds authentication
app.post('/api/proxy/sessions', async (req, res) => {{
    const response = await fetch(
        `${{BACKEND_URL}}/projects/${{PROJECT_NAME}}/agentic-sessions`,
        {{
            method: 'POST',
            headers: {{
                'Authorization': `Bearer ${{process.env.BOT_TOKEN}}`,
                'Content-Type': 'application/json'
            }},
            body: JSON.stringify(req.body)
        }}
    );
    const data = await response.json();
    res.json(data);
}});
```

---

## Error Handling

**404 Not Found**:
```json
{{
  "error": "Session not found"
}}
```

**400 Bad Request**:
```json
{{
  "error": "Invalid session name format"
}}
```

**401 Unauthorized**:
```json
{{
  "error": "Invalid or missing authentication token"
}}
```

**Example with error handling**:
```javascript
try {{
    const response = await fetch(url, options);
    if (!response.ok) {{
        const error = await response.json();
        throw new Error(`API error: ${{error.error}}`);
    }}
    const data = await response.json();
    return data;
}} catch (err) {{
    console.error('Request failed:', err.message);
}}
```

---

## Common Integration Patterns

### Dashboard with Session List
```html
<!DOCTYPE html>
<html>
<body>
    <h1>ACP Sessions Dashboard</h1>
    <button onclick="loadSessions()">Refresh</button>
    <button onclick="createNewSession()">Create Session</button>
    <div id="sessions"></div>

    <script>
    const API = {{
        baseUrl: '{backend_url}',
        project: '{project_name}',
        token: '{bot_token}',

        async request(path, options = {{}}) {{
            const response = await fetch(
                `${{this.baseUrl}}${{path}}`,
                {{
                    ...options,
                    headers: {{
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${{this.token}}`,
                        ...options.headers
                    }}
                }}
            );
            return response.json();
        }}
    }};

    async function loadSessions() {{
        const data = await API.request(
            `/projects/${{API.project}}/agentic-sessions`
        );

        document.getElementById('sessions').innerHTML =
            data.sessions.map(s => `
                <div>
                    <strong>${{s.displayName || s.name}}</strong>
                    [${{s.phase}}]
                    <button onclick="stopSession('${{s.name}}')">Stop</button>
                </div>
            `).join('');
    }}

    async function createNewSession() {{
        const name = 'session-' + Date.now();
        await API.request(
            `/projects/${{API.project}}/agentic-sessions`,
            {{
                method: 'POST',
                body: JSON.stringify({{
                    sessionName: name,
                    displayName: 'Dashboard Session',
                    initialPrompt: 'Hello!'
                }})
            }}
        );
        loadSessions();
    }}

    async function stopSession(name) {{
        await API.request(
            `/projects/${{API.project}}/agentic-sessions/${{name}}/stop`,
            {{ method: 'POST' }}
        );
        loadSessions();
    }}

    loadSessions();
    </script>
</body>
</html>
```

### Python Script
```python
import os
import requests

BASE_URL = os.getenv('BACKEND_API_URL', '{backend_url}')
PROJECT = os.getenv('PROJECT_NAME', '{project_name}')
TOKEN = os.getenv('BOT_TOKEN', '{bot_token}')

headers = {{
    'Content-Type': 'application/json',
    'Authorization': f'Bearer {{TOKEN}}'
}}

def list_sessions():
    url = f'{{BASE_URL}}/projects/{{PROJECT}}/agentic-sessions'
    resp = requests.get(url, headers=headers)
    return resp.json()['sessions']

def create_session(name, prompt):
    url = f'{{BASE_URL}}/projects/{{PROJECT}}/agentic-sessions'
    data = {{
        'sessionName': name,
        'initialPrompt': prompt
    }}
    resp = requests.post(url, json=data, headers=headers)
    return resp.json()

# Usage
sessions = list_sessions()
print(f'Found {{len(sessions)}} sessions')

new_session = create_session('script-session', 'Analyze the code')
print(f'Created: {{new_session["name"]}}')
```

---

## Session Name Requirements

Session names must be DNS-compatible:
- Lowercase letters (a-z)
- Numbers (0-9)
- Hyphens (-)
- No spaces, underscores, or special characters
- Maximum 63 characters

**Valid**: `my-session-123`, `build-feature-v2`, `task-2024-03-19`
**Invalid**: `My Session`, `task_123`, `session@prod`

---

## Phase Values

Sessions progress through these phases:
- `Pending`: Created, waiting to start
- `Running`: Actively executing
- `Stopped`: Gracefully stopped
- `Failed`: Error occurred
- `Completed`: Successfully finished

---

## Current Configuration

- **Backend URL**: `{backend_url}`
- **Project**: `{project_name}`
- **Auth Token**: {("Set" if bot_token and bot_token != "your-bot-token" else "Not set (use BOT_TOKEN env var)")}

Use these values when generating code or making API calls from this session.
"""
        return {"content": [{"type": "text", "text": docs}]}

    tools.append(acp_get_api_reference)

    return tools


def _create_default_client() -> Optional[BackendAPIClient]:
    """Create a default BackendAPIClient from environment variables.

    Returns:
        BackendAPIClient instance, or None if required env vars are missing
    """
    backend_url = os.getenv("BACKEND_API_URL", "").strip()
    project_name = (
        os.getenv("PROJECT_NAME") or os.getenv("AGENTIC_SESSION_NAMESPACE", "")
    ).strip()

    if not backend_url or not project_name:
        logger.debug(
            "Backend API client cannot be created: "
            "BACKEND_API_URL or PROJECT_NAME not set"
        )
        return None

    try:
        return BackendAPIClient(
            backend_url=backend_url,
            project_name=project_name,
        )
    except ValueError as e:
        logger.warning(f"Failed to create backend API client: {e}")
        return None
