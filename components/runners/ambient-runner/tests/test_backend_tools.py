"""Unit tests for backend API tools."""

import json
from unittest.mock import MagicMock, patch
from urllib.error import HTTPError

import pytest

from ambient_runner.tools.backend_api import BackendAPIClient


class TestBackendAPIClient:
    """Test the BackendAPIClient class."""

    def test_init_from_env(self, monkeypatch):
        """Test client initialization from environment variables."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.setenv("BOT_TOKEN", "test-token-123")

        client = BackendAPIClient()

        assert client.backend_url == "http://backend:8080/api"
        assert client.project_name == "test-project"
        assert client.bot_token == "test-token-123"

    def test_init_from_params(self):
        """Test client initialization from constructor params."""
        client = BackendAPIClient(
            backend_url="http://custom:9090/api",
            project_name="custom-project",
            bot_token="custom-token",
        )

        assert client.backend_url == "http://custom:9090/api"
        assert client.project_name == "custom-project"
        assert client.bot_token == "custom-token"

    def test_init_missing_backend_url(self, monkeypatch):
        """Test client raises error when BACKEND_API_URL is missing."""
        monkeypatch.delenv("BACKEND_API_URL", raising=False)
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        with pytest.raises(ValueError, match="BACKEND_API_URL"):
            BackendAPIClient()

    def test_init_missing_project_name(self, monkeypatch):
        """Test client raises error when PROJECT_NAME is missing."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.delenv("PROJECT_NAME", raising=False)
        monkeypatch.delenv("AGENTIC_SESSION_NAMESPACE", raising=False)

        with pytest.raises(ValueError, match="PROJECT_NAME"):
            BackendAPIClient()

    def test_url_trailing_slash_stripped(self):
        """Test that trailing slashes are stripped from backend URL."""
        client = BackendAPIClient(
            backend_url="http://backend:8080/api///",
            project_name="test-project",
        )
        assert client.backend_url == "http://backend:8080/api"

    @patch("urllib.request.urlopen")
    def test_list_sessions(self, mock_urlopen, monkeypatch):
        """Test listing sessions returns paginated response with items key."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {
                "items": [
                    {
                        "metadata": {"name": "session-1"},
                        "status": {"phase": "Running"},
                    },
                    {
                        "metadata": {"name": "session-2"},
                        "status": {"phase": "Stopped"},
                    },
                ],
                "totalCount": 2,
                "hasMore": False,
            }
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        result = client.list_sessions(include_completed=True)

        assert len(result["items"]) == 2
        assert result["items"][0]["metadata"]["name"] == "session-1"
        assert result["totalCount"] == 2

    @patch("urllib.request.urlopen")
    def test_list_sessions_filters_completed(self, mock_urlopen, monkeypatch):
        """Test that list_sessions filters out stopped/completed/failed sessions by default."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {
                "items": [
                    {
                        "metadata": {"name": "session-1"},
                        "status": {"phase": "Running"},
                    },
                    {
                        "metadata": {"name": "session-2"},
                        "status": {"phase": "Stopped"},
                    },
                    {
                        "metadata": {"name": "session-3"},
                        "status": {"phase": "Pending"},
                    },
                ],
                "totalCount": 3,
                "hasMore": False,
            }
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        result = client.list_sessions(include_completed=False)

        assert len(result["items"]) == 2
        assert result["items"][0]["metadata"]["name"] == "session-1"
        assert result["items"][1]["metadata"]["name"] == "session-3"

    @patch("urllib.request.urlopen")
    def test_list_sessions_filter_by_phase(self, mock_urlopen, monkeypatch):
        """Test filtering sessions by specific phase."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {
                "items": [
                    {
                        "metadata": {"name": "session-1"},
                        "status": {"phase": "Running"},
                    },
                    {
                        "metadata": {"name": "session-2"},
                        "status": {"phase": "Stopped"},
                    },
                ],
                "totalCount": 2,
                "hasMore": False,
            }
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        result = client.list_sessions(phase="Running")

        assert len(result["items"]) == 1
        assert result["items"][0]["metadata"]["name"] == "session-1"

    @patch("urllib.request.urlopen")
    def test_list_sessions_with_search(self, mock_urlopen, monkeypatch):
        """Test that search param is passed as query parameter."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {"items": [], "totalCount": 0, "hasMore": False}
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        client.list_sessions(search="my-task")

        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        assert "search=my-task" in request.full_url

    @patch("urllib.request.urlopen")
    def test_get_session(self, mock_urlopen, monkeypatch):
        """Test getting a specific session."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {
                "name": "session-1",
                "phase": "Running",
                "spec": {"initialPrompt": "Test prompt"},
            }
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        session = client.get_session("session-1")

        assert session["name"] == "session-1"
        assert session["phase"] == "Running"

    @patch("urllib.request.urlopen")
    def test_create_session_minimal(self, mock_urlopen, monkeypatch):
        """Test creating a session with minimal params."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {"name": "new-session", "phase": "Pending"}
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        session = client.create_session("new-session")

        assert session["name"] == "new-session"

        # Verify the request
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        assert "/projects/test-project/agentic-sessions" in request.full_url
        assert request.method == "POST"

    @patch("urllib.request.urlopen")
    def test_create_session_with_all_params(self, mock_urlopen, monkeypatch):
        """Test creating a session with all params including workflow."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {"name": "new-session", "phase": "Running"}
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        repos = [{"url": "https://github.com/test/repo", "branch": "main"}]
        workflow = {
            "gitUrl": "https://github.com/org/workflows",
            "branch": "main",
            "path": "workflows/bugfix",
        }
        session = client.create_session(
            session_name="new-session",
            initial_prompt="Test prompt",
            display_name="Test Session",
            repos=repos,
            model="claude-sonnet-4-5",
            workflow=workflow,
        )

        assert session["name"] == "new-session"

        # Verify the request payload
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        payload = json.loads(request.data.decode("utf-8"))
        assert payload["sessionName"] == "new-session"
        assert payload["initialPrompt"] == "Test prompt"
        assert payload["displayName"] == "Test Session"
        assert payload["repos"] == repos
        assert payload["llmSettings"]["model"] == "claude-sonnet-4-5"
        assert payload["activeWorkflow"] == workflow

    @patch("urllib.request.urlopen")
    def test_stop_session(self, mock_urlopen, monkeypatch):
        """Test stopping a session."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps({"status": "ok"}).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        result = client.stop_session("session-1")

        assert result["status"] == "ok"

        # Verify the request
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        assert (
            "/projects/test-project/agentic-sessions/session-1/stop" in request.full_url
        )
        assert request.method == "POST"

    @patch("urllib.request.urlopen")
    def test_send_message(self, mock_urlopen, monkeypatch):
        """Test sending a message to a session."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {"runId": "run-123", "threadId": "thread-456"}
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        result = client.send_message("session-1", "Hello agent")

        assert result["runId"] == "run-123"
        assert result["threadId"] == "thread-456"

        # Verify the request
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        assert (
            "/projects/test-project/agentic-sessions/session-1/agui/run"
            in request.full_url
        )
        payload = json.loads(request.data.decode("utf-8"))
        assert payload["messages"][0]["content"] == "Hello agent"
        assert payload["messages"][0]["role"] == "user"
        assert "id" in payload["messages"][0]  # AG-UI requires message ID

    @patch("urllib.request.urlopen")
    def test_send_message_with_thread_id(self, mock_urlopen, monkeypatch):
        """Test sending a message with a thread ID."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {"runId": "run-123", "threadId": "thread-456"}
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        result = client.send_message("session-1", "Hello agent", thread_id="thread-456")

        assert result["runId"] == "run-123"

        # Verify thread ID in payload
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        payload = json.loads(request.data.decode("utf-8"))
        assert payload["threadId"] == "thread-456"

    @patch("urllib.request.urlopen")
    def test_http_error_handling(self, mock_urlopen, monkeypatch):
        """Test that HTTP errors are properly raised."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_error = HTTPError(
            "http://backend:8080/api/sessions",
            404,
            "Not Found",
            {},
            MagicMock(read=lambda: b'{"error": "Not found"}'),
        )
        mock_urlopen.side_effect = mock_error

        client = BackendAPIClient()
        with pytest.raises(HTTPError) as exc_info:
            client.get_session("nonexistent")

        assert exc_info.value.code == 404

    @patch("urllib.request.urlopen")
    def test_auth_header_included(self, mock_urlopen, monkeypatch):
        """Test that the Authorization header is included when BOT_TOKEN is set."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.setenv("BOT_TOKEN", "secret-token")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {"items": [], "totalCount": 0, "hasMore": False}
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient()
        client.list_sessions()

        # Verify Authorization header
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        assert request.headers["Authorization"] == "Bearer secret-token"

    @patch("urllib.request.urlopen")
    def test_no_auth_header_when_token_missing(self, mock_urlopen, monkeypatch):
        """Test that no Authorization header is sent when BOT_TOKEN is not set."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.delenv("BOT_TOKEN", raising=False)

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {"items": [], "totalCount": 0, "hasMore": False}
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        client = BackendAPIClient(bot_token="")
        client.list_sessions()

        # Verify no Authorization header
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        assert "Authorization" not in request.headers


class TestBackendMCPTools:
    """Test the backend MCP tool integration."""

    def test_create_backend_tools_returns_list(self):
        """Test that create_backend_mcp_tools returns a list of tools."""
        from ambient_runner.bridges.claude.backend_tools import (
            create_backend_mcp_tools,
        )

        # Mock the sdk_tool decorator (decorator factory pattern)
        def mock_tool(name, description, schema):
            def decorator(func):
                func._tool_name = name
                func._tool_description = description
                func._tool_schema = schema
                return func

            return decorator

        # Create with valid client
        client = BackendAPIClient(
            backend_url="http://backend:8080/api",
            project_name="test-project",
        )
        tools = create_backend_mcp_tools(sdk_tool_decorator=mock_tool, client=client)

        assert isinstance(tools, list)
        assert len(tools) == 9

        # Verify tool names
        tool_names = [t._tool_name for t in tools]
        assert "acp_list_sessions" in tool_names
        assert "acp_get_session" in tool_names
        assert "acp_create_session" in tool_names
        assert "acp_stop_session" in tool_names
        assert "acp_send_message" in tool_names
        assert "acp_get_session_status" in tool_names
        assert "acp_restart_session" in tool_names
        assert "acp_list_workflows" in tool_names
        assert "acp_get_api_reference" in tool_names

    def test_create_backend_tools_returns_empty_when_no_env(self, monkeypatch):
        """Test that create_backend_mcp_tools returns empty list when env vars missing."""
        from ambient_runner.bridges.claude.backend_tools import (
            create_backend_mcp_tools,
        )

        monkeypatch.delenv("BACKEND_API_URL", raising=False)
        monkeypatch.delenv("PROJECT_NAME", raising=False)
        monkeypatch.delenv("AGENTIC_SESSION_NAMESPACE", raising=False)

        def mock_tool(name, description, schema):
            def decorator(func):
                return func

            return decorator

        tools = create_backend_mcp_tools(sdk_tool_decorator=mock_tool)

        assert tools == []

    @patch("urllib.request.urlopen")
    @pytest.mark.asyncio
    async def test_tool_execution_list_sessions(self, mock_urlopen, monkeypatch):
        """Test executing the acp_list_sessions tool."""
        from ambient_runner.bridges.claude.backend_tools import (
            create_backend_mcp_tools,
        )

        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {
                "items": [
                    {
                        "metadata": {
                            "name": "session-1",
                            "creationTimestamp": "2026-01-01T00:00:00Z",
                        },
                        "spec": {"displayName": "Test Session"},
                        "status": {"phase": "Running", "agentStatus": "working"},
                    }
                ],
                "totalCount": 1,
                "hasMore": False,
            }
        ).encode("utf-8")
        mock_response.__enter__.return_value = mock_response
        mock_urlopen.return_value = mock_response

        def mock_tool(name, description, schema):
            def decorator(func):
                return func

            return decorator

        client = BackendAPIClient()
        tools = create_backend_mcp_tools(sdk_tool_decorator=mock_tool, client=client)

        list_tool = (
            next(
                t
                for t in tools
                if hasattr(t, "_tool_name") and t._tool_name == "acp_list_sessions"
            )
            if any(hasattr(t, "_tool_name") for t in tools)
            else next(t for t in tools if t.__name__ == "acp_list_sessions")
        )
        result = await list_tool({"include_completed": False})

        # Extract text from content array
        assert "content" in result
        assert len(result["content"]) > 0
        result_text = result["content"][0]["text"]
        result_data = json.loads(result_text)
        assert result_data["success"] is True
        assert result_data["count"] == 1
        assert result_data["sessions"][0]["name"] == "session-1"
        assert result_data["sessions"][0]["phase"] == "Running"
        assert result_data["sessions"][0]["agentStatus"] == "working"

    @pytest.mark.asyncio
    async def test_api_reference_tool(self, monkeypatch):
        """Test the acp_get_api_reference tool returns documentation."""
        from ambient_runner.bridges.claude.backend_tools import (
            create_backend_mcp_tools,
        )

        monkeypatch.setenv("BACKEND_API_URL", "http://test-backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.setenv("BOT_TOKEN", "test-token-123")

        def mock_tool(name, description, schema):
            def decorator(func):
                return func

            return decorator

        client = BackendAPIClient()
        tools = create_backend_mcp_tools(sdk_tool_decorator=mock_tool, client=client)

        api_ref_tool = (
            next(
                t
                for t in tools
                if hasattr(t, "_tool_name") and t._tool_name == "acp_get_api_reference"
            )
            if any(hasattr(t, "_tool_name") for t in tools)
            else next(t for t in tools if t.__name__ == "acp_get_api_reference")
        )
        result = await api_ref_tool({})

        # Extract text from content array
        assert "content" in result
        assert len(result["content"]) > 0
        docs = result["content"][0]["text"]

        # Verify it returns markdown documentation
        assert isinstance(docs, str)
        assert "# Ambient Code Platform API Reference" in docs
        assert "Base URL" in docs
        assert "http://test-backend:8080/api" in docs
        assert "test-project" in docs

        # Verify all endpoints are documented
        assert "List Sessions" in docs
        assert "GET" in docs
        assert "/api/projects/{projectName}/agentic-sessions" in docs
        assert "Create Session" in docs
        assert "POST" in docs
        assert "Stop Session" in docs
        assert "Send Message" in docs

        # Verify code examples are included
        assert "fetch(" in docs
        assert "const headers" in docs
        assert "Authorization" in docs
        assert "Bearer" in docs

        # Verify HTML example
        assert "<button" in docs
        assert "onclick" in docs

        # Verify Python example
        assert "import requests" in docs
        assert "def list_sessions()" in docs
