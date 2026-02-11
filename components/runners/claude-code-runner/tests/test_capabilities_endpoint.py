"""Unit tests for the capabilities endpoint."""

from unittest.mock import MagicMock, patch

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from endpoints.capabilities import router


@pytest.fixture
def client():
    """Create a test client with the capabilities router."""
    app = FastAPI()
    app.include_router(router)
    return TestClient(app)


class TestCapabilitiesEndpoint:
    """Test GET /capabilities response shape and values."""

    @patch("endpoints.capabilities.state")
    def test_returns_expected_fields(self, mock_state, client):
        mock_state._obs = None
        mock_state._configured_model = "claude-sonnet-4-5"
        mock_state.context = MagicMock()
        mock_state.context.session_id = "test-session"

        resp = client.get("/capabilities")
        assert resp.status_code == 200
        data = resp.json()

        # Required fields
        assert data["framework"] == "claude-agent-sdk"
        assert isinstance(data["agent_features"], list)
        assert isinstance(data["platform_features"], list)
        assert isinstance(data["file_system"], bool)
        assert isinstance(data["mcp"], bool)
        assert isinstance(data["session_persistence"], bool)

    @patch("endpoints.capabilities.state")
    def test_agent_features_list(self, mock_state, client):
        mock_state._obs = None
        mock_state._configured_model = ""
        mock_state.context = None

        resp = client.get("/capabilities")
        data = resp.json()
        assert "agentic_chat" in data["agent_features"]
        assert "thinking" in data["agent_features"]

    @patch("endpoints.capabilities.state")
    def test_platform_features_list(self, mock_state, client):
        mock_state._obs = None
        mock_state._configured_model = ""
        mock_state.context = None

        resp = client.get("/capabilities")
        data = resp.json()
        assert "repos" in data["platform_features"]
        assert "workflows" in data["platform_features"]
        assert "feedback" in data["platform_features"]
        assert "mcp_diagnostics" in data["platform_features"]

    @patch("endpoints.capabilities.state")
    def test_tracing_langfuse_when_obs_present(self, mock_state, client):
        mock_obs = MagicMock()
        mock_obs.langfuse_client = MagicMock()  # truthy
        mock_state._obs = mock_obs
        mock_state._configured_model = ""
        mock_state.context = None

        resp = client.get("/capabilities")
        assert resp.json()["tracing"] == "langfuse"

    @patch("endpoints.capabilities.state")
    def test_tracing_none_when_no_obs(self, mock_state, client):
        mock_state._obs = None
        mock_state._configured_model = ""
        mock_state.context = None

        resp = client.get("/capabilities")
        assert resp.json()["tracing"] is None

    @patch("endpoints.capabilities.state")
    def test_tracing_none_when_obs_has_no_client(self, mock_state, client):
        mock_obs = MagicMock()
        mock_obs.langfuse_client = None
        mock_state._obs = mock_obs
        mock_state._configured_model = ""
        mock_state.context = None

        resp = client.get("/capabilities")
        assert resp.json()["tracing"] is None

    @patch("endpoints.capabilities.state")
    def test_model_returned(self, mock_state, client):
        mock_state._obs = None
        mock_state._configured_model = "claude-4-opus"
        mock_state.context = None

        resp = client.get("/capabilities")
        assert resp.json()["model"] == "claude-4-opus"

    @patch("endpoints.capabilities.state")
    def test_model_none_when_empty(self, mock_state, client):
        mock_state._obs = None
        mock_state._configured_model = ""
        mock_state.context = None

        resp = client.get("/capabilities")
        assert resp.json()["model"] is None

    @patch("endpoints.capabilities.state")
    def test_session_id_returned(self, mock_state, client):
        mock_state._obs = None
        mock_state._configured_model = ""
        mock_state.context = MagicMock()
        mock_state.context.session_id = "sess-xyz"

        resp = client.get("/capabilities")
        assert resp.json()["session_id"] == "sess-xyz"

    @patch("endpoints.capabilities.state")
    def test_session_id_none_when_no_context(self, mock_state, client):
        mock_state._obs = None
        mock_state._configured_model = ""
        mock_state.context = None

        resp = client.get("/capabilities")
        assert resp.json()["session_id"] is None
