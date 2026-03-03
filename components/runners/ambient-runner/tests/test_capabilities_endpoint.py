"""Unit tests for the capabilities endpoint."""

from unittest.mock import MagicMock

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from ambient_runner.bridge import FrameworkCapabilities
from ambient_runner.endpoints.capabilities import router


def _make_mock_bridge(
    *,
    tracing=None,
    configured_model="",
    session_id=None,
):
    """Create a mock bridge with the given capabilities."""
    bridge = MagicMock()
    bridge.capabilities.return_value = FrameworkCapabilities(
        framework="claude-agent-sdk",
        agent_features=[
            "agentic_chat",
            "backend_tool_rendering",
            "shared_state",
            "human_in_the_loop",
            "thinking",
        ],
        file_system=True,
        mcp=True,
        tracing=tracing,
        session_persistence=True,
    )
    bridge.configured_model = configured_model
    if session_id is not None:
        bridge.context = MagicMock()
        bridge.context.session_id = session_id
    else:
        bridge.context = None
    return bridge


@pytest.fixture
def make_client():
    """Factory to create a test client with a mock bridge."""
    def _factory(**kwargs):
        app = FastAPI()
        app.state.bridge = _make_mock_bridge(**kwargs)
        app.include_router(router)
        return TestClient(app)
    return _factory


class TestCapabilitiesEndpoint:
    """Test GET /capabilities response shape and values."""

    def test_returns_expected_fields(self, make_client):
        client = make_client(configured_model="claude-sonnet-4-5", session_id="test-session")
        resp = client.get("/capabilities")
        assert resp.status_code == 200
        data = resp.json()
        assert data["framework"] == "claude-agent-sdk"
        assert isinstance(data["agent_features"], list)
        assert isinstance(data["platform_features"], list)
        assert isinstance(data["file_system"], bool)
        assert isinstance(data["mcp"], bool)
        assert isinstance(data["session_persistence"], bool)

    def test_agent_features_list(self, make_client):
        client = make_client()
        data = client.get("/capabilities").json()
        assert "agentic_chat" in data["agent_features"]
        assert "thinking" in data["agent_features"]

    def test_tracing_langfuse_when_configured(self, make_client):
        client = make_client(tracing="langfuse")
        assert client.get("/capabilities").json()["tracing"] == "langfuse"

    def test_tracing_none_when_not_configured(self, make_client):
        client = make_client(tracing=None)
        assert client.get("/capabilities").json()["tracing"] is None

    def test_model_returned(self, make_client):
        client = make_client(configured_model="claude-4-opus")
        assert client.get("/capabilities").json()["model"] == "claude-4-opus"

    def test_model_none_when_empty(self, make_client):
        client = make_client(configured_model="")
        assert client.get("/capabilities").json()["model"] is None

    def test_session_id_returned(self, make_client):
        client = make_client(session_id="sess-xyz")
        assert client.get("/capabilities").json()["session_id"] == "sess-xyz"

    def test_session_id_none_when_no_context(self, make_client):
        client = make_client(session_id=None)
        assert client.get("/capabilities").json()["session_id"] is None
