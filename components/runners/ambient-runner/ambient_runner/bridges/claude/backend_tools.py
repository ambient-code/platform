"""Backend API tools for Claude Agent SDK.

Provides session management tools as MCP-compatible SDK tools.
"""

import logging
from typing import Any, Callable, List, Optional

from ambient_runner.tools.backend_api import BackendAPIClient

logger = logging.getLogger(__name__)


def create_backend_mcp_tools(
    sdk_tool_decorator: Callable,
    client: Optional[BackendAPIClient] = None,
) -> List[Any]:
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

    @sdk_tool_decorator
    def acp_list_sessions(include_completed: bool = False) -> str:
        """List all active agentic sessions in the current project.

        This tool retrieves all running and pending sessions. By default, it excludes completed/stopped sessions.

        Use this to:
        - See what sessions are currently running
        - Get session names and IDs for use with other tools
        - Check session status and metadata

        Args:
            include_completed: Whether to include stopped/completed sessions (default: false)

        Returns:
            JSON string with session list and count
        """
        import json

        try:
            sessions = api_client.list_sessions(include_completed=include_completed)
            return json.dumps(
                {
                    "success": True,
                    "sessions": sessions,
                    "count": len(sessions),
                },
                indent=2,
            )
        except Exception as e:
            logger.error(f"Error listing sessions: {e}", exc_info=True)
            return json.dumps({"success": False, "error": str(e)}, indent=2)

    tools.append(acp_list_sessions)

    @sdk_tool_decorator
    def acp_get_session(session_name: str) -> str:
        """Get detailed information about a specific agentic session.

        Retrieves the full session object including spec, status, and metadata.

        Use this to:
        - Get detailed info about a specific session
        - Check session configuration (model, repos, prompts)
        - See session status and phase

        Args:
            session_name: The name of the session (from list_sessions or known identifier)

        Returns:
            JSON string with full session details
        """
        import json

        try:
            session = api_client.get_session(session_name)
            return json.dumps({"success": True, "session": session}, indent=2)
        except Exception as e:
            logger.error(f"Error getting session {session_name}: {e}", exc_info=True)
            return json.dumps({"success": False, "error": str(e)}, indent=2)

    tools.append(acp_get_session)

    @sdk_tool_decorator
    def acp_create_session(
        session_name: str,
        initial_prompt: Optional[str] = None,
        display_name: Optional[str] = None,
        repos: Optional[str] = None,
        model: Optional[str] = None,
    ) -> str:
        """Create a new agentic session in the current project.

        Creates and starts a new Claude session with the specified configuration.

        Use this to:
        - Spawn a new agent session for a task
        - Start a session with specific repos and prompts
        - Create sessions with custom model settings

        Args:
            session_name: Unique identifier (DNS-compatible: lowercase, hyphens, no spaces)
            initial_prompt: (optional) Initial message to send to the agent
            display_name: (optional) Human-readable name
            repos: (optional) JSON array of repo configs: '[{"url": "https://...", "branch": "main"}]'
            model: (optional) LLM model override (e.g., "claude-sonnet-4-5")

        Returns:
            JSON string with created session details

        Example:
            acp_create_session(
                session_name="my-task",
                initial_prompt="Review recent PRs",
                display_name="PR Review",
                repos='[{"url": "https://github.com/org/repo", "branch": "main"}]'
            )
        """
        import json

        try:
            # Parse repos if provided
            repos_list = None
            if repos:
                try:
                    repos_list = json.loads(repos)
                except json.JSONDecodeError as e:
                    return json.dumps(
                        {
                            "success": False,
                            "error": f"Invalid repos JSON: {e}",
                        },
                        indent=2,
                    )

            session = api_client.create_session(
                session_name=session_name,
                initial_prompt=initial_prompt,
                display_name=display_name,
                repos=repos_list,
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
        except Exception as e:
            logger.error(f"Error creating session {session_name}: {e}", exc_info=True)
            return json.dumps({"success": False, "error": str(e)}, indent=2)

    tools.append(acp_create_session)

    @sdk_tool_decorator
    def acp_stop_session(session_name: str) -> str:
        """Stop a running agentic session.

        Gracefully stops the specified session, cleaning up resources.

        Use this to:
        - Stop a session that has completed its work
        - Clean up idle or stuck sessions
        - Free resources

        Args:
            session_name: Name of the session to stop

        Returns:
            JSON string with confirmation
        """
        import json

        try:
            result = api_client.stop_session(session_name)
            return json.dumps(
                {
                    "success": True,
                    "message": f"Session '{session_name}' stop initiated",
                    "result": result,
                },
                indent=2,
            )
        except Exception as e:
            logger.error(f"Error stopping session {session_name}: {e}", exc_info=True)
            return json.dumps({"success": False, "error": str(e)}, indent=2)

    tools.append(acp_stop_session)

    @sdk_tool_decorator
    def acp_send_message(
        session_name: str,
        message: str,
        thread_id: Optional[str] = None,
    ) -> str:
        """Send a message to an agentic session.

        Sends a user message to the specified session, triggering a new agent run.

        Use this to:
        - Send commands or questions to a running session
        - Continue a conversation with an agent
        - Provide feedback or additional context

        Args:
            session_name: Name of the target session
            message: Message content to send
            thread_id: (optional) Thread ID for multi-threaded sessions

        Returns:
            JSON string with run metadata (runId, threadId)

        Note: This is asynchronous - the agent will process the message in the background.
        To see the response, you would need to monitor events (via frontend or logs).
        """
        import json

        try:
            result = api_client.send_message(
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
        except Exception as e:
            logger.error(
                f"Error sending message to session {session_name}: {e}", exc_info=True
            )
            return json.dumps({"success": False, "error": str(e)}, indent=2)

    tools.append(acp_send_message)

    return tools


def _create_default_client() -> Optional[BackendAPIClient]:
    """Create a default BackendAPIClient from environment variables.

    Returns:
        BackendAPIClient instance, or None if required env vars are missing
    """
    import os

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
