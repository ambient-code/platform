"""Unit tests for PlatformBridge ABC and ClaudeBridge."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from ag_ui.core import EventType, RunAgentInput

from ambient_runner.bridge import FrameworkCapabilities, PlatformBridge
from ambient_runner.bridges.claude import ClaudeBridge
from ambient_runner.platform.context import RunnerContext


# ------------------------------------------------------------------
# PlatformBridge ABC tests
# ------------------------------------------------------------------


class TestPlatformBridgeABC:
    """Verify the abstract contract."""

    def test_cannot_instantiate_directly(self):
        with pytest.raises(TypeError):
            PlatformBridge()

    def test_minimal_subclass_works(self):
        """A subclass implementing the three required methods can be instantiated."""

        class MinimalBridge(PlatformBridge):
            def capabilities(self):
                return FrameworkCapabilities(framework="test")

            async def run(self, input_data):
                yield  # pragma: no cover

            async def interrupt(self, thread_id=None):
                pass

        bridge = MinimalBridge()
        assert bridge.capabilities().framework == "test"

    def test_lifecycle_defaults(self):
        """Default lifecycle methods are no-ops and safe to call."""

        class MinimalBridge(PlatformBridge):
            def capabilities(self):
                return FrameworkCapabilities(framework="test")

            async def run(self, input_data):
                yield  # pragma: no cover

            async def interrupt(self, thread_id=None):
                pass

        bridge = MinimalBridge()
        assert bridge.context is None
        assert bridge.configured_model == ""
        assert bridge.obs is None
        assert bridge.get_error_context() == ""
        bridge.set_context(RunnerContext(session_id="s1", workspace_path="/tmp"))
        bridge.mark_dirty()


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

    def test_session_persistence(self):
        assert ClaudeBridge().capabilities().session_persistence is True

    def test_tracing_none_before_observability_init(self):
        """Before observability is set up, tracing should be None."""
        bridge = ClaudeBridge()
        assert bridge.capabilities().tracing is None

    def test_tracing_langfuse_after_observability_init(self):
        """After observability is set up, tracing should be 'langfuse'."""
        bridge = ClaudeBridge()
        mock_obs = MagicMock()
        mock_obs.langfuse_client = MagicMock()
        bridge._obs = mock_obs
        assert bridge.capabilities().tracing == "langfuse"


class TestClaudeBridgeLifecycle:
    """Test lifecycle methods on ClaudeBridge."""

    def test_set_context(self):
        bridge = ClaudeBridge()
        assert bridge.context is None
        ctx = RunnerContext(session_id="s1", workspace_path="/w")
        bridge.set_context(ctx)
        assert bridge.context is ctx
        assert bridge.context.session_id == "s1"

    def test_mark_dirty_resets_state(self):
        bridge = ClaudeBridge()
        bridge._ready = True
        bridge._first_run = False
        bridge._adapter = MagicMock()
        bridge.mark_dirty()
        assert bridge._ready is False
        assert bridge._first_run is True
        assert bridge._adapter is None

    def test_configured_model_empty_by_default(self):
        assert ClaudeBridge().configured_model == ""

    def test_obs_none_by_default(self):
        assert ClaudeBridge().obs is None

    def test_session_manager_none_before_init(self):
        assert ClaudeBridge().session_manager is None

    def test_get_error_context_empty_by_default(self):
        assert ClaudeBridge().get_error_context() == ""

    def test_get_error_context_with_stderr(self):
        bridge = ClaudeBridge()
        bridge._stderr_lines = ["error: something broke", "at line 42"]
        ctx = bridge.get_error_context()
        assert "something broke" in ctx
        assert "line 42" in ctx


@pytest.mark.asyncio
class TestClaudeBridgeRunGuards:
    """Test run() and interrupt() guard conditions."""

    async def test_run_raises_without_context(self):
        bridge = ClaudeBridge()
        input_data = RunAgentInput(
            thread_id="t1", run_id="r1", messages=[], state={},
            tools=[], context=[], forwarded_props={},
        )
        with pytest.raises(RuntimeError, match="Context not set"):
            async for _ in bridge.run(input_data):
                pass

    async def test_interrupt_raises_without_session_manager(self):
        bridge = ClaudeBridge()
        with pytest.raises(RuntimeError, match="No active session manager"):
            await bridge.interrupt()

    async def test_interrupt_raises_with_unknown_thread(self):
        from ambient_runner.bridges.claude.session import SessionManager

        bridge = ClaudeBridge()
        bridge._session_manager = SessionManager()
        bridge.set_context(RunnerContext(session_id="s1", workspace_path="/w"))
        with pytest.raises(RuntimeError, match="No active session"):
            await bridge.interrupt("nonexistent-thread")


@pytest.mark.asyncio
class TestClaudeBridgeShutdown:
    """Test shutdown behaviour."""

    async def test_shutdown_with_no_resources(self):
        """Shutdown should not raise when nothing is initialised."""
        bridge = ClaudeBridge()
        await bridge.shutdown()

    async def test_shutdown_calls_session_manager(self):
        bridge = ClaudeBridge()
        mock_manager = AsyncMock()
        bridge._session_manager = mock_manager
        await bridge.shutdown()
        mock_manager.shutdown.assert_awaited_once()

    async def test_shutdown_calls_obs_finalize(self):
        bridge = ClaudeBridge()
        mock_obs = AsyncMock()
        bridge._obs = mock_obs
        await bridge.shutdown()
        mock_obs.finalize.assert_awaited_once()
