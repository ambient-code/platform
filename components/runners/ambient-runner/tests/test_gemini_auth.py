"""Tests for Gemini CLI authentication setup."""

import warnings

import pytest

from ambient_runner.bridges.gemini_cli.auth import setup_gemini_cli_auth
from ambient_runner.platform.context import RunnerContext
from ambient_runner.platform.utils import is_vertex_enabled


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------


def _make_context(**env_overrides) -> RunnerContext:
    """Create a RunnerContext with a clean environment (only overrides)."""
    # Wipe keys that would leak from the host environment
    clean = {
        "GEMINI_API_KEY": "",
        "GOOGLE_API_KEY": "",
        "USE_VERTEX": "",
        "GEMINI_USE_VERTEX": "",
        "LLM_MODEL": "",
        "GOOGLE_CLOUD_PROJECT": "",
        "GOOGLE_CLOUD_LOCATION": "",
    }
    clean.update(env_overrides)
    return RunnerContext(session_id="s1", workspace_path="/workspace", environment=clean)


# ------------------------------------------------------------------
# _is_vertex_enabled()
# ------------------------------------------------------------------


class TestIsVertexEnabled:
    """Test the _is_vertex_enabled helper."""

    def test_returns_false_when_unset(self, monkeypatch):
        monkeypatch.delenv("USE_VERTEX", raising=False)
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)
        assert is_vertex_enabled(legacy_var="GEMINI_USE_VERTEX") is False

    @pytest.mark.parametrize("value", ["1", "true", "True", "yes", "YES"])
    def test_use_vertex_truthy_values(self, monkeypatch, value):
        monkeypatch.setenv("USE_VERTEX", value)
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)
        assert is_vertex_enabled(legacy_var="GEMINI_USE_VERTEX") is True

    def test_use_vertex_false_for_non_truthy(self, monkeypatch):
        monkeypatch.setenv("USE_VERTEX", "0")
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)
        assert is_vertex_enabled(legacy_var="GEMINI_USE_VERTEX") is False

    def test_legacy_gemini_use_vertex_fallback(self, monkeypatch):
        monkeypatch.delenv("USE_VERTEX", raising=False)
        monkeypatch.setenv("GEMINI_USE_VERTEX", "1")
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always")
            assert is_vertex_enabled(legacy_var="GEMINI_USE_VERTEX") is True
        assert any(issubclass(x.category, DeprecationWarning) for x in w)

    def test_use_vertex_takes_precedence_over_legacy(self, monkeypatch):
        monkeypatch.setenv("USE_VERTEX", "1")
        monkeypatch.setenv("GEMINI_USE_VERTEX", "1")
        # Should return True from USE_VERTEX without a deprecation warning
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always")
            result = is_vertex_enabled(legacy_var="GEMINI_USE_VERTEX")
        assert result is True
        # No deprecation warning because USE_VERTEX matched first
        assert not any(issubclass(x.category, DeprecationWarning) for x in w)


# ------------------------------------------------------------------
# setup_gemini_cli_auth()
# ------------------------------------------------------------------


@pytest.mark.asyncio
class TestSetupGeminiCliAuth:
    """Test the full auth setup function."""

    async def test_api_key_precedence_gemini_wins(self, monkeypatch):
        """GEMINI_API_KEY should take precedence over GOOGLE_API_KEY."""
        monkeypatch.setenv("GEMINI_API_KEY", "gemini-key-123")
        monkeypatch.setenv("GOOGLE_API_KEY", "google-key-456")
        monkeypatch.delenv("USE_VERTEX", raising=False)
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)

        ctx = _make_context(
            GEMINI_API_KEY="gemini-key-123",
            GOOGLE_API_KEY="google-key-456",
        )
        model, api_key, use_vertex = await setup_gemini_cli_auth(ctx)

        assert api_key == "gemini-key-123"
        assert use_vertex is False

    async def test_google_api_key_fallback(self, monkeypatch):
        """When GEMINI_API_KEY is absent, GOOGLE_API_KEY is used."""
        monkeypatch.delenv("GEMINI_API_KEY", raising=False)
        monkeypatch.setenv("GOOGLE_API_KEY", "google-key-456")
        monkeypatch.delenv("USE_VERTEX", raising=False)
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)

        ctx = _make_context(GOOGLE_API_KEY="google-key-456")
        model, api_key, use_vertex = await setup_gemini_cli_auth(ctx)

        assert api_key == "google-key-456"
        assert use_vertex is False

    async def test_no_keys_returns_empty_api_key(self, monkeypatch):
        """When no keys are set, api_key is empty (gcloud fallback)."""
        monkeypatch.delenv("GEMINI_API_KEY", raising=False)
        monkeypatch.delenv("GOOGLE_API_KEY", raising=False)
        monkeypatch.delenv("USE_VERTEX", raising=False)
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)

        ctx = _make_context()
        model, api_key, use_vertex = await setup_gemini_cli_auth(ctx)

        assert api_key == ""
        assert use_vertex is False

    async def test_vertex_mode_returns_empty_api_key(self, monkeypatch):
        """In Vertex mode, api_key is always empty and use_vertex is True."""
        monkeypatch.setenv("USE_VERTEX", "1")
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)
        monkeypatch.delenv("GEMINI_API_KEY", raising=False)
        monkeypatch.delenv("GOOGLE_API_KEY", raising=False)

        ctx = _make_context(USE_VERTEX="1")
        model, api_key, use_vertex = await setup_gemini_cli_auth(ctx)

        assert api_key == ""
        assert use_vertex is True

    async def test_vertex_mode_ignores_api_keys(self, monkeypatch):
        """In Vertex mode, even if API keys are set, they're not returned."""
        monkeypatch.setenv("USE_VERTEX", "1")
        monkeypatch.setenv("GEMINI_API_KEY", "should-be-ignored")
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)

        ctx = _make_context(
            USE_VERTEX="1",
            GEMINI_API_KEY="should-be-ignored",
        )
        model, api_key, use_vertex = await setup_gemini_cli_auth(ctx)

        assert api_key == ""
        assert use_vertex is True

    async def test_model_override_via_llm_model(self, monkeypatch):
        """LLM_MODEL env var overrides the default model."""
        monkeypatch.delenv("GEMINI_API_KEY", raising=False)
        monkeypatch.delenv("GOOGLE_API_KEY", raising=False)
        monkeypatch.delenv("USE_VERTEX", raising=False)
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)

        ctx = _make_context(LLM_MODEL="gemini-2.5-pro")
        model, api_key, use_vertex = await setup_gemini_cli_auth(ctx)

        assert model == "gemini-2.5-pro"

    async def test_default_model_when_llm_model_unset(self, monkeypatch):
        """Without LLM_MODEL, the default model from config is used."""
        from ag_ui_gemini_cli.config import DEFAULT_MODEL

        monkeypatch.delenv("GEMINI_API_KEY", raising=False)
        monkeypatch.delenv("GOOGLE_API_KEY", raising=False)
        monkeypatch.delenv("USE_VERTEX", raising=False)
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)
        monkeypatch.delenv("LLM_MODEL", raising=False)

        # Don't pass LLM_MODEL in environment so the default is used
        ctx = RunnerContext(
            session_id="s1",
            workspace_path="/workspace",
            environment={
                "GEMINI_API_KEY": "",
                "GOOGLE_API_KEY": "",
                "USE_VERTEX": "",
                "GEMINI_USE_VERTEX": "",
            },
        )
        model, api_key, use_vertex = await setup_gemini_cli_auth(ctx)

        assert model == DEFAULT_MODEL

    async def test_vertex_project_and_location_logged(self, monkeypatch, caplog):
        """Verify GOOGLE_CLOUD_PROJECT and GOOGLE_CLOUD_LOCATION are logged."""
        monkeypatch.setenv("USE_VERTEX", "1")
        monkeypatch.delenv("GEMINI_USE_VERTEX", raising=False)
        monkeypatch.delenv("GEMINI_API_KEY", raising=False)
        monkeypatch.delenv("GOOGLE_API_KEY", raising=False)
        monkeypatch.setenv("GOOGLE_CLOUD_PROJECT", "my-project")
        monkeypatch.setenv("GOOGLE_CLOUD_LOCATION", "us-central1")

        ctx = _make_context(
            USE_VERTEX="1",
            GOOGLE_CLOUD_PROJECT="my-project",
            GOOGLE_CLOUD_LOCATION="us-central1",
        )

        import logging

        with caplog.at_level(logging.INFO, logger="ambient_runner.bridges.gemini_cli.auth"):
            model, api_key, use_vertex = await setup_gemini_cli_auth(ctx)

        assert use_vertex is True
        assert "my-project" in caplog.text
        assert "us-central1" in caplog.text
