"""Backend API client tools for Claude Agent SDK.

Provides custom tools for session management operations until MCP server is ready.
These tools allow a running session to interface with the backend API.
"""

import json
import logging
import os
import urllib.request
import uuid
from typing import Any, Dict, List, Optional
from urllib.error import HTTPError, URLError

logger = logging.getLogger(__name__)


class BackendAPIClient:
    """Client for making authenticated requests to the backend API."""

    def __init__(
        self,
        backend_url: Optional[str] = None,
        project_name: Optional[str] = None,
        bot_token: Optional[str] = None,
    ):
        """Initialize the backend API client.

        Args:
            backend_url: Base URL of the backend API (defaults to BACKEND_API_URL env var)
            project_name: Project name (defaults to PROJECT_NAME or AGENTIC_SESSION_NAMESPACE env var)
            bot_token: Bot authentication token (defaults to BOT_TOKEN env var)
        """
        self.backend_url = (backend_url or os.getenv("BACKEND_API_URL", "")).rstrip("/")
        self.project_name = (
            project_name
            or os.getenv("PROJECT_NAME")
            or os.getenv("AGENTIC_SESSION_NAMESPACE", "")
        ).strip()
        self.bot_token = (bot_token or os.getenv("BOT_TOKEN", "")).strip()

        if not self.backend_url:
            raise ValueError("BACKEND_API_URL environment variable is required")
        if not self.project_name:
            raise ValueError(
                "PROJECT_NAME or AGENTIC_SESSION_NAMESPACE environment variable is required"
            )

    def _make_request(
        self,
        method: str,
        path: str,
        data: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make an authenticated HTTP request to the backend API.

        Args:
            method: HTTP method (GET, POST, etc.)
            path: API path (will be prefixed with backend_url)
            data: Optional JSON payload for POST/PUT requests

        Returns:
            Parsed JSON response

        Raises:
            HTTPError: If the request fails
        """
        url = f"{self.backend_url}{path}"
        headers = {"Content-Type": "application/json"}

        if self.bot_token:
            headers["Authorization"] = f"Bearer {self.bot_token}"

        request_kwargs: Dict[str, Any] = {"method": method, "headers": headers}
        if data is not None:
            request_kwargs["data"] = json.dumps(data).encode("utf-8")

        req = urllib.request.Request(url, **request_kwargs)

        try:
            with urllib.request.urlopen(req, timeout=30) as response:
                response_data = response.read().decode("utf-8")
                if response_data:
                    return json.loads(response_data)
                return {}
        except HTTPError as e:
            error_body = e.read().decode("utf-8") if e.fp else ""
            logger.error(f"HTTP {e.code} error from {url}: {error_body}")
            raise
        except URLError as e:
            logger.error(f"URL error from {url}: {e.reason}")
            raise

    def list_sessions(self, include_completed: bool = False) -> List[Dict[str, Any]]:
        """List all sessions in the project.

        Args:
            include_completed: Whether to include completed sessions

        Returns:
            List of session objects
        """
        path = f"/projects/{self.project_name}/agentic-sessions"
        response = self._make_request("GET", path)

        sessions = response.get("sessions", [])
        if not include_completed:
            # Filter out completed sessions (phase == "Stopped")
            sessions = [s for s in sessions if s.get("phase") != "Stopped"]

        return sessions

    def get_session(self, session_name: str) -> Dict[str, Any]:
        """Get details of a specific session.

        Args:
            session_name: Name of the session to retrieve

        Returns:
            Session object with full details
        """
        path = f"/projects/{self.project_name}/agentic-sessions/{session_name}"
        return self._make_request("GET", path)

    def create_session(
        self,
        session_name: str,
        initial_prompt: Optional[str] = None,
        display_name: Optional[str] = None,
        repos: Optional[List[Dict[str, str]]] = None,
        model: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Create a new agentic session.

        Args:
            session_name: Unique name for the session (must be DNS-compatible)
            initial_prompt: Optional initial prompt to send to the agent
            display_name: Optional human-readable display name
            repos: Optional list of repository configurations [{"url": "...", "branch": "..."}]
            model: Optional LLM model override (e.g., "claude-sonnet-4-5")

        Returns:
            Created session object
        """
        path = f"/projects/{self.project_name}/agentic-sessions"

        payload: Dict[str, Any] = {
            "sessionName": session_name,
        }

        if initial_prompt:
            payload["initialPrompt"] = initial_prompt
        if display_name:
            payload["displayName"] = display_name
        if repos:
            payload["repos"] = repos
        if model:
            payload["llmSettings"] = {"model": model}

        return self._make_request("POST", path, data=payload)

    def stop_session(self, session_name: str) -> Dict[str, Any]:
        """Stop a running session.

        Args:
            session_name: Name of the session to stop

        Returns:
            Response from the backend
        """
        path = f"/projects/{self.project_name}/agentic-sessions/{session_name}/stop"
        return self._make_request("POST", path)

    def send_message(
        self,
        session_name: str,
        message: str,
        thread_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Send a message to a session (AG-UI run endpoint).

        Args:
            session_name: Name of the session
            message: Message content to send
            thread_id: Optional thread ID for multi-threaded sessions

        Returns:
            Run metadata (runId, threadId)
        """
        path = f"/projects/{self.project_name}/agentic-sessions/{session_name}/agui/run"

        msg_id = str(uuid.uuid4())
        payload: Dict[str, Any] = {
            "messages": [{"id": msg_id, "role": "user", "content": message}],
        }
        if thread_id:
            payload["threadId"] = thread_id

        return self._make_request("POST", path, data=payload)

    def get_session_events(
        self,
        session_name: str,
        thread_id: Optional[str] = None,
        limit: Optional[int] = None,
    ) -> List[Dict[str, Any]]:
        """Get events from a session (AG-UI events endpoint).

        Note: This returns historical events only (not a live stream).

        Args:
            session_name: Name of the session
            thread_id: Optional thread ID filter
            limit: Optional limit on number of events

        Returns:
            List of event objects
        """
        path = (
            f"/projects/{self.project_name}/agentic-sessions/{session_name}/agui/events"
        )

        # Note: This endpoint is typically SSE-based for live streaming,
        # but can be called once for historical events
        query_params = []
        if thread_id:
            query_params.append(f"threadId={thread_id}")
        if limit:
            query_params.append(f"limit={limit}")

        if query_params:
            path = f"{path}?{'&'.join(query_params)}"

        # For now, we'll just return the raw response
        # A full implementation would parse SSE stream
        return self._make_request("GET", path).get("events", [])
