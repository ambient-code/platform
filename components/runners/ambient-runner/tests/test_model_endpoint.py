"""Unit tests for the POST /model endpoint."""

import asyncio
from unittest.mock import MagicMock, patch

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from ambient_runner.endpoints.model import router


def _make_mock_bridge(
    *,
    session_id="test-session",
    has_context=True,
    lock_locked=False,
    has_worker=True,
):
    """Create a mock bridge with configurable session manager and context."""
    bridge = MagicMock()

    if has_context:
        bridge.context = MagicMock()
        bridge.context.session_id = session_id
    else:
        bridge.context = None

    # Session manager with per-thread lock
    lock = asyncio.Lock()
    if lock_locked:
        # Simulate a locked lock by acquiring it (non-async safe for test setup)
        lock._locked = True

    session_manager = MagicMock()
    session_manager.get_lock.return_value = lock

    if has_worker:
        worker = MagicMock()
        worker._between_run_queue = asyncio.Queue()
        session_manager.get_existing.return_value = worker
    else:
        session_manager.get_existing.return_value = None

    bridge._session_manager = session_manager

    return bridge


@pytest.fixture(autouse=True)
def _reset_model_change_lock():
    """Ensure the module-level _model_change_lock is released between tests."""
    from ambient_runner.endpoints import model as mod

    # Replace with a fresh lock so no test leaks state
    mod._model_change_lock = asyncio.Lock()
    yield


@pytest.fixture
def make_client():
    """Factory to create a test client with a mock bridge."""

    def _factory(*, env_model="claude-sonnet-4-5", **bridge_kwargs):
        app = FastAPI()
        app.state.bridge = _make_mock_bridge(**bridge_kwargs)
        app.include_router(router)
        with patch.dict("os.environ", {"LLM_MODEL": env_model}):
            client = TestClient(app)
        return client, app.state.bridge

    return _factory


class TestModelEndpoint:
    """Test POST /model request handling."""

    def test_success_switches_model(self, make_client):
        """POST /model with a valid new model returns 200 with model and previousModel."""
        with patch.dict("os.environ", {"LLM_MODEL": "claude-sonnet-4-5"}):
            client, bridge = make_client(env_model="claude-sonnet-4-5")
            resp = client.post("/model", json={"model": "claude-opus-4"})

        assert resp.status_code == 200
        data = resp.json()
        assert data["message"] == "Model switched"
        assert data["model"] == "claude-opus-4"
        assert data["previousModel"] == "claude-sonnet-4-5"
        bridge.mark_dirty.assert_called_once()

    def test_empty_model_returns_400(self, make_client):
        """POST /model with an empty model string returns 400."""
        client, _ = make_client()
        resp = client.post("/model", json={"model": ""})

        assert resp.status_code == 400
        assert "model is required" in resp.json()["detail"]

    def test_whitespace_only_model_returns_400(self, make_client):
        """POST /model with whitespace-only model returns 400."""
        client, _ = make_client()
        resp = client.post("/model", json={"model": "   "})

        assert resp.status_code == 400
        assert "model is required" in resp.json()["detail"]

    def test_missing_model_field_returns_400(self, make_client):
        """POST /model with no model field in body returns 400."""
        client, _ = make_client()
        resp = client.post("/model", json={})

        assert resp.status_code == 400
        assert "model is required" in resp.json()["detail"]

    def test_same_model_returns_unchanged(self, make_client):
        """POST /model with same model as current returns 200 with 'Model unchanged'."""
        with patch.dict("os.environ", {"LLM_MODEL": "claude-sonnet-4-5"}):
            client, bridge = make_client(env_model="claude-sonnet-4-5")
            resp = client.post("/model", json={"model": "claude-sonnet-4-5"})

        assert resp.status_code == 200
        data = resp.json()
        assert data["message"] == "Model unchanged"
        assert data["model"] == "claude-sonnet-4-5"
        assert "previousModel" not in data
        bridge.mark_dirty.assert_not_called()

    def test_context_not_initialized_returns_503(self, make_client):
        """POST /model when bridge.context is None returns 503."""
        client, _ = make_client(has_context=False)
        resp = client.post("/model", json={"model": "claude-opus-4"})

        assert resp.status_code == 503
        assert "Context not initialized" in resp.json()["detail"]

    def test_locked_run_returns_422(self, make_client):
        """POST /model while agent is mid-generation returns 422."""
        client, _ = make_client(lock_locked=True)
        with patch.dict("os.environ", {"LLM_MODEL": "claude-sonnet-4-5"}):
            resp = client.post("/model", json={"model": "claude-opus-4"})

        assert resp.status_code == 422
        assert "Cannot switch model" in resp.json()["detail"]

    def test_updates_env_variable(self, make_client):
        """POST /model updates the LLM_MODEL environment variable."""
        with patch.dict("os.environ", {"LLM_MODEL": "claude-sonnet-4-5"}):
            client, _ = make_client(env_model="claude-sonnet-4-5")
            resp = client.post("/model", json={"model": "claude-opus-4"})
            import os

            assert resp.status_code == 200
            assert os.environ["LLM_MODEL"] == "claude-opus-4"

    def test_emits_event_to_worker_queue(self, make_client):
        """POST /model emits a model_switched event to the between-run queue."""
        with patch.dict("os.environ", {"LLM_MODEL": "claude-sonnet-4-5"}):
            client, bridge = make_client(env_model="claude-sonnet-4-5")
            resp = client.post("/model", json={"model": "claude-opus-4"})

        assert resp.status_code == 200
        worker = bridge._session_manager.get_existing.return_value
        assert not worker._between_run_queue.empty()
        event = worker._between_run_queue.get_nowait()
        assert event.name == "ambient:model_switched"
        assert event.value["newModel"] == "claude-opus-4"
        assert event.value["previousModel"] == "claude-sonnet-4-5"
