"""Custom tools for backend API operations.

These tools are registered with the Claude Agent SDK to provide
session management capabilities until the MCP server is ready.
"""

import json
import logging
from typing import Any, Dict, Optional

from ambient_runner.tools.backend_api import BackendAPIClient

logger = logging.getLogger(__name__)


def create_backend_tools() -> Dict[str, Dict[str, Any]]:
    """Create tool definitions for backend API operations.

    Returns:
        Dictionary mapping tool names to MCP-style tool definitions
    """
    return {
        "acp_list_sessions": {
            "name": "acp_list_sessions",
            "description": """List all active agentic sessions in the current project.

This tool retrieves all running and pending sessions. By default, it excludes completed/stopped sessions.

Use this to:
- See what sessions are currently running
- Get session names and IDs for use with other tools
- Check session status and metadata

Returns a list of session objects with fields like:
- name: Session identifier (use this with other tools)
- displayName: Human-readable name
- phase: Current status (Running, Pending, Stopped, etc.)
- spec: Session configuration (model, repos, etc.)
""",
            "input_schema": {
                "type": "object",
                "properties": {
                    "include_completed": {
                        "type": "boolean",
                        "description": "Whether to include stopped/completed sessions (default: false)",
                        "default": False,
                    }
                },
                "required": [],
            },
        },
        "acp_get_session": {
            "name": "acp_get_session",
            "description": """Get detailed information about a specific agentic session.

Retrieves the full session object including spec, status, and metadata.

Use this to:
- Get detailed info about a specific session
- Check session configuration (model, repos, prompts)
- See session status and phase

Args:
- session_name: The name of the session (from list_sessions or known identifier)

Returns the full session object with all fields.
""",
            "input_schema": {
                "type": "object",
                "properties": {
                    "session_name": {
                        "type": "string",
                        "description": "Name of the session to retrieve",
                    }
                },
                "required": ["session_name"],
            },
        },
        "acp_create_session": {
            "name": "acp_create_session",
            "description": """Create a new agentic session in the current project.

Creates and starts a new Claude session with the specified configuration.

Use this to:
- Spawn a new agent session for a task
- Start a session with specific repos and prompts
- Create sessions with custom model settings

Args:
- session_name: Unique identifier (DNS-compatible: lowercase, hyphens, no spaces)
- initial_prompt: (optional) Initial message to send to the agent
- display_name: (optional) Human-readable name
- repos: (optional) List of repo configs: [{"url": "https://...", "branch": "main"}]
- model: (optional) LLM model override (e.g., "claude-sonnet-4-5")

Returns the created session object.

Example:
```
{
  "session_name": "my-task-session",
  "initial_prompt": "Review the recent PRs",
  "display_name": "PR Review Session",
  "repos": [{"url": "https://github.com/org/repo", "branch": "main"}]
}
```
""",
            "input_schema": {
                "type": "object",
                "properties": {
                    "session_name": {
                        "type": "string",
                        "description": "Unique session name (lowercase, hyphens only, must be DNS-compatible)",
                    },
                    "initial_prompt": {
                        "type": "string",
                        "description": "Optional initial prompt to send to the agent",
                    },
                    "display_name": {
                        "type": "string",
                        "description": "Optional human-readable display name",
                    },
                    "repos": {
                        "type": "array",
                        "description": "Optional list of repository configurations",
                        "items": {
                            "type": "object",
                            "properties": {
                                "url": {"type": "string"},
                                "branch": {"type": "string"},
                            },
                            "required": ["url"],
                        },
                    },
                    "model": {
                        "type": "string",
                        "description": "Optional LLM model override",
                    },
                },
                "required": ["session_name"],
            },
        },
        "acp_stop_session": {
            "name": "acp_stop_session",
            "description": """Stop a running agentic session.

Gracefully stops the specified session, cleaning up resources.

Use this to:
- Stop a session that has completed its work
- Clean up idle or stuck sessions
- Free resources

Args:
- session_name: Name of the session to stop

Returns confirmation of the stop operation.
""",
            "input_schema": {
                "type": "object",
                "properties": {
                    "session_name": {
                        "type": "string",
                        "description": "Name of the session to stop",
                    }
                },
                "required": ["session_name"],
            },
        },
        "acp_send_message": {
            "name": "acp_send_message",
            "description": """Send a message to an agentic session.

Sends a user message to the specified session, triggering a new agent run.

Use this to:
- Send commands or questions to a running session
- Continue a conversation with an agent
- Provide feedback or additional context

Args:
- session_name: Name of the target session
- message: Message content to send
- thread_id: (optional) Thread ID for multi-threaded sessions

Returns run metadata (runId, threadId) for tracking the message.

Note: This is asynchronous - the agent will process the message in the background.
To see the response, you would need to monitor events (via frontend or logs).
""",
            "input_schema": {
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
        },
    }


class BackendToolExecutor:
    """Executor for backend API tools."""

    def __init__(self, client: Optional[BackendAPIClient] = None):
        """Initialize the tool executor.

        Args:
            client: Optional BackendAPIClient instance (will create default if not provided)
        """
        self.client = client or BackendAPIClient()

    def execute(self, tool_name: str, tool_input: Dict[str, Any]) -> str:
        """Execute a backend tool.

        Args:
            tool_name: Name of the tool to execute
            tool_input: Input parameters for the tool

        Returns:
            JSON string with the tool result
        """
        try:
            if tool_name == "acp_list_sessions":
                include_completed = tool_input.get("include_completed", False)
                sessions = self.client.list_sessions(
                    include_completed=include_completed
                )
                return json.dumps(
                    {
                        "success": True,
                        "sessions": sessions,
                        "count": len(sessions),
                    },
                    indent=2,
                )

            elif tool_name == "acp_get_session":
                session_name = tool_input["session_name"]
                session = self.client.get_session(session_name)
                return json.dumps({"success": True, "session": session}, indent=2)

            elif tool_name == "acp_create_session":
                session_name = tool_input["session_name"]
                initial_prompt = tool_input.get("initial_prompt")
                display_name = tool_input.get("display_name")
                repos = tool_input.get("repos")
                model = tool_input.get("model")

                session = self.client.create_session(
                    session_name=session_name,
                    initial_prompt=initial_prompt,
                    display_name=display_name,
                    repos=repos,
                    model=model,
                )
                return json.dumps(
                    {
                        "success": True,
                        "message": f"Session '{session_name}' created successfully",
                        "session": session,
                    },
                    indent=2,
                )

            elif tool_name == "acp_stop_session":
                session_name = tool_input["session_name"]
                result = self.client.stop_session(session_name)
                return json.dumps(
                    {
                        "success": True,
                        "message": f"Session '{session_name}' stop initiated",
                        "result": result,
                    },
                    indent=2,
                )

            elif tool_name == "acp_send_message":
                session_name = tool_input["session_name"]
                message = tool_input["message"]
                thread_id = tool_input.get("thread_id")

                result = self.client.send_message(
                    session_name=session_name,
                    message=message,
                    thread_id=thread_id,
                )
                return json.dumps(
                    {
                        "success": True,
                        "message": f"Message sent to session '{session_name}'",
                        "run": result,
                    },
                    indent=2,
                )

            else:
                return json.dumps(
                    {"success": False, "error": f"Unknown tool: {tool_name}"},
                    indent=2,
                )

        except Exception as e:
            logger.error(f"Error executing tool {tool_name}: {e}", exc_info=True)
            return json.dumps(
                {"success": False, "error": str(e), "tool": tool_name}, indent=2
            )
