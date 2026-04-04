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
from urllib.parse import urlencode

from ambient_runner.platform.utils import get_bot_token

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
            bot_token: Bot authentication token (explicit override; if None, reads
                dynamically from file mount or BOT_TOKEN env var on each request)
        """
        self.backend_url = (backend_url or os.getenv("BACKEND_API_URL", "")).rstrip("/")
        self.project_name = (
            project_name
            or os.getenv("PROJECT_NAME")
            or os.getenv("AGENTIC_SESSION_NAMESPACE", "")
        ).strip()
        # Store explicit override separately so _make_request can re-read the
        # token from the file mount on every call (kubelet refreshes the file
        # when the Secret is rotated, but env vars are frozen at pod start).
        self._bot_token_override = bot_token
        # Expose self.bot_token for backward-compatibility with existing callers.
        self.bot_token = (
            bot_token if bot_token is not None else get_bot_token()
        ).strip()

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

        # Re-read the token on every request so kubelet-refreshed file mounts
        # are picked up without restarting the process.
        token = (
            self._bot_token_override.strip()
            if self._bot_token_override is not None
            else get_bot_token()
        )
        if token:
            headers["Authorization"] = f"Bearer {token}"

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

    def _resolve_session_id(self, session_name: str) -> str:
        """Return the session ID for a given session name.

        Looks up the session by name within the current project and returns
        its ID (KSUID), which is required by action routes like /stop and /start.

        Raises:
            ValueError: If no session with that name is found in the project.
        """
        search_expr = f"name = '{session_name}' and project_id = '{self.project_name}'"
        params = urlencode({"search": search_expr, "size": 10})
        path = f"/api/ambient/v1/sessions?{params}"
        response = self._make_request("GET", path)
        items = response.get("items", [])
        if not items:
            raise ValueError(f"Session '{session_name}' not found in project '{self.project_name}'")
        return items[0]["id"]

    def list_sessions(
        self,
        include_completed: bool = False,
        search: Optional[str] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        phase: Optional[str] = None,
    ) -> Dict[str, Any]:
        """List sessions in the project with filtering and pagination.

        Args:
            include_completed: Whether to include stopped/completed sessions
            search: Additional TSL search expression to AND with the project filter
            limit: Max number of sessions to return (default 100)
            offset: Offset for pagination
            phase: Filter by phase (Running, Pending, Stopped, Failed, Completed)

        Returns:
            Dict with 'items', 'totalCount', 'hasMore', 'nextOffset' keys
        """
        base_search = f"project_id = '{self.project_name}'"
        if search:
            base_search = f"{base_search} and {search}"

        query_params: Dict[str, Any] = {
            "search": base_search,
            "size": limit if limit is not None else 100,
        }
        if offset is not None:
            query_params["page"] = (offset // query_params["size"]) + 1
        path = f"/api/ambient/v1/sessions?{urlencode(query_params)}"

        response = self._make_request("GET", path)

        items = response.get("items", [])

        if phase:
            phase_lower = phase.lower()
            items = [
                s
                for s in items
                if s.get("phase", "").lower() == phase_lower
            ]
        elif not include_completed:
            items = [
                s
                for s in items
                if s.get("phase", "").lower()
                not in ("stopped", "completed", "failed")
            ]

        return {
            "items": items,
            "totalCount": len(items),
            "hasMore": response.get("hasMore", False),
            "nextOffset": response.get("nextOffset"),
        }

    def get_session(self, session_name: str) -> Dict[str, Any]:
        """Get details of a specific session by name.

        Args:
            session_name: Name of the session to retrieve

        Returns:
            Session object with full details
        """
        session_id = self._resolve_session_id(session_name)
        path = f"/api/ambient/v1/sessions/{session_id}"
        return self._make_request("GET", path)

    def create_session(
        self,
        session_name: str,
        initial_prompt: Optional[str] = None,
        display_name: Optional[str] = None,
        repos: Optional[List[Dict[str, str]]] = None,
        model: Optional[str] = None,
        workflow: Optional[Dict[str, str]] = None,
    ) -> Dict[str, Any]:
        """Create a new agentic session.

        Automatically sets the current session as the parent so the child
        inherits the parent's userContext (and therefore credentials).

        Args:
            session_name: Unique name for the session (must be DNS-compatible)
            initial_prompt: Optional initial prompt to send to the agent
            display_name: Optional human-readable display name
            repos: Optional list of repository configurations [{"url": "...", "branch": "..."}]
            model: Optional LLM model override (e.g., "claude-sonnet-4-5")
            workflow: Optional workflow config {"gitUrl": "...", "branch": "...", "path": "..."}

        Returns:
            Created session object
        """
        path = "/api/ambient/v1/sessions"

        payload: Dict[str, Any] = {
            "name": session_name,
            "project_id": self.project_name,
        }

        parent_session = os.getenv("AGENTIC_SESSION_NAME", "").strip()
        if parent_session:
            payload["parent_session_id"] = parent_session

        if initial_prompt:
            payload["prompt"] = initial_prompt
        if display_name:
            payload["display_name"] = display_name
        if repos:
            payload["repos"] = json.dumps(repos) if not isinstance(repos, str) else repos
        if model:
            payload["llm_model"] = model
        if workflow:
            payload["workflow_id"] = workflow.get("id") or workflow.get("gitUrl", "")

        return self._make_request("POST", path, data=payload)

    def stop_session(self, session_name: str) -> Dict[str, Any]:
        """Stop a running session.

        Args:
            session_name: Name of the session to stop

        Returns:
            Response from the backend
        """
        session_id = self._resolve_session_id(session_name)
        path = f"/api/ambient/v1/sessions/{session_id}/stop"
        return self._make_request("POST", path)

    def send_message(
        self,
        session_name: str,
        message: str,
        thread_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Send a message to a session.

        Args:
            session_name: Name of the session
            message: Message content to send
            thread_id: Unused (retained for API compatibility)

        Returns:
            Pushed message object
        """
        session_id = self._resolve_session_id(session_name)
        path = f"/api/ambient/v1/sessions/{session_id}/messages"

        payload: Dict[str, Any] = {
            "event_type": "user",
            "payload": message,
        }

        return self._make_request("POST", path, data=payload)

    def start_session(self, session_name: str) -> Dict[str, Any]:
        """Start a stopped session.

        Args:
            session_name: Name of the session to start

        Returns:
            Response from the backend
        """
        session_id = self._resolve_session_id(session_name)
        path = f"/api/ambient/v1/sessions/{session_id}/start"
        return self._make_request("POST", path)

    def list_workflows(self) -> List[Dict[str, Any]]:
        """List available out-of-the-box workflows.

        Returns:
            List of workflow objects with id, name, description, gitUrl, branch, path
        """
        path = "/workflows/ootb"
        query = f"?project={self.project_name}" if self.project_name else ""
        response = self._make_request("GET", f"{path}{query}")
        return response.get("workflows", [])

    # Cap exported events to avoid fetching/parsing huge session histories
    _MAX_EXPORT_EVENTS = 200

    def get_session_events(
        self,
        session_name: str,
        max_events: Optional[int] = None,
    ) -> List[Dict[str, Any]]:
        """Get recent historical events from a session via the export endpoint.

        Uses the /export endpoint which returns a JSON response with
        persisted AG-UI events. The /agui/events endpoint is SSE-based
        and would block indefinitely.

        Only the last `max_events` events are returned to avoid
        transferring/parsing entire session histories for long sessions.

        Args:
            session_name: Name of the session
            max_events: Max events to return (default: _MAX_EXPORT_EVENTS)

        Returns:
            List of event objects (tail-sliced)
        """
        cap = max_events if max_events and max_events > 0 else self._MAX_EXPORT_EVENTS

        session_id = self._resolve_session_id(session_name)
        path = f"/api/ambient/v1/sessions/{session_id}/messages"
        response = self._make_request("GET", path)
        items = response.get("items", [])
        if isinstance(items, list):
            return items[-cap:]
        return []
