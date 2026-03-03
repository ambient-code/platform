"""Unit tests for LangGraphBridge."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from ag_ui.core import RunAgentInput

from ambient_runner.bridge import PlatformBridge
from ambient_runner.bridges.langgraph import LangGraphBridge
from ambient_runner.platform.context import RunnerContext


class TestLangGraphBridgeCapabilities:
    """Test LangGraphBridge capabilities are correctly different from Claude."""

    def test_framework_name(self):
        assert LangGraphBridge().capabilities().framework == "langgraph"

    def test_no_filesystem(self):
        assert LangGraphBridge().capabilities().file_system is False

    def test_no_mcp(self):
        assert LangGraphBridge().capabilities().mcp is False

    def test_langsmith_tracing(self):
        assert LangGraphBridge().capabilities().tracing == "langsmith"

    def test_no_session_persistence(self):
        assert LangGraphBridge().capabilities().session_persistence is False

    def test_agent_features(self):
        caps = LangGraphBridge().capabilities()
        assert "agentic_chat" in caps.agent_features
        assert "shared_state" in caps.agent_features
        assert "human_in_the_loop" in caps.agent_features
        # Claude-specific features should NOT be present
        assert "backend_tool_rendering" not in caps.agent_features
        assert "thinking" not in caps.agent_features

    def test_is_platform_bridge_subclass(self):
        assert issubclass(LangGraphBridge, PlatformBridge)


class TestLangGraphBridgeLifecycle:
    """Test lifecycle methods."""

    def test_set_context(self):
        bridge = LangGraphBridge()
        assert bridge.context is None
        ctx = RunnerContext(session_id="s1", workspace_path="/w")
        bridge.set_context(ctx)
        assert bridge.context is ctx

    def test_default_model_empty(self):
        assert LangGraphBridge().configured_model == ""


@pytest.mark.asyncio
class TestLangGraphBridgeRun:
    """Test run and interrupt lifecycle."""

    async def test_run_raises_without_langgraph_url(self):
        """Should raise RuntimeError because ag_ui_langgraph isn't installed or URL is empty."""
        bridge = LangGraphBridge()
        ctx = RunnerContext(session_id="s1", workspace_path="/w")
        bridge.set_context(ctx)
        input_data = RunAgentInput(
            thread_id="t1", run_id="r1", messages=[], state={},
            tools=[], context=[], forwarded_props={},
        )
        with pytest.raises(RuntimeError):
            async for _ in bridge.run(input_data):
                pass

    async def test_interrupt_raises_if_no_adapter(self):
        bridge = LangGraphBridge()
        with pytest.raises(RuntimeError, match="no adapter to interrupt"):
            await bridge.interrupt()

    async def test_interrupt_with_adapter_that_supports_it(self):
        bridge = LangGraphBridge()
        mock_adapter = MagicMock()
        mock_adapter.interrupt = AsyncMock()
        bridge._adapter = mock_adapter
        await bridge.interrupt()
        mock_adapter.interrupt.assert_awaited_once()

    async def test_interrupt_with_adapter_without_support(self):
        bridge = LangGraphBridge()
        mock_adapter = MagicMock(spec=[])  # spec=[] means no attributes
        bridge._adapter = mock_adapter
        # Should not raise, just log a warning
        await bridge.interrupt()
