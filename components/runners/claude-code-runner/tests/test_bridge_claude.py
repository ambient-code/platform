"""Unit tests for PlatformBridge ABC and ClaudeBridge."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from ag_ui.core import EventType, RunAgentInput

from ambient_runner.bridge import FrameworkCapabilities, PlatformBridge, PlatformContext
from ambient_runner.bridges.claude import ClaudeBridge


# ------------------------------------------------------------------
# PlatformBridge ABC tests
# ------------------------------------------------------------------


class TestPlatformBridgeABC:
    """Verify the abstract contract."""

    def test_cannot_instantiate_directly(self):
        with pytest.raises(TypeError):
            PlatformBridge()

    def test_needs_rebuild_default_returns_false(self):
        """The default implementation of needs_rebuild returns False."""

        class MinimalBridge(PlatformBridge):
            def capabilities(self):
                return FrameworkCapabilities(framework="test")

            def create_adapter(self, ctx):
                return None

            async def run(self, input_data):
                yield  # pragma: no cover

            async def interrupt(self):
                pass

        bridge = MinimalBridge()
        ctx = PlatformContext(session_id="s1", workspace_path="/tmp")
        assert bridge.needs_rebuild(ctx) is False


class TestPlatformContext:
    """Tests for the PlatformContext dataclass."""

    def test_defaults(self):
        ctx = PlatformContext(session_id="s1", workspace_path="/w")
        assert ctx.session_id == "s1"
        assert ctx.workspace_path == "/w"
        assert ctx.cwd_path == ""
        assert ctx.add_dirs == []
        assert ctx.model == ""
        assert ctx.mcp_servers == {}
        assert ctx.allowed_tools == []
        assert ctx.system_prompt == {}
        assert ctx.first_run is True
        assert ctx.is_resume is False
        assert ctx.environment == {}
        assert ctx.extra == {}

    def test_custom_values(self):
        ctx = PlatformContext(
            session_id="s2",
            workspace_path="/work",
            cwd_path="/work/src",
            model="claude-4",
            mcp_servers={"jira": {"url": "http://jira"}},
            allowed_tools=["Read", "Write"],
            first_run=False,
            is_resume=True,
            environment={"LLM_MAX_TOKENS": "4096"},
        )
        assert ctx.model == "claude-4"
        assert ctx.mcp_servers == {"jira": {"url": "http://jira"}}
        assert ctx.allowed_tools == ["Read", "Write"]
        assert ctx.first_run is False
        assert ctx.is_resume is True


class TestFrameworkCapabilities:
    """Tests for the FrameworkCapabilities dataclass."""

    def test_defaults(self):
        caps = FrameworkCapabilities(framework="test")
        assert caps.framework == "test"
        assert caps.agent_features == []
        assert caps.file_system is False
        assert caps.mcp is False
        assert caps.tracing is None
        assert caps.session_persistence is False


# ------------------------------------------------------------------
# ClaudeBridge tests
# ------------------------------------------------------------------


class TestClaudeBridgeCapabilities:
    """Test ClaudeBridge.capabilities() returns correct values."""

    def test_framework_name(self):
        assert ClaudeBridge().capabilities().framework == "claude-agent-sdk"

    def test_agent_features(self):
        caps = ClaudeBridge().capabilities()
        assert "agentic_chat" in caps.agent_features
        assert "backend_tool_rendering" in caps.agent_features
        assert "thinking" in caps.agent_features

    def test_file_system_support(self):
        assert ClaudeBridge().capabilities().file_system is True

    def test_mcp_support(self):
        assert ClaudeBridge().capabilities().mcp is True

    def test_tracing(self):
        assert ClaudeBridge().capabilities().tracing == "langfuse"

    def test_session_persistence(self):
        assert ClaudeBridge().capabilities().session_persistence is True


class TestClaudeBridgeBuildOptions:
    """Test the private _build_options method with various contexts."""

    def test_basic_options(self):
        ctx = PlatformContext(
            session_id="s1",
            workspace_path="/w",
            cwd_path="/w/src",
            model="claude-sonnet",
            allowed_tools=["Read"],
            mcp_servers={"jira": {}},
            system_prompt={"type": "text", "content": "Hello"},
        )
        opts = ClaudeBridge._build_options(ctx)

        assert opts["cwd"] == "/w/src"
        assert opts["model"] == "claude-sonnet"
        assert opts["allowed_tools"] == ["Read"]
        assert opts["mcp_servers"] == {"jira": {}}
        assert opts["system_prompt"] == {"type": "text", "content": "Hello"}
        assert opts["permission_mode"] == "acceptEdits"
        assert opts["include_partial_messages"] is True

    def test_add_dirs_included_when_present(self):
        ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w", add_dirs=["/extra"])
        opts = ClaudeBridge._build_options(ctx)
        assert opts["add_dirs"] == ["/extra"]

    def test_add_dirs_excluded_when_empty(self):
        ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w")
        opts = ClaudeBridge._build_options(ctx)
        assert "add_dirs" not in opts

    def test_max_tokens_from_environment(self):
        ctx = PlatformContext(
            session_id="s1", workspace_path="/w", cwd_path="/w",
            environment={"LLM_MAX_TOKENS": "8192"},
        )
        opts = ClaudeBridge._build_options(ctx)
        assert opts["max_tokens"] == 8192

    def test_max_tokens_fallback_to_MAX_TOKENS(self):
        ctx = PlatformContext(
            session_id="s1", workspace_path="/w", cwd_path="/w",
            environment={"MAX_TOKENS": "4096"},
        )
        opts = ClaudeBridge._build_options(ctx)
        assert opts["max_tokens"] == 4096

    def test_invalid_max_tokens_ignored(self):
        ctx = PlatformContext(
            session_id="s1", workspace_path="/w", cwd_path="/w",
            environment={"LLM_MAX_TOKENS": "not_a_number"},
        )
        opts = ClaudeBridge._build_options(ctx)
        assert "max_tokens" not in opts

    def test_temperature_from_environment(self):
        ctx = PlatformContext(
            session_id="s1", workspace_path="/w", cwd_path="/w",
            environment={"LLM_TEMPERATURE": "0.7"},
        )
        opts = ClaudeBridge._build_options(ctx)
        assert opts["temperature"] == pytest.approx(0.7)

    def test_continue_conversation_on_non_first_run(self):
        ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w", first_run=False)
        opts = ClaudeBridge._build_options(ctx)
        assert opts["continue_conversation"] is True

    def test_continue_conversation_on_resume(self):
        ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w", is_resume=True)
        opts = ClaudeBridge._build_options(ctx)
        assert opts["continue_conversation"] is True

    def test_no_continue_on_first_run(self):
        ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w", first_run=True, is_resume=False)
        opts = ClaudeBridge._build_options(ctx)
        assert "continue_conversation" not in opts

    def test_model_not_set_when_empty(self):
        ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w", model="")
        opts = ClaudeBridge._build_options(ctx)
        assert "model" not in opts


class TestClaudeBridgeNeedsRebuild:
    """Test rebuild detection logic."""

    def test_needs_rebuild_when_no_prior_context(self):
        bridge = ClaudeBridge()
        ctx = PlatformContext(session_id="s1", workspace_path="/w")
        assert bridge.needs_rebuild(ctx) is True

    def test_no_rebuild_when_same_config(self):
        bridge = ClaudeBridge()
        ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w/src", model="claude-4")
        bridge._last_ctx = ctx
        assert bridge.needs_rebuild(ctx) is False

    def test_rebuild_when_cwd_changes(self):
        bridge = ClaudeBridge()
        bridge._last_ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w/old")
        new_ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w/new")
        assert bridge.needs_rebuild(new_ctx) is True

    def test_rebuild_when_model_changes(self):
        bridge = ClaudeBridge()
        bridge._last_ctx = PlatformContext(session_id="s1", workspace_path="/w", model="old-model")
        new_ctx = PlatformContext(session_id="s1", workspace_path="/w", model="new-model")
        assert bridge.needs_rebuild(new_ctx) is True

    def test_rebuild_when_mcp_servers_change(self):
        bridge = ClaudeBridge()
        bridge._last_ctx = PlatformContext(session_id="s1", workspace_path="/w", mcp_servers={"a": {}})
        new_ctx = PlatformContext(session_id="s1", workspace_path="/w", mcp_servers={"b": {}})
        assert bridge.needs_rebuild(new_ctx) is True


@pytest.mark.asyncio
class TestClaudeBridgeRunAndInterrupt:
    """Test run/interrupt lifecycle with a mocked adapter."""

    async def test_run_raises_if_no_adapter(self):
        bridge = ClaudeBridge()
        input_data = RunAgentInput(
            thread_id="t1", run_id="r1", messages=[], state={},
            tools=[], context=[], forwarded_props={},
        )
        with pytest.raises(RuntimeError, match="adapter not created"):
            _ = [e async for e in bridge.run(input_data)]

    async def test_interrupt_raises_if_no_adapter(self):
        bridge = ClaudeBridge()
        with pytest.raises(RuntimeError, match="no adapter to interrupt"):
            await bridge.interrupt()

    @patch("ambient_runner.bridges.claude.ClaudeAgentAdapter")
    async def test_create_adapter_stores_instance(self, MockAdapter):
        MockAdapter.return_value = MagicMock()
        bridge = ClaudeBridge()
        ctx = PlatformContext(session_id="s1", workspace_path="/w", cwd_path="/w")
        adapter = bridge.create_adapter(ctx)
        assert bridge._adapter is adapter
        assert bridge._last_ctx is ctx
