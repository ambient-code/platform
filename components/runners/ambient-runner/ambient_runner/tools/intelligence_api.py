"""API client for the repo intelligence service (ambient-api-server).

Provides methods to query, create, and manage repo intelligence and
findings. Used by MCP memory tools to give agents persistent project
knowledge across sessions.
"""

import json
import logging
import os
import urllib.request
from typing import Any, Dict, Optional
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode

from ambient_runner.platform.utils import get_bot_token

logger = logging.getLogger(__name__)


class IntelligenceAPIClient:
    """Client for repo intelligence endpoints on the ambient-api-server."""

    def __init__(
        self,
        api_server_url: Optional[str] = None,
        project_id: Optional[str] = None,
        bot_token: Optional[str] = None,
    ):
        raw_url = (
            api_server_url
            or os.getenv("API_SERVER_URL", "")
            or os.getenv("BACKEND_API_URL", "")
        ).rstrip("/")
        # BACKEND_API_URL typically ends with /api (e.g. http://backend:8080/api)
        # but our paths start with /api/ambient/v1/. Strip trailing /api to
        # avoid double /api/api/.
        if raw_url.endswith("/api"):
            raw_url = raw_url[:-4]
        self.api_server_url = raw_url
        self.project_id = (
            project_id
            or os.getenv("PROJECT_NAME")
            or os.getenv("AGENTIC_SESSION_NAMESPACE", "")
        ).strip()
        self._bot_token_override = bot_token

        if not self.api_server_url:
            raise ValueError(
                "API_SERVER_URL or BACKEND_API_URL environment variable is required"
            )

    def _make_request(
        self,
        method: str,
        path: str,
        data: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        url = f"{self.api_server_url}{path}"
        headers = {"Content-Type": "application/json"}

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
            logger.error(f"Intelligence API HTTP {e.code} from {url}: {error_body}")
            raise
        except URLError as e:
            logger.error(f"Intelligence API URL error from {url}: {e.reason}")
            raise

    def lookup_intelligence(self, repo_url: str) -> Optional[Dict[str, Any]]:
        """Get intelligence for a repo in this project, or None if not found."""
        params = urlencode(
            {"project_id": self.project_id, "repo_url": repo_url}
        )
        path = f"/api/ambient/v1/repo_intelligences/lookup?{params}"
        try:
            return self._make_request("GET", path)
        except HTTPError as e:
            if e.code == 404:
                return None
            raise

    def create_intelligence(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new repo intelligence record."""
        data["project_id"] = self.project_id
        return self._make_request(
            "POST", "/api/ambient/v1/repo_intelligences", data
        )

    @staticmethod
    def _escape_tsl(value: str) -> str:
        """Escape a value for safe interpolation into TSL search strings."""
        return value.replace("'", "''")

    def list_findings(
        self,
        intelligence_id: str,
        file_path: Optional[str] = None,
        category: Optional[str] = None,
        status: str = "active",
    ) -> Dict[str, Any]:
        """List findings for an intelligence record with optional filters."""
        filters = [f"status = '{self._escape_tsl(status)}'"]
        if file_path:
            filters.append(f"file_path like '%{self._escape_tsl(file_path)}%'")
        if category:
            filters.append(f"category = '{self._escape_tsl(category)}'")

        search = " and ".join(filters)
        params = urlencode({"search": search})
        path = f"/api/ambient/v1/repo_intelligences/{intelligence_id}/findings?{params}"
        return self._make_request("GET", path)

    def create_finding(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new repo finding."""
        return self._make_request(
            "POST", "/api/ambient/v1/repo_findings", data
        )

    def intelligence_exists(self, repo_url: str) -> bool:
        """Check if intelligence exists for a repo in this project."""
        return self.lookup_intelligence(repo_url) is not None

    def delete_intelligence(self, repo_url: str) -> bool:
        """Delete intelligence for a repo via the DeleteByLookup endpoint.

        Uses DELETE /lookup?project_id=X&repo_url=Y so the server handles
        lookup and soft-delete atomically in a single request.
        Returns True if deleted, False if no record existed.
        """
        params = urlencode(
            {"project_id": self.project_id, "repo_url": repo_url}
        )
        path = f"/api/ambient/v1/repo_intelligences/lookup?{params}"
        try:
            self._make_request("DELETE", path)
            logger.info(
                f"Deleted intelligence for {repo_url} "
                f"(project={self.project_id})"
            )
            return True
        except HTTPError as e:
            if e.code == 404:
                return False
            raise
