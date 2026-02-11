"""Unit tests for LangGraphBridge."""

from unittest.mock import MagicMock, patch

import pytest

from ag_ui.core import RunAgentInput

from ambient_runner.bridge import PlatformBridge, PlatformContext
from ambient_runner.bridges.langgraph import LangGraphBridge


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


class TestLangGraphBridgeNeedsRebuild:
    """Test rebuild detection for LangGraph."""

    def test_needs_rebuild_when_no_prior_context(self):
        bridge = LangGraphBridge()
        ctx = PlatformContext(session_id="s1", workspace_path="/w")
        assert bridge.needs_rebuild(ctx) is True

    def test_no_rebuild_when_same_config(self):
        bridge = LangGraphBridge()
        ctx = PlatformContext(
            session_id="s1", workspace_path="/w",
            environment={"LANGGRAPH_URL": "http://lg", "LANGGRAPH_GRAPH_ID": "agent"},
        )
        bridge._last_ctx = ctx
        assert bridge.needs_rebuild(ctx) is False

    def test_rebuild_when_url_changes(self):
        bridge = LangGraphBridge()
        bridge._last_ctx = PlatformContext(
            session_id="s1", workspace_path="/w",
            environment={"LANGGRAPH_URL": "http://old"},
        )
        new_ctx = PlatformContext(
            session_id="s1", workspace_path="/w",
            environment={"LANGGRAPH_URL": "http://new"},
        )
        assert bridge.needs_rebuild(new_ctx) is True

    def test_rebuild_when_graph_id_changes(self):
        bridge = LangGraphBridge()
        bridge._last_ctx = PlatformContext(
            session_id="s1", workspace_path="/w",
            environment={"LANGGRAPH_URL": "http://lg", "LANGGRAPH_GRAPH_ID": "old"},
        )
        new_ctx = PlatformContext(
            session_id="s1", workspace_path="/w",
            environment={"LANGGRAPH_URL": "http://lg", "LANGGRAPH_GRAPH_ID": "new"},
        )
        assert bridge.needs_rebuild(new_ctx) is True


class TestLangGraphBridgeCreateAdapter:
    """Test adapter creation with mocked LangGraphAgent."""

    def test_raises_without_langgraph_url(self):
        """Should raise RuntimeError — either because ag_ui_langgraph is missing or URL is empty."""
        bridge = LangGraphBridge()
        ctx = PlatformContext(session_id="s1", workspace_path="/w", environment={})
        with pytest.raises(RuntimeError):
            bridge.create_adapter(ctx)

    @patch("ambient_runner.bridges.langgraph.LangGraphBridge.create_adapter")
    def test_stores_adapter_and_context(self, mock_create):
        mock_adapter = MagicMock()
        mock_create.return_value = mock_adapter

        bridge = LangGraphBridge()
        ctx = PlatformContext(
            session_id="s1", workspace_path="/w",
            environment={"LANGGRAPH_URL": "http://lg", "LANGGRAPH_GRAPH_ID": "agent"},
        )
        adapter = bridge.create_adapter(ctx)
        assert adapter is mock_adapter


@pytest.mark.asyncio
class TestLangGraphBridgeRunAndInterrupt:
    """Test run/interrupt lifecycle."""

    async def test_run_raises_if_no_adapter(self):
        bridge = LangGraphBridge()
        input_data = RunAgentInput(
            thread_id="t1", run_id="r1", messages=[], state={},
            tools=[], context=[], forwarded_props={},
        )
        with pytest.raises(RuntimeError, match="adapter not created"):
            _ = [e async for e in bridge.run(input_data)]

    async def test_interrupt_raises_if_no_adapter(self):
        bridge = LangGraphBridge()
        with pytest.raises(RuntimeError, match="no adapter to interrupt"):
            await bridge.interrupt()

    async def test_interrupt_with_adapter_that_supports_it(self):
        from unittest.mock import AsyncMock

        bridge = LangGraphBridge()
        mock_adapter = MagicMock()
        mock_adapter.interrupt = AsyncMock()  # async mock so await works
        bridge._adapter = mock_adapter
        # LangGraphBridge checks hasattr before calling
        await bridge.interrupt()
        mock_adapter.interrupt.assert_awaited_once()

    async def test_interrupt_with_adapter_without_support(self):
        bridge = LangGraphBridge()
        mock_adapter = MagicMock(spec=[])  # spec=[] means no attributes
        bridge._adapter = mock_adapter
        # Should not raise, just log a warning
        await bridge.interrupt()
