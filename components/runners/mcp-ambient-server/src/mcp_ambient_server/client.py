"""HTTP client for Ambient Code Platform backend API."""

import os
import logging
from typing import Any, Optional
import httpx


logger = logging.getLogger(__name__)


class APIClient:
    """Async HTTP client for Ambient Code backend API.

    Handles authentication, error handling, and type-safe requests to the
    backend API running in the same Kubernetes cluster.
    """

    def __init__(self, base_url: Optional[str] = None, token: Optional[str] = None):
        """Initialize API client with authentication.

        Args:
            base_url: Backend API base URL (default from BACKEND_API_URL env)
            token: Bearer token (default from BOT_TOKEN env)

        Raises:
            ValueError: If BOT_TOKEN is not set
        """
        self.base_url = base_url or os.getenv(
            "BACKEND_API_URL",
            "http://vteam-backend.ambient-code.svc.cluster.local:8080/api",
        )
        self.token = token or os.getenv("BOT_TOKEN", "")

        if not self.token:
            raise ValueError("BOT_TOKEN environment variable must be set")

        logger.info(
            "Initialized API client: base_url=%s, token_len=%d",
            self.base_url,
            len(self.token),
        )

        self.client = httpx.AsyncClient(
            base_url=self.base_url,
            headers={"Authorization": f"Bearer {self.token}"},
            timeout=30.0,
        )

    async def close(self):
        """Close the HTTP client."""
        await self.client.aclose()

    async def _handle_response(
        self, response: httpx.Response, resource_type: str = "resource"
    ) -> dict[str, Any]:
        """Handle HTTP response with error mapping.

        Args:
            response: HTTP response object
            resource_type: Type of resource for error messages

        Returns:
            Parsed JSON response

        Raises:
            Exception: With informative error message based on status code
        """
        if response.status_code >= 200 and response.status_code < 300:
            return response.json()

        # Map status codes to user-friendly error messages
        if response.status_code == 401:
            raise Exception(
                "Authentication failed. BOT_TOKEN may be invalid or expired."
            )
        elif response.status_code == 403:
            raise Exception(
                f"Access denied. User does not have permission for this {resource_type}."
            )
        elif response.status_code == 404:
            raise Exception(f"{resource_type.capitalize()} not found")
        elif response.status_code >= 500:
            error_msg = "Unknown backend error"
            try:
                error_data = response.json()
                error_msg = error_data.get("error", error_msg)
            except Exception:
                pass
            raise Exception(f"Backend API error: {error_msg}")
        else:
            raise Exception(f"Request failed with status {response.status_code}")

    # Project Management

    async def create_project(
        self, name: str, display_name: str = "", description: str = ""
    ) -> dict[str, Any]:
        """Create a new project.

        Args:
            name: Project name (required, must be valid Kubernetes namespace name)
            display_name: Display name (optional, used on OpenShift)
            description: Project description (optional, used on OpenShift)

        Returns:
            Created project object with name, namespace, status

        Raises:
            Exception: If project creation fails (conflict, forbidden, etc.)
        """
        payload = {"name": name}
        if display_name:
            payload["displayName"] = display_name
        if description:
            payload["description"] = description

        try:
            response = await self.client.post("/projects", json=payload)
            return await self._handle_response(response, "project")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def list_projects(self) -> list[dict[str, Any]]:
        """List all projects accessible by the user.

        Returns:
            List of project objects
        """
        try:
            response = await self.client.get("/projects")
            data = await self._handle_response(response, "projects")
            return data.get("items", [])
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def get_project(self, project_name: str) -> dict[str, Any]:
        """Get project details.

        Args:
            project_name: Name of the project

        Returns:
            Project object
        """
        try:
            response = await self.client.get(f"/projects/{project_name}")
            return await self._handle_response(response, "project")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def check_project_access(self, project_name: str) -> dict[str, Any]:
        """Check user's access permissions for a project.

        Args:
            project_name: Name of the project

        Returns:
            Access information object
        """
        try:
            response = await self.client.get(f"/projects/{project_name}/access")
            return await self._handle_response(response, "access information")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    # Session Browsing

    async def list_sessions(self, project_name: str) -> list[dict[str, Any]]:
        """List agentic sessions in a project.

        Args:
            project_name: Name of the project

        Returns:
            List of session objects
        """
        try:
            response = await self.client.get(
                f"/projects/{project_name}/agentic-sessions"
            )
            data = await self._handle_response(response, "sessions")
            return data.get("items", [])
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def get_session(self, project_name: str, session_name: str) -> dict[str, Any]:
        """Get agentic session details.

        Args:
            project_name: Name of the project
            session_name: Name of the session

        Returns:
            Session object
        """
        try:
            response = await self.client.get(
                f"/projects/{project_name}/agentic-sessions/{session_name}"
            )
            return await self._handle_response(response, "session")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def get_session_k8s_resources(
        self, project_name: str, session_name: str
    ) -> dict[str, Any]:
        """Get Kubernetes resources for a session.

        Args:
            project_name: Name of the project
            session_name: Name of the session

        Returns:
            Kubernetes resources object
        """
        try:
            response = await self.client.get(
                f"/projects/{project_name}/agentic-sessions/{session_name}/k8s-resources"
            )
            return await self._handle_response(response, "k8s resources")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def list_session_workspace(
        self, project_name: str, session_name: str
    ) -> list[dict[str, Any]]:
        """List files in session workspace.

        Args:
            project_name: Name of the project
            session_name: Name of the session

        Returns:
            List of workspace file objects
        """
        try:
            response = await self.client.get(
                f"/projects/{project_name}/agentic-sessions/{session_name}/workspace"
            )
            data = await self._handle_response(response, "workspace")
            return data.get("files", [])
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    # Workspace File Access

    async def get_workspace_file(
        self, project_name: str, session_name: str, path: str
    ) -> dict[str, Any]:
        """Get contents of a file in session workspace.

        Args:
            project_name: Name of the project
            session_name: Name of the session
            path: Path to the file (relative to workspace root)

        Returns:
            File content object
        """
        # Validate path to prevent path traversal attacks
        if ".." in path:
            raise ValueError("Path cannot contain '..' components")

        try:
            response = await self.client.get(
                f"/projects/{project_name}/agentic-sessions/{session_name}/workspace/{path}"
            )
            return await self._handle_response(response, "file")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    # Workflow & Cluster Info

    async def list_ootb_workflows(self) -> list[dict[str, Any]]:
        """List out-of-the-box workflows.

        Returns:
            List of workflow objects
        """
        try:
            response = await self.client.get("/workflows/ootb")
            data = await self._handle_response(response, "workflows")
            return data.get("workflows", [])
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def get_workflow_metadata(
        self, project_name: str, session_name: str
    ) -> dict[str, Any]:
        """Get workflow metadata for a session.

        Args:
            project_name: Name of the project
            session_name: Name of the session

        Returns:
            Workflow metadata object
        """
        try:
            response = await self.client.get(
                f"/projects/{project_name}/agentic-sessions/{session_name}/workflow/metadata"
            )
            return await self._handle_response(response, "workflow metadata")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def get_cluster_info(self) -> dict[str, Any]:
        """Get cluster information.

        Returns:
            Cluster info object
        """
        try:
            response = await self.client.get("/cluster-info")
            return await self._handle_response(response, "cluster info")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")

    async def get_health(self) -> dict[str, Any]:
        """Get backend health status.

        Returns:
            Health status object
        """
        try:
            # Health endpoint is at root, not under /api - use full URL
            health_url = self.base_url.replace("/api", "") + "/health"
            async with httpx.AsyncClient(
                headers={"Authorization": f"Bearer {self.token}"},
                timeout=30.0,
            ) as client:
                response = await client.get(health_url)
                return await self._handle_response(response, "health status")
        except httpx.ConnectError:
            raise Exception("Cannot reach backend API. Check cluster connectivity.")
        except httpx.TimeoutException:
            raise Exception("Request timed out after 30s. Backend may be overloaded.")
