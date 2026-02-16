import os
from typing import Optional
from urllib.parse import urlparse

import httpx

from ._base import APIError


class AmbientClient:
    def __init__(
        self,
        base_url: str,
        token: str,
        project: str,
        *,
        base_path: str = "/api/ambient-api-server/v1",
        timeout: float = 30.0,
    ):
        self._validate_token(token)
        self._validate_project(project)
        self._validate_base_url(base_url)

        self._base_url = base_url.rstrip("/")
        self._base_path = base_path
        self._token = token
        self._project = project

        self._http = httpx.Client(
            timeout=timeout,
            headers={
                "Authorization": f"Bearer {token}",
                "X-Ambient-Project": project,
                "Content-Type": "application/json",
            },
        )

        self._api_cache: dict = {}

    def __enter__(self) -> "AmbientClient":
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:  # type: ignore[no-untyped-def]
        self.close()

    def close(self) -> None:
        self._http.close()

    def _request(
        self,
        method: str,
        path: str,
        *,
        json: Optional[dict] = None,
        params: Optional[dict] = None,
        expect_json: bool = True,
    ) -> dict:
        url = self._base_url + self._base_path + path

        try:
            resp = self._http.request(method, url, json=json, params=params)
        except httpx.RequestError as exc:
            raise ConnectionError(f"request failed: {exc}") from exc

        if resp.status_code >= 400:
            try:
                data = resp.json()
            except Exception:
                data = {}
            raise APIError(
                status_code=resp.status_code,
                code=data.get("code", str(resp.status_code)),
                reason=data.get("reason", resp.text[:200] if resp.text else ""),
                operation_id=data.get("operation_id", ""),
            )

        if not expect_json:
            return {}

        return resp.json()  # type: ignore[no-any-return]

    @property
    def sessions(self):  # type: ignore[no-untyped-def]
        return self._get_api("sessions")

    @property
    def agents(self):  # type: ignore[no-untyped-def]
        return self._get_api("agents")

    @property
    def tasks(self):  # type: ignore[no-untyped-def]
        return self._get_api("tasks")

    @property
    def skills(self):  # type: ignore[no-untyped-def]
        return self._get_api("skills")

    @property
    def workflows(self):  # type: ignore[no-untyped-def]
        return self._get_api("workflows")

    @property
    def users(self):  # type: ignore[no-untyped-def]
        return self._get_api("users")

    @property
    def workflow_skills(self):  # type: ignore[no-untyped-def]
        return self._get_api("workflow_skills")

    @property
    def workflow_tasks(self):  # type: ignore[no-untyped-def]
        return self._get_api("workflow_tasks")

    @property
    def projects(self):  # type: ignore[no-untyped-def]
        return self._get_api("projects")

    @property
    def project_settings(self):  # type: ignore[no-untyped-def]
        return self._get_api("project_settings")

    @property
    def permissions(self):  # type: ignore[no-untyped-def]
        return self._get_api("permissions")

    @property
    def repository_refs(self):  # type: ignore[no-untyped-def]
        return self._get_api("repository_refs")

    @property
    def project_keys(self):  # type: ignore[no-untyped-def]
        return self._get_api("project_keys")

    def _get_api(self, name: str):  # type: ignore[no-untyped-def]
        if name not in self._api_cache:
            if name == "sessions":
                from ._session_api import SessionAPI
                self._api_cache[name] = SessionAPI(self)
            elif name == "agents":
                from ._agent_api import AgentAPI
                self._api_cache[name] = AgentAPI(self)
            elif name == "tasks":
                from ._task_api import TaskAPI
                self._api_cache[name] = TaskAPI(self)
            elif name == "skills":
                from ._skill_api import SkillAPI
                self._api_cache[name] = SkillAPI(self)
            elif name == "workflows":
                from ._workflow_api import WorkflowAPI
                self._api_cache[name] = WorkflowAPI(self)
            elif name == "users":
                from ._user_api import UserAPI
                self._api_cache[name] = UserAPI(self)
            elif name == "workflow_skills":
                from ._workflow_skill_api import WorkflowSkillAPI
                self._api_cache[name] = WorkflowSkillAPI(self)
            elif name == "workflow_tasks":
                from ._workflow_task_api import WorkflowTaskAPI
                self._api_cache[name] = WorkflowTaskAPI(self)
            elif name == "projects":
                from ._project_api import ProjectAPI
                self._api_cache[name] = ProjectAPI(self)
            elif name == "project_settings":
                from ._project_settings_api import ProjectSettingsAPI
                self._api_cache[name] = ProjectSettingsAPI(self)
            elif name == "permissions":
                from ._permission_api import PermissionAPI
                self._api_cache[name] = PermissionAPI(self)
            elif name == "repository_refs":
                from ._repository_ref_api import RepositoryRefAPI
                self._api_cache[name] = RepositoryRefAPI(self)
            elif name == "project_keys":
                from ._project_key_api import ProjectKeyAPI
                self._api_cache[name] = ProjectKeyAPI(self)
        return self._api_cache[name]

    @classmethod
    def from_env(cls, **kwargs) -> "AmbientClient":  # type: ignore[no-untyped-def]
        base_url = kwargs.pop("base_url", None) or os.getenv(
            "AMBIENT_API_URL", "http://localhost:8080"
        )
        token = kwargs.pop("token", None) or os.getenv("AMBIENT_TOKEN")
        project = kwargs.pop("project", None) or os.getenv("AMBIENT_PROJECT")

        if not token:
            raise ValueError("AMBIENT_TOKEN environment variable is required")
        if not project:
            raise ValueError("AMBIENT_PROJECT environment variable is required")

        return cls(base_url=base_url, token=token, project=project, **kwargs)

    def _validate_token(self, token: str) -> None:
        if not token:
            raise ValueError("token cannot be empty")
        if len(token) < 10:
            raise ValueError("token appears too short to be valid")
        if token.lower() in ("your_token_here", "your-token-here", "token", "bearer"):
            raise ValueError("token appears to be a placeholder value")

    def _validate_project(self, project: str) -> None:
        if not project:
            raise ValueError("project cannot be empty")
        if not project.replace("-", "").replace("_", "").isalnum():
            raise ValueError("project must contain only alphanumeric characters, hyphens, and underscores")
        if len(project) > 63:
            raise ValueError("project name cannot exceed 63 characters")

    def _validate_base_url(self, base_url: str) -> None:
        if not base_url:
            raise ValueError("base URL cannot be empty")
        parsed = urlparse(base_url)
        if not parsed.scheme or not parsed.netloc:
            raise ValueError("base URL must include scheme and host")
        if parsed.scheme not in ("http", "https"):
            raise ValueError("base URL scheme must be http or https")
        if "example.com" in parsed.netloc:
            raise ValueError("base URL appears to contain placeholder domain")
