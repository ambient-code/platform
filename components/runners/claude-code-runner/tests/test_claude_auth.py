"""Unit tests for ambient_runner.bridges.claude.auth — Vertex AI and API key setup."""

import asyncio
import os
import tempfile
from pathlib import Path
from unittest.mock import MagicMock

import pytest

from ambient_runner.bridges.claude.auth import (
    VERTEX_MODEL_MAP,
    map_to_vertex_model,
    setup_sdk_authentication,
    setup_vertex_credentials,
)
from ambient_runner.platform.context import RunnerContext


# ---------------------------------------------------------------------------
# Helper
# ---------------------------------------------------------------------------


def _make_context(**env_overrides) -> RunnerContext:
    """Create a RunnerContext with specific env vars (avoids chdir)."""
    ctx = object.__new__(RunnerContext)
    ctx.session_id = "test-session"
    ctx.workspace_path = "/tmp/test"
    ctx.metadata = {}
    ctx.environment = {**env_overrides}
    return ctx


# ---------------------------------------------------------------------------
# map_to_vertex_model
# ---------------------------------------------------------------------------


class TestMapToVertexModel:
    def test_known_models_map_correctly(self):
        for api_name, vertex_name in VERTEX_MODEL_MAP.items():
            assert map_to_vertex_model(api_name) == vertex_name
            assert "@" in vertex_name

    def test_unknown_model_passthrough(self):
        assert map_to_vertex_model("my-custom-model") == "my-custom-model"

    def test_empty_string_passthrough(self):
        assert map_to_vertex_model("") == ""


# ---------------------------------------------------------------------------
# setup_vertex_credentials
# ---------------------------------------------------------------------------


class TestSetupVertexCredentials:
    @pytest.mark.asyncio
    async def test_success_with_valid_credentials(self):
        with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
            cred_path = f.name

        try:
            ctx = _make_context(
                GOOGLE_APPLICATION_CREDENTIALS=cred_path,
                ANTHROPIC_VERTEX_PROJECT_ID="my-project",
                CLOUD_ML_REGION="us-central1",
            )
            result = await setup_vertex_credentials(ctx)
            assert result["credentials_path"] == cred_path
            assert result["project_id"] == "my-project"
            assert result["region"] == "us-central1"
        finally:
            os.unlink(cred_path)

    @pytest.mark.asyncio
    async def test_error_missing_credentials_path(self):
        ctx = _make_context(
            GOOGLE_APPLICATION_CREDENTIALS="",
            ANTHROPIC_VERTEX_PROJECT_ID="proj",
            CLOUD_ML_REGION="us-central1",
        )
        with pytest.raises(RuntimeError, match="GOOGLE_APPLICATION_CREDENTIALS"):
            await setup_vertex_credentials(ctx)

    @pytest.mark.asyncio
    async def test_error_missing_project_id(self):
        with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
            cred_path = f.name
        try:
            ctx = _make_context(
                GOOGLE_APPLICATION_CREDENTIALS=cred_path,
                ANTHROPIC_VERTEX_PROJECT_ID="",
                CLOUD_ML_REGION="us-central1",
            )
            with pytest.raises(RuntimeError, match="ANTHROPIC_VERTEX_PROJECT_ID"):
                await setup_vertex_credentials(ctx)
        finally:
            os.unlink(cred_path)

    @pytest.mark.asyncio
    async def test_error_missing_region(self):
        with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
            cred_path = f.name
        try:
            ctx = _make_context(
                GOOGLE_APPLICATION_CREDENTIALS=cred_path,
                ANTHROPIC_VERTEX_PROJECT_ID="proj",
                CLOUD_ML_REGION="",
            )
            with pytest.raises(RuntimeError, match="CLOUD_ML_REGION"):
                await setup_vertex_credentials(ctx)
        finally:
            os.unlink(cred_path)

    @pytest.mark.asyncio
    async def test_error_file_does_not_exist(self):
        ctx = _make_context(
            GOOGLE_APPLICATION_CREDENTIALS="/nonexistent/path.json",
            ANTHROPIC_VERTEX_PROJECT_ID="proj",
            CLOUD_ML_REGION="us-central1",
        )
        with pytest.raises(RuntimeError, match="not found"):
            await setup_vertex_credentials(ctx)


# ---------------------------------------------------------------------------
# setup_sdk_authentication
# ---------------------------------------------------------------------------


class TestSetupSdkAuthentication:
    @pytest.mark.asyncio
    async def test_anthropic_api_key(self, monkeypatch):
        monkeypatch.delenv("ANTHROPIC_API_KEY", raising=False)
        ctx = _make_context(ANTHROPIC_API_KEY="sk-test-key")
        api_key, use_vertex, model = await setup_sdk_authentication(ctx)
        assert api_key == "sk-test-key"
        assert use_vertex is False
        assert model  # should have a default model

    @pytest.mark.asyncio
    async def test_no_auth_raises(self):
        ctx = _make_context(ANTHROPIC_API_KEY="", CLAUDE_CODE_USE_VERTEX="")
        with pytest.raises(RuntimeError, match="ANTHROPIC_API_KEY"):
            await setup_sdk_authentication(ctx)

    @pytest.mark.asyncio
    async def test_custom_model(self, monkeypatch):
        monkeypatch.delenv("ANTHROPIC_API_KEY", raising=False)
        ctx = _make_context(ANTHROPIC_API_KEY="sk-key", LLM_MODEL="claude-opus-4-5")
        _, _, model = await setup_sdk_authentication(ctx)
        assert model == "claude-opus-4-5"

    @pytest.mark.asyncio
    async def test_default_model_when_none_specified(self, monkeypatch):
        monkeypatch.delenv("ANTHROPIC_API_KEY", raising=False)
        ctx = _make_context(ANTHROPIC_API_KEY="sk-key")
        _, _, model = await setup_sdk_authentication(ctx)
        assert model == "claude-sonnet-4-5"
        assert "@" not in model  # no Vertex date suffix for API key auth
