"""Unit tests for adapter.py (setup_platform, build_adapter, _build_options)."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from context import RunnerContext


# ------------------------------------------------------------------
# _build_options tests (pure function, no mocks needed)
# ------------------------------------------------------------------


class TestBuildOptions:
    """Test the _build_options function that assembles SDK options."""

    def _make_context(self, **env_overrides) -> RunnerContext:
        """Create a minimal RunnerContext for testing (avoids chdir side-effect)."""
        ctx = object.__new__(RunnerContext)
        ctx.session_id = "test"
        ctx.workspace_path = "/tmp/test"
        ctx.environment = {"IS_RESUME": "false", **env_overrides}
        ctx.metadata = {}
        return ctx

    def test_basic_options_structure(self):
        from adapter import _build_options

        ctx = self._make_context()
        opts = _build_options(
            context=ctx,
            cwd_path="/work",
            add_dirs=[],
            configured_model="claude-4",
            allowed_tools=["Read", "Write"],
            mcp_servers={"jira": {}},
            system_prompt_config={"type": "text"},
            sdk_stderr_handler=lambda x: None,
            first_run=True,
        )

        assert opts["cwd"] == "/work"
        assert opts["model"] == "claude-4"
        assert opts["allowed_tools"] == ["Read", "Write"]
        assert opts["mcp_servers"] == {"jira": {}}
        assert opts["permission_mode"] == "acceptEdits"
        assert opts["include_partial_messages"] is True
        assert "continue_conversation" not in opts

    def test_add_dirs_included(self):
        from adapter import _build_options

        ctx = self._make_context()
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=["/extra1", "/extra2"],
            configured_model="", allowed_tools=[], mcp_servers={},
            system_prompt_config={}, sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert opts["add_dirs"] == ["/extra1", "/extra2"]

    def test_add_dirs_excluded_when_empty(self):
        from adapter import _build_options

        ctx = self._make_context()
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[],
            configured_model="", allowed_tools=[], mcp_servers={},
            system_prompt_config={}, sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert "add_dirs" not in opts

    def test_max_tokens_from_LLM_MAX_TOKENS(self):
        from adapter import _build_options

        ctx = self._make_context(LLM_MAX_TOKENS="8192")
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[], configured_model="",
            allowed_tools=[], mcp_servers={}, system_prompt_config={},
            sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert opts["max_tokens"] == 8192

    def test_max_tokens_fallback_MAX_TOKENS(self):
        from adapter import _build_options

        ctx = self._make_context(MAX_TOKENS="4096")
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[], configured_model="",
            allowed_tools=[], mcp_servers={}, system_prompt_config={},
            sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert opts["max_tokens"] == 4096

    def test_invalid_max_tokens_ignored(self):
        from adapter import _build_options

        ctx = self._make_context(LLM_MAX_TOKENS="nope")
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[], configured_model="",
            allowed_tools=[], mcp_servers={}, system_prompt_config={},
            sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert "max_tokens" not in opts

    def test_temperature_from_env(self):
        from adapter import _build_options

        ctx = self._make_context(LLM_TEMPERATURE="0.5")
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[], configured_model="",
            allowed_tools=[], mcp_servers={}, system_prompt_config={},
            sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert opts["temperature"] == pytest.approx(0.5)

    def test_continue_conversation_on_second_run(self):
        from adapter import _build_options

        ctx = self._make_context()
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[], configured_model="",
            allowed_tools=[], mcp_servers={}, system_prompt_config={},
            sdk_stderr_handler=lambda x: None, first_run=False,
        )
        assert opts["continue_conversation"] is True

    def test_continue_conversation_on_resume(self):
        from adapter import _build_options

        ctx = self._make_context(IS_RESUME="true")
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[], configured_model="",
            allowed_tools=[], mcp_servers={}, system_prompt_config={},
            sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert opts["continue_conversation"] is True

    def test_no_continue_on_first_run_no_resume(self):
        from adapter import _build_options

        ctx = self._make_context()
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[], configured_model="",
            allowed_tools=[], mcp_servers={}, system_prompt_config={},
            sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert "continue_conversation" not in opts

    def test_model_not_set_when_empty(self):
        from adapter import _build_options

        ctx = self._make_context()
        opts = _build_options(
            context=ctx, cwd_path="/w", add_dirs=[], configured_model="",
            allowed_tools=[], mcp_servers={}, system_prompt_config={},
            sdk_stderr_handler=lambda x: None, first_run=True,
        )
        assert "model" not in opts


# ------------------------------------------------------------------
# build_adapter tests (mocked dependencies)
# ------------------------------------------------------------------


class TestBuildAdapter:
    """Test build_adapter with mocked platform modules."""

    @patch("adapter.ClaudeAgentAdapter")
    @patch("adapter.mcp_mod")
    @patch("adapter.prompts")
    def test_creates_adapter_with_correct_args(self, mock_prompts, mock_mcp, MockAdapter):
        from adapter import build_adapter

        mock_mcp.build_mcp_servers.return_value = {"jira": {"url": "http://jira"}}
        mock_mcp.build_allowed_tools.return_value = ["Read", "Write", "Bash"]
        mock_mcp.log_auth_status = MagicMock()
        mock_prompts.build_sdk_system_prompt.return_value = {"type": "text", "content": "Hello"}
        MockAdapter.return_value = MagicMock()

        ctx = object.__new__(RunnerContext)
        ctx.session_id = "test"
        ctx.workspace_path = "/workspace"
        ctx.environment = {"IS_RESUME": "false"}
        ctx.metadata = {}

        adapter = build_adapter(
            ctx,
            configured_model="claude-sonnet",
            cwd_path="/workspace/src",
            add_dirs=["/extra"],
            first_run=True,
            obs=None,
        )

        # Verify ClaudeAgentAdapter was created
        MockAdapter.assert_called_once()
        call_kwargs = MockAdapter.call_args[1]
        assert call_kwargs["name"] == "claude_code_runner"

        # Verify MCP was called
        mock_mcp.build_mcp_servers.assert_called_once_with(ctx, "/workspace/src", None)
        mock_mcp.log_auth_status.assert_called_once()
        mock_mcp.build_allowed_tools.assert_called_once()

        # Verify prompts were built
        mock_prompts.build_sdk_system_prompt.assert_called_once_with("/workspace", "/workspace/src")


# ------------------------------------------------------------------
# setup_platform tests (mocked auth/workspace)
# ------------------------------------------------------------------


@pytest.mark.asyncio
class TestSetupPlatform:
    """Test setup_platform orchestration."""

    @patch("adapter.workspace")
    @patch("adapter.auth")
    async def test_returns_model_and_platform_info(self, mock_auth, mock_workspace):
        from adapter import setup_platform

        mock_auth.setup_sdk_authentication = AsyncMock(return_value=("key123", False, "claude-4"))
        mock_auth.populate_runtime_credentials = AsyncMock()
        mock_workspace.resolve_sdk_paths.return_value = ("/w/src", ["/w/extra"])

        ctx = object.__new__(RunnerContext)
        ctx.session_id = "test"
        ctx.workspace_path = "/w"
        ctx.environment = {}
        ctx.metadata = {}

        model, info = await setup_platform(ctx)

        assert model == "claude-4"
        assert info["cwd_path"] == "/w/src"
        assert info["add_dirs"] == ["/w/extra"]

        mock_auth.setup_sdk_authentication.assert_awaited_once_with(ctx)
        mock_auth.populate_runtime_credentials.assert_awaited_once_with(ctx)
        mock_workspace.resolve_sdk_paths.assert_called_once_with(ctx)
