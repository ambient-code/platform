"""
HTTP client for the Ambient Platform Public API.
"""

import json
import time
from typing import Optional
import httpx

from .types import (
    SessionResponse,
    SessionListResponse,
    CreateSessionRequest,
    CreateSessionResponse,
    ErrorResponse,
    StatusCompleted,
    StatusFailed,
)
from .exceptions import (
    AmbientAPIError,
    AmbientConnectionError,
    SessionNotFoundError,
    AuthenticationError,
)


class AmbientClient:
    """Simple HTTP client for the Ambient Platform API."""

    def __init__(
        self,
        base_url: str,
        token: str,
        project: str,
        timeout: float = 30.0,
    ):
        """
        Initialize the Ambient Platform client.

        Args:
            base_url: API base URL (e.g., "https://api.ambient-code.io")
            token: Bearer token for authentication
            project: Project name (Kubernetes namespace)
            timeout: HTTP request timeout in seconds
        """
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.project = project
        self.timeout = timeout
        
        # Create HTTP client with headers
        self.client = httpx.Client(
            timeout=timeout,
            headers={
                "Authorization": f"Bearer {token}",
                "X-Ambient-Project": project,
                "Content-Type": "application/json",
            },
        )

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

    def close(self):
        """Close the HTTP client."""
        self.client.close()

    def _handle_response(self, response: httpx.Response) -> dict:
        """Handle HTTP response and raise appropriate exceptions."""
        try:
            data = response.json()
        except (json.JSONDecodeError, ValueError):
            data = {"error": f"Invalid JSON response: {response.text}"}

        if response.status_code == 401:
            error_msg = data.get("error", "Unauthorized")
            raise AuthenticationError(f"Authentication failed: {error_msg}")
        elif response.status_code == 404:
            error_msg = data.get("error", "Not found")
            raise SessionNotFoundError(error_msg)
        elif response.status_code >= 400:
            error_msg = data.get("error", f"HTTP {response.status_code}")
            message = data.get("message", "")
            full_msg = f"{error_msg}" + (f" - {message}" if message else "")
            raise AmbientAPIError(f"API error ({response.status_code}): {full_msg}")

        return data

    def create_session(self, request: CreateSessionRequest) -> CreateSessionResponse:
        """
        Create a new agentic session.

        Args:
            request: Session creation request

        Returns:
            CreateSessionResponse with session ID and message

        Raises:
            AmbientAPIError: If the API request fails
            AuthenticationError: If authentication fails
            AmbientConnectionError: If connection fails
        """
        url = f"{self.base_url}/v1/sessions"
        
        try:
            response = self.client.post(url, json=request.to_dict())
        except httpx.RequestError as e:
            raise AmbientConnectionError(f"Failed to connect to API: {e}")

        data = self._handle_response(response)
        return CreateSessionResponse.from_dict(data)

    def get_session(self, session_id: str) -> SessionResponse:
        """
        Retrieve a session by ID.

        Args:
            session_id: Unique session identifier

        Returns:
            SessionResponse with session details

        Raises:
            SessionNotFoundError: If session doesn't exist
            AmbientAPIError: If the API request fails
            AuthenticationError: If authentication fails
            AmbientConnectionError: If connection fails
        """
        url = f"{self.base_url}/v1/sessions/{session_id}"
        
        try:
            response = self.client.get(url)
        except httpx.RequestError as e:
            raise AmbientConnectionError(f"Failed to connect to API: {e}")

        data = self._handle_response(response)
        return SessionResponse.from_dict(data)

    def list_sessions(self) -> SessionListResponse:
        """
        List all accessible sessions.

        Returns:
            SessionListResponse with session list and total count

        Raises:
            AmbientAPIError: If the API request fails
            AuthenticationError: If authentication fails
            AmbientConnectionError: If connection fails
        """
        url = f"{self.base_url}/v1/sessions"
        
        try:
            response = self.client.get(url)
        except httpx.RequestError as e:
            raise AmbientConnectionError(f"Failed to connect to API: {e}")

        data = self._handle_response(response)
        return SessionListResponse.from_dict(data)

    def wait_for_completion(
        self,
        session_id: str,
        poll_interval: float = 5.0,
        timeout: Optional[float] = None,
    ) -> SessionResponse:
        """
        Poll a session until it reaches a terminal state.

        Args:
            session_id: Session ID to monitor
            poll_interval: Time between polls in seconds
            timeout: Maximum time to wait in seconds (None = no limit)

        Returns:
            SessionResponse when session completes or fails

        Raises:
            TimeoutError: If timeout is reached
            SessionNotFoundError: If session doesn't exist
            AmbientAPIError: If the API request fails
            AmbientConnectionError: If connection fails
        """
        start_time = time.time()
        
        while True:
            session = self.get_session(session_id)
            
            # Check if session reached terminal state
            if session.status in (StatusCompleted, StatusFailed):
                return session
            
            # Check timeout
            if timeout and (time.time() - start_time) > timeout:
                raise TimeoutError(
                    f"Session monitoring timed out after {timeout} seconds"
                )
            
            # Wait before next poll
            time.sleep(poll_interval)

    @classmethod
    def from_env(cls, **kwargs) -> "AmbientClient":
        """
        Create client from environment variables.

        Environment variables:
            AMBIENT_API_URL: API base URL (default: http://localhost:8080)
            AMBIENT_TOKEN: Bearer token (required)
            AMBIENT_PROJECT: Project name (required)

        Args:
            **kwargs: Additional arguments to override environment

        Returns:
            Configured AmbientClient

        Raises:
            ValueError: If required environment variables are missing
        """
        import os
        
        base_url = kwargs.get("base_url") or os.getenv(
            "AMBIENT_API_URL", "http://localhost:8080"
        )
        token = kwargs.get("token") or os.getenv("AMBIENT_TOKEN")
        project = kwargs.get("project") or os.getenv("AMBIENT_PROJECT")
        
        if not token:
            raise ValueError("AMBIENT_TOKEN environment variable is required")
        if not project:
            raise ValueError("AMBIENT_PROJECT environment variable is required")
        
        return cls(
            base_url=base_url,
            token=token,
            project=project,
            **{k: v for k, v in kwargs.items() if k not in ("base_url", "token", "project")}
        )