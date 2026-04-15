"""Unit tests for project intelligence memory tools and API client."""

import asyncio
import json
from unittest.mock import MagicMock, patch

import pytest

from ambient_runner.tools.intelligence_api import IntelligenceAPIClient


# ---------------------------------------------------------------------------
# IntelligenceAPIClient tests
# ---------------------------------------------------------------------------


class TestIntelligenceAPIClient:
    """Test the IntelligenceAPIClient class."""

    def test_init_from_env(self, monkeypatch):
        monkeypatch.setenv("API_SERVER_URL", "http://api-server:8000")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.setenv("BOT_TOKEN", "test-token")

        client = IntelligenceAPIClient()
        assert client.api_server_url == "http://api-server:8000"
        assert client.project_id == "test-project"

    def test_init_falls_back_to_backend_url(self, monkeypatch):
        monkeypatch.delenv("API_SERVER_URL", raising=False)
        monkeypatch.setenv("BACKEND_API_URL", "http://backend:8080/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        client = IntelligenceAPIClient()
        # Trailing /api is stripped to avoid double /api/api/ in paths
        assert client.api_server_url == "http://backend:8080"

    def test_init_from_params(self):
        client = IntelligenceAPIClient(
            api_server_url="http://custom:9000",
            project_id="my-project",
            bot_token="my-token",
        )
        assert client.api_server_url == "http://custom:9000"
        assert client.project_id == "my-project"

    def test_init_missing_url_raises(self, monkeypatch):
        monkeypatch.delenv("API_SERVER_URL", raising=False)
        monkeypatch.delenv("BACKEND_API_URL", raising=False)
        with pytest.raises(ValueError, match="API_SERVER_URL"):
            IntelligenceAPIClient()

    def test_url_trailing_slash_stripped(self):
        client = IntelligenceAPIClient(
            api_server_url="http://api:8000///",
            project_id="p",
        )
        assert client.api_server_url == "http://api:8000"

    def test_strips_trailing_api_from_backend_url(self):
        """BACKEND_API_URL ends with /api; client should strip it to avoid /api/api/."""
        client = IntelligenceAPIClient(
            api_server_url="http://backend:8080/api",
            project_id="p",
        )
        assert client.api_server_url == "http://backend:8080"

    def test_does_not_strip_api_from_api_server_url(self):
        """API_SERVER_URL that doesn't end with /api should not be stripped."""
        client = IntelligenceAPIClient(
            api_server_url="http://api-server:8000",
            project_id="p",
        )
        assert client.api_server_url == "http://api-server:8000"

    @patch("urllib.request.urlopen")
    def test_lookup_intelligence_found(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"id": "intel-1", "repo_url": "https://github.com/org/repo", "language": "go"}
        ).encode()
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        client = IntelligenceAPIClient(
            api_server_url="http://api:8000", project_id="proj"
        )
        result = client.lookup_intelligence("https://github.com/org/repo")

        assert result is not None
        assert result["id"] == "intel-1"
        assert result["language"] == "go"

    @patch("urllib.request.urlopen")
    def test_lookup_intelligence_not_found(self, mock_urlopen):
        from urllib.error import HTTPError

        mock_urlopen.side_effect = HTTPError(
            url="http://api:8000/lookup",
            code=404,
            msg="Not Found",
            hdrs=None,
            fp=MagicMock(read=MagicMock(return_value=b'{"kind":"Error"}')),
        )

        client = IntelligenceAPIClient(
            api_server_url="http://api:8000", project_id="proj"
        )
        result = client.lookup_intelligence("https://github.com/org/nonexistent")
        assert result is None

    @patch("urllib.request.urlopen")
    def test_intelligence_exists_true(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"id": "intel-1"}).encode()
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        client = IntelligenceAPIClient(
            api_server_url="http://api:8000", project_id="proj"
        )
        assert client.intelligence_exists("https://github.com/org/repo") is True

    @patch("urllib.request.urlopen")
    def test_intelligence_exists_false(self, mock_urlopen):
        from urllib.error import HTTPError

        mock_urlopen.side_effect = HTTPError(
            url="", code=404, msg="", hdrs=None,
            fp=MagicMock(read=MagicMock(return_value=b"")),
        )

        client = IntelligenceAPIClient(
            api_server_url="http://api:8000", project_id="proj"
        )
        assert client.intelligence_exists("https://github.com/org/repo") is False

    @patch("urllib.request.urlopen")
    def test_create_finding(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"id": "finding-1", "title": "Bug found"}
        ).encode()
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        client = IntelligenceAPIClient(
            api_server_url="http://api:8000", project_id="proj"
        )
        result = client.create_finding({"title": "Bug found"})
        assert result["id"] == "finding-1"

    @patch("urllib.request.urlopen")
    def test_list_findings_with_filters(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"items": [{"id": "f1"}], "total": 1}
        ).encode()
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        client = IntelligenceAPIClient(
            api_server_url="http://api:8000", project_id="proj"
        )
        result = client.list_findings(
            "intel-1", file_path="sessions.go", category="caveat"
        )
        assert result["total"] == 1

        # Verify the URL contains the search filters
        call_args = mock_urlopen.call_args
        req = call_args[0][0]
        assert "sessions.go" in req.full_url
        assert "caveat" in req.full_url


# ---------------------------------------------------------------------------
# MCP tool function tests
# ---------------------------------------------------------------------------


def _mock_sdk_tool(name, description, schema):
    """Mock decorator that just returns the function."""
    def decorator(fn):
        fn._tool_name = name
        return fn
    return decorator


class TestMemoryQueryTool:
    """Test the memory_query MCP tool."""

    def _create_tools(self, client):
        from ambient_runner.bridges.claude.memory_tools import create_memory_mcp_tools
        return create_memory_mcp_tools(
            sdk_tool_decorator=_mock_sdk_tool, client=client
        )

    def test_query_no_intelligence(self):
        client = MagicMock()
        client.lookup_intelligence.return_value = None

        tools = self._create_tools(client)
        query_tool = tools[0]

        result = asyncio.run(query_tool({"repo_url": "https://github.com/org/repo"}))
        data = json.loads(result["content"][0]["text"])
        assert data["found"] is False

    def test_query_with_intelligence(self):
        client = MagicMock()
        client.lookup_intelligence.return_value = {
            "id": "intel-1",
            "summary": "Go backend",
            "language": "go",
        }

        tools = self._create_tools(client)
        query_tool = tools[0]

        result = asyncio.run(query_tool({"repo_url": "https://github.com/org/repo"}))
        data = json.loads(result["content"][0]["text"])
        assert data["found"] is True
        assert data["intelligence"]["language"] == "go"

    def test_query_with_file_filter(self):
        client = MagicMock()
        client.lookup_intelligence.return_value = {"id": "intel-1"}
        client.list_findings.return_value = {
            "items": [{"title": "Bug in sessions.go"}],
            "total": 1,
        }

        tools = self._create_tools(client)
        query_tool = tools[0]

        result = asyncio.run(
            query_tool(
                {"repo_url": "https://github.com/org/repo", "file_path": "sessions.go"}
            )
        )
        data = json.loads(result["content"][0]["text"])
        assert data["found"] is True
        assert data["findings_count"] == 1
        client.list_findings.assert_called_once_with(
            "intel-1", file_path="sessions.go", category=None
        )

    def test_query_error_returns_error_response(self):
        client = MagicMock()
        client.lookup_intelligence.side_effect = Exception("connection refused")

        tools = self._create_tools(client)
        query_tool = tools[0]

        result = asyncio.run(query_tool({"repo_url": "https://github.com/org/repo"}))
        assert result["isError"] is True
        data = json.loads(result["content"][0]["text"])
        assert "connection refused" in data["error"]


class TestMemoryStoreTool:
    """Test the memory_store MCP tool."""

    def _create_tools(self, client):
        from ambient_runner.bridges.claude.memory_tools import create_memory_mcp_tools
        return create_memory_mcp_tools(
            sdk_tool_decorator=_mock_sdk_tool, client=client
        )

    def test_store_no_intelligence_record(self):
        client = MagicMock()
        client.lookup_intelligence.return_value = None

        tools = self._create_tools(client)
        store_tool = tools[1]

        result = asyncio.run(
            store_tool(
                {
                    "repo_url": "https://github.com/org/repo",
                    "file_path": "main.go",
                    "category": "caveat",
                    "title": "Not thread-safe",
                    "body": "Race condition",
                }
            )
        )
        data = json.loads(result["content"][0]["text"])
        assert data["stored"] is False
        client.create_finding.assert_not_called()

    def test_store_creates_finding(self):
        client = MagicMock()
        client.lookup_intelligence.return_value = {"id": "intel-1"}
        client.create_finding.return_value = {"id": "finding-1"}

        tools = self._create_tools(client)
        store_tool = tools[1]

        result = asyncio.run(
            store_tool(
                {
                    "repo_url": "https://github.com/org/repo",
                    "file_path": "main.go",
                    "category": "caveat",
                    "title": "Not thread-safe",
                    "body": "Race condition on shared map",
                    "severity": "critical",
                    "confidence": 0.9,
                }
            )
        )
        data = json.loads(result["content"][0]["text"])
        assert data["stored"] is True
        assert data["finding_id"] == "finding-1"

        call_data = client.create_finding.call_args[0][0]
        assert call_data["intelligence_id"] == "intel-1"
        assert call_data["file_path"] == "main.go"
        assert call_data["category"] == "caveat"
        assert call_data["severity"] == "critical"
        assert call_data["source_type"] == "agent_analysis"


class TestMemoryWarnTool:
    """Test the memory_warn MCP tool."""

    def _create_tools(self, client):
        from ambient_runner.bridges.claude.memory_tools import create_memory_mcp_tools
        return create_memory_mcp_tools(
            sdk_tool_decorator=_mock_sdk_tool, client=client
        )

    def test_warn_no_repo_url(self):
        client = MagicMock()

        tools = self._create_tools(client)
        warn_tool = tools[2]

        result = asyncio.run(warn_tool({"file_path": "main.go"}))
        data = json.loads(result["content"][0]["text"])
        assert data["warnings"] == []
        assert "repo_url" in data["message"]

    def test_warn_no_intelligence(self):
        client = MagicMock()
        client.lookup_intelligence.return_value = None

        tools = self._create_tools(client)
        warn_tool = tools[2]

        result = asyncio.run(
            warn_tool(
                {"file_path": "main.go", "repo_url": "https://github.com/org/repo"}
            )
        )
        data = json.loads(result["content"][0]["text"])
        assert data["warnings"] == []

    def test_warn_no_findings(self):
        client = MagicMock()
        client.lookup_intelligence.return_value = {"id": "intel-1"}
        client.list_findings.return_value = {"items": []}

        tools = self._create_tools(client)
        warn_tool = tools[2]

        result = asyncio.run(
            warn_tool(
                {"file_path": "clean.go", "repo_url": "https://github.com/org/repo"}
            )
        )
        data = json.loads(result["content"][0]["text"])
        assert data["warnings"] == []
        assert "No known findings" in data["message"]

    def test_warn_returns_findings(self):
        client = MagicMock()
        client.lookup_intelligence.return_value = {"id": "intel-1"}
        client.list_findings.return_value = {
            "items": [
                {"title": "Race condition", "severity": "critical"},
                {"title": "Missing error check", "severity": "warning"},
            ]
        }

        tools = self._create_tools(client)
        warn_tool = tools[2]

        result = asyncio.run(
            warn_tool(
                {"file_path": "sessions.go", "repo_url": "https://github.com/org/repo"}
            )
        )
        data = json.loads(result["content"][0]["text"])
        assert data["count"] == 2
        assert len(data["warnings"]) == 2
        assert "2 known finding(s)" in data["message"]


class TestCreateMemoryMcpTools:
    """Test the tool factory function."""

    def test_returns_empty_when_no_client(self, monkeypatch):
        monkeypatch.delenv("API_SERVER_URL", raising=False)
        monkeypatch.delenv("BACKEND_API_URL", raising=False)

        from ambient_runner.bridges.claude.memory_tools import create_memory_mcp_tools
        tools = create_memory_mcp_tools(sdk_tool_decorator=_mock_sdk_tool)
        assert tools == []

    def test_returns_three_tools(self):
        client = MagicMock(spec=IntelligenceAPIClient)
        from ambient_runner.bridges.claude.memory_tools import create_memory_mcp_tools
        tools = create_memory_mcp_tools(
            sdk_tool_decorator=_mock_sdk_tool, client=client
        )
        assert len(tools) == 3
        assert tools[0]._tool_name == "memory_query"
        assert tools[1]._tool_name == "memory_store"
        assert tools[2]._tool_name == "memory_warn"
