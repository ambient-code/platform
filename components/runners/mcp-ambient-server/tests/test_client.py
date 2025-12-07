"""Unit tests for API client."""

import pytest
import pytest_asyncio
import httpx
from unittest.mock import AsyncMock, Mock, patch
from mcp_ambient_server.client import APIClient


@pytest.fixture
def mock_env(monkeypatch):
    """Mock environment variables for testing."""
    monkeypatch.setenv("BOT_TOKEN", "test-token-12345")
    monkeypatch.setenv("BACKEND_API_URL", "http://test-backend:8080/api")


@pytest_asyncio.fixture
async def api_client(mock_env):
    """Create API client for testing."""
    client = APIClient()
    yield client
    await client.close()


def mock_httpx_response(status_code: int, json_data: dict = None):
    """Create a mock httpx response."""
    response = Mock()
    response.status_code = status_code
    if json_data is not None:
        response.json = Mock(return_value=json_data)
    return response


@pytest.mark.asyncio
async def test_client_initialization_success(mock_env):
    """Test successful client initialization with env vars."""
    client = APIClient()
    assert client.base_url == "http://test-backend:8080/api"
    assert client.token == "test-token-12345"
    await client.close()


@pytest.mark.asyncio
async def test_client_initialization_missing_token(monkeypatch):
    """Test client initialization fails without BOT_TOKEN."""
    monkeypatch.delenv("BOT_TOKEN", raising=False)
    with pytest.raises(ValueError, match="BOT_TOKEN"):
        APIClient()


@pytest.mark.asyncio
async def test_list_projects_success(api_client):
    """Test successful project listing."""
    mock_response = mock_httpx_response(
        200,
        {
            "items": [
                {"metadata": {"name": "project1"}},
                {"metadata": {"name": "project2"}},
            ]
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        projects = await api_client.list_projects()
        assert len(projects) == 2
        assert projects[0]["metadata"]["name"] == "project1"


@pytest.mark.asyncio
async def test_get_project_not_found(api_client):
    """Test getting non-existent project returns 404."""
    mock_response = AsyncMock()
    mock_response.status_code = 404

    with patch.object(api_client.client, "get", return_value=mock_response):
        with pytest.raises(Exception, match="not found"):
            await api_client.get_project("nonexistent")


@pytest.mark.asyncio
async def test_authentication_error(api_client):
    """Test 401 response raises authentication error."""
    mock_response = AsyncMock()
    mock_response.status_code = 401

    with patch.object(api_client.client, "get", return_value=mock_response):
        with pytest.raises(Exception, match="Authentication failed"):
            await api_client.list_projects()


@pytest.mark.asyncio
async def test_authorization_error(api_client):
    """Test 403 response raises access denied error."""
    mock_response = AsyncMock()
    mock_response.status_code = 403

    with patch.object(api_client.client, "get", return_value=mock_response):
        with pytest.raises(Exception, match="Access denied"):
            await api_client.get_project("forbidden-project")


@pytest.mark.asyncio
async def test_server_error(api_client):
    """Test 500 response raises backend error."""
    mock_response = AsyncMock()
    mock_response.status_code = 500
    mock_response.json.return_value = {"error": "Internal server error"}

    with patch.object(api_client.client, "get", return_value=mock_response):
        with pytest.raises(Exception, match="Backend API error"):
            await api_client.list_projects()


@pytest.mark.asyncio
async def test_connection_error(api_client):
    """Test connection error raises connectivity message."""
    with patch.object(
        api_client.client,
        "get",
        side_effect=httpx.ConnectError("Connection refused"),
    ):
        with pytest.raises(Exception, match="Cannot reach backend API"):
            await api_client.list_projects()


@pytest.mark.asyncio
async def test_timeout_error(api_client):
    """Test timeout error raises timeout message."""
    with patch.object(
        api_client.client,
        "get",
        side_effect=httpx.TimeoutException("Timeout"),
    ):
        with pytest.raises(Exception, match="timed out"):
            await api_client.list_projects()


@pytest.mark.asyncio
async def test_path_traversal_protection(api_client):
    """Test that path traversal attempts are blocked."""
    with pytest.raises(ValueError, match="cannot contain"):
        await api_client.get_workspace_file(
            "test-project", "test-session", "../etc/passwd"
        )


@pytest.mark.asyncio
async def test_list_sessions_success(api_client):
    """Test successful session listing."""
    mock_response = mock_httpx_response(
        200,
        {
            "items": [
                {"metadata": {"name": "session1"}},
                {"metadata": {"name": "session2"}},
            ]
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        sessions = await api_client.list_sessions("test-project")
        assert len(sessions) == 2


@pytest.mark.asyncio
async def test_get_workspace_file_success(api_client):
    """Test successful workspace file retrieval."""
    mock_response = mock_httpx_response(
        200,
        {
            "content": "file contents",
            "path": "README.md",
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        file_data = await api_client.get_workspace_file(
            "test-project", "test-session", "README.md"
        )
        assert file_data["path"] == "README.md"
        assert file_data["content"] == "file contents"


@pytest.mark.asyncio
async def test_create_project_success(api_client):
    """Test successful project creation."""
    mock_response = mock_httpx_response(201, {"metadata": {"name": "test-proj"}})

    with patch.object(
        api_client.client, "post", new_callable=AsyncMock, return_value=mock_response
    ):
        project = await api_client.create_project(
            "test-proj", "Test Project", "Description"
        )
        assert project["metadata"]["name"] == "test-proj"


@pytest.mark.asyncio
async def test_check_project_access_success(api_client):
    """Test successful project access check."""
    mock_response = mock_httpx_response(
        200, {"allowed": True, "permissions": ["read", "write"]}
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        access = await api_client.check_project_access("test-project")
        assert access["allowed"] is True


@pytest.mark.asyncio
async def test_get_session_success(api_client):
    """Test successful session retrieval."""
    mock_response = mock_httpx_response(
        200,
        {
            "metadata": {"name": "session1"},
            "spec": {"prompt": "test"},
            "status": {"phase": "Running"},
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        session = await api_client.get_session("test-project", "session1")
        assert session["metadata"]["name"] == "session1"
        assert session["status"]["phase"] == "Running"


@pytest.mark.asyncio
async def test_get_session_k8s_resources_success(api_client):
    """Test successful K8s resources retrieval."""
    mock_response = mock_httpx_response(
        200,
        {
            "job": {"name": "job1", "status": "Running"},
            "pods": [{"name": "pod1", "phase": "Running"}],
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        resources = await api_client.get_session_k8s_resources(
            "test-project", "session1"
        )
        assert resources["job"]["name"] == "job1"
        assert len(resources["pods"]) == 1


@pytest.mark.asyncio
async def test_list_ootb_workflows_success(api_client):
    """Test successful OOTB workflows listing."""
    mock_response = mock_httpx_response(
        200,
        {
            "workflows": [
                {"name": "workflow1", "description": "Test workflow"},
                {"name": "workflow2", "description": "Another workflow"},
            ]
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        workflows = await api_client.list_ootb_workflows()
        assert len(workflows) == 2
        assert workflows[0]["name"] == "workflow1"


@pytest.mark.asyncio
async def test_get_workflow_metadata_success(api_client):
    """Test successful workflow metadata retrieval."""
    mock_response = mock_httpx_response(
        200,
        {
            "name": "rfe-workflow",
            "agents": ["PM", "Architect", "Staff Engineer"],
            "status": "active",
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        metadata = await api_client.get_workflow_metadata("test-project", "session1")
        assert metadata["name"] == "rfe-workflow"
        assert len(metadata["agents"]) == 3


@pytest.mark.asyncio
async def test_get_cluster_info_success(api_client):
    """Test successful cluster info retrieval."""
    mock_response = mock_httpx_response(
        200,
        {
            "isOpenShift": False,
            "version": "1.28",
            "vertexEnabled": False,
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        cluster_info = await api_client.get_cluster_info()
        assert cluster_info["isOpenShift"] is False
        assert cluster_info["version"] == "1.28"


@pytest.mark.asyncio
async def test_create_project_conflict(api_client):
    """Test project creation with conflict error."""
    mock_response = AsyncMock()
    mock_response.status_code = 409

    with patch.object(api_client.client, "post", return_value=mock_response):
        with pytest.raises(Exception, match="failed with status 409"):
            await api_client.create_project("existing-project")


@pytest.mark.asyncio
async def test_list_sessions_empty(api_client):
    """Test listing sessions in empty project."""
    mock_response = mock_httpx_response(200, {"items": []})

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        sessions = await api_client.list_sessions("test-project")
        assert len(sessions) == 0


@pytest.mark.asyncio
async def test_get_session_not_found(api_client):
    """Test getting non-existent session."""
    mock_response = AsyncMock()
    mock_response.status_code = 404

    with patch.object(api_client.client, "get", return_value=mock_response):
        with pytest.raises(Exception, match="not found"):
            await api_client.get_session("test-project", "nonexistent")


@pytest.mark.asyncio
async def test_list_session_workspace_success(api_client):
    """Test successful workspace listing."""
    mock_response = mock_httpx_response(
        200,
        {
            "files": [
                {"name": "file1.txt", "size": 100},
                {"name": "file2.md", "size": 200},
            ]
        },
    )

    with patch.object(
        api_client.client, "get", new_callable=AsyncMock, return_value=mock_response
    ):
        files = await api_client.list_session_workspace("test-project", "session1")
        assert len(files) == 2
        assert files[0]["name"] == "file1.txt"


@pytest.mark.asyncio
async def test_check_project_access_forbidden(api_client):
    """Test project access check when forbidden."""
    mock_response = AsyncMock()
    mock_response.status_code = 403

    with patch.object(api_client.client, "get", return_value=mock_response):
        with pytest.raises(Exception, match="Access denied"):
            await api_client.check_project_access("forbidden-project")


@pytest.mark.asyncio
async def test_get_workflow_metadata_not_found(api_client):
    """Test workflow metadata for non-existent session."""
    mock_response = AsyncMock()
    mock_response.status_code = 404

    with patch.object(api_client.client, "get", return_value=mock_response):
        with pytest.raises(Exception, match="not found"):
            await api_client.get_workflow_metadata("test-project", "nonexistent")
