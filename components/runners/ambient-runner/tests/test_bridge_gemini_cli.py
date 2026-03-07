"""Tests for GeminiCLIBridge -- lifecycle, capabilities, error context, MCP status."""

import json
import tempfile
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from ag_ui.core import RunAgentInput

from ambient_runner.bridges.gemini_cli.bridge import GeminiCLIBridge
from ambient_runner.platform.context import RunnerContext


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------


def _make_context(**env_overrides) -> RunnerContext:
    clean = {
        "GEMINI_API_KEY": "",
        "GOOGLE_API_KEY": "",
        "USE_VERTEX": "",
        "GEMINI_USE_VERTEX": "",
        "LLM_MODEL": "",
    }
    clean.update(env_overrides)
    return RunnerContext(
        session_id="s1", workspace_path="/workspace", environment=clean
    )


# ------------------------------------------------------------------
# Capabilities
# ------------------------------------------------------------------


class TestGeminiCLIBridgeCapabilities:
    """Test capabilities() reporting."""

    def test_framework_name(self):
        bridge = GeminiCLIBridge()
        assert bridge.capabilities().framework == "gemini-cli"

    def test_agent_features(self):
        bridge = GeminiCLIBridge()
        caps = bridge.capabilities()
        assert "agentic_chat" in caps.agent_features
        assert "backend_tool_rendering" in caps.agent_features

    def test_file_system_support(self):
        assert GeminiCLIBridge().capabilities().file_system is True

    def test_mcp_support(self):
        assert GeminiCLIBridge().capabilities().mcp is True

    def test_tracing_none_before_observability_init(self):
        bridge = GeminiCLIBridge()
        assert bridge.capabilities().tracing is None

    def test_tracing_langfuse_after_observability_init(self):
        bridge = GeminiCLIBridge()
        mock_obs = MagicMock()
        mock_obs.langfuse_client = MagicMock()
        bridge._obs = mock_obs
        assert bridge.capabilities().tracing == "langfuse"


# ------------------------------------------------------------------
# Lifecycle / lazy init
# ------------------------------------------------------------------


class TestGeminiCLIBridgeLifecycle:
    """Test lifecycle methods."""

    def test_context_none_by_default(self):
        bridge = GeminiCLIBridge()
        assert bridge.context is None

    def test_set_context(self):
        bridge = GeminiCLIBridge()
        ctx = _make_context()
        bridge.set_context(ctx)
        assert bridge.context is ctx

    def test_configured_model_empty_by_default(self):
        assert GeminiCLIBridge().configured_model == ""

    def test_obs_none_by_default(self):
        assert GeminiCLIBridge().obs is None

    def test_session_manager_none_before_init(self):
        bridge = GeminiCLIBridge()
        assert bridge._session_manager is None

    def test_not_ready_by_default(self):
        bridge = GeminiCLIBridge()
        assert bridge._ready is False

    @pytest.mark.asyncio
    async def test_ensure_ready_raises_without_context(self):
        """_ensure_ready should raise if context is not set."""
        bridge = GeminiCLIBridge()
        with pytest.raises(RuntimeError, match="Context not set"):
            await bridge._ensure_ready()

    @pytest.mark.asyncio
    async def test_setup_platform_not_called_until_run(self):
        """_setup_platform should NOT be called on construction."""
        bridge = GeminiCLIBridge()
        bridge.set_context(_make_context())
        # Still not ready -- _setup_platform hasn't been called
        assert bridge._ready is False
        assert bridge._session_manager is None


# ------------------------------------------------------------------
# mark_dirty()
# ------------------------------------------------------------------


class TestGeminiCLIBridgeMarkDirty:
    """Test mark_dirty() resets state for re-initialization."""

    def test_mark_dirty_resets_ready_flag(self):
        bridge = GeminiCLIBridge()
        bridge._ready = True
        bridge.mark_dirty()
        assert bridge._ready is False

    def test_mark_dirty_clears_adapter(self):
        bridge = GeminiCLIBridge()
        bridge._adapter = MagicMock()
        bridge.mark_dirty()
        assert bridge._adapter is None

    def test_mark_dirty_shuts_down_session_manager(self):
        """mark_dirty() should trigger shutdown on the old session manager."""
        bridge = GeminiCLIBridge()
        mock_manager = AsyncMock()
        bridge._session_manager = mock_manager

        # We need a running event loop for mark_dirty to schedule shutdown
        with patch("asyncio.get_running_loop") as mock_loop:
            mock_loop.return_value = MagicMock()
            with patch("asyncio.ensure_future") as mock_future:
                mock_future.return_value = MagicMock()
                bridge.mark_dirty()

        assert bridge._session_manager is None


# ------------------------------------------------------------------
# get_error_context()
# ------------------------------------------------------------------


class TestGeminiCLIBridgeErrorContext:
    """Test get_error_context() returns stderr from workers."""

    def test_returns_empty_when_no_session_manager(self):
        bridge = GeminiCLIBridge()
        assert bridge.get_error_context() == ""

    def test_returns_empty_when_no_stderr(self):
        bridge = GeminiCLIBridge()
        from ambient_runner.bridges.gemini_cli.session import GeminiSessionManager

        bridge._session_manager = GeminiSessionManager()
        bridge._session_manager.get_or_create_worker("t1", model="m")
        assert bridge.get_error_context() == ""

    def test_returns_stderr_from_active_workers(self):
        bridge = GeminiCLIBridge()
        from ambient_runner.bridges.gemini_cli.session import GeminiSessionManager

        mgr = GeminiSessionManager()
        bridge._session_manager = mgr

        worker = mgr.get_or_create_worker("t1", model="m")
        worker._stderr_lines = ["error: auth failed", "error: retry limit"]

        ctx = bridge.get_error_context()
        assert "Gemini CLI stderr:" in ctx
        assert "auth failed" in ctx
        assert "retry limit" in ctx

    def test_collects_stderr_from_multiple_workers(self):
        bridge = GeminiCLIBridge()
        from ambient_runner.bridges.gemini_cli.session import GeminiSessionManager

        mgr = GeminiSessionManager()
        bridge._session_manager = mgr

        w1 = mgr.get_or_create_worker("t1", model="m")
        w1._stderr_lines = ["err-from-t1"]
        w2 = mgr.get_or_create_worker("t2", model="m")
        w2._stderr_lines = ["err-from-t2"]

        ctx = bridge.get_error_context()
        assert "err-from-t1" in ctx
        assert "err-from-t2" in ctx


# ------------------------------------------------------------------
# get_mcp_status()
# ------------------------------------------------------------------


@pytest.mark.asyncio
class TestGeminiCLIBridgeMCPStatus:
    """Test get_mcp_status() reads .gemini/settings.json."""

    async def test_no_settings_path_returns_empty(self):
        bridge = GeminiCLIBridge()
        bridge._mcp_settings_path = None
        result = await bridge.get_mcp_status()
        assert result == {"servers": [], "totalCount": 0}

    async def test_missing_settings_file_returns_empty(self):
        bridge = GeminiCLIBridge()
        bridge._mcp_settings_path = "/nonexistent/path/settings.json"
        result = await bridge.get_mcp_status()
        assert result == {"servers": [], "totalCount": 0}

    async def test_valid_settings_returns_server_info(self):
        """Parse a .gemini/settings.json with MCP servers."""
        with tempfile.TemporaryDirectory() as tmpdir:
            settings_path = Path(tmpdir) / "settings.json"
            settings = {
                "mcpServers": {
                    "my-tool": {
                        "command": "npx",
                        "args": ["-y", "my-mcp-server"],
                    },
                    "remote-service": {
                        "url": "https://mcp.example.com/sse",
                    },
                    "http-service": {
                        "httpUrl": "https://mcp.example.com/api",
                    },
                }
            }
            settings_path.write_text(json.dumps(settings))

            bridge = GeminiCLIBridge()
            bridge._mcp_settings_path = str(settings_path)

            result = await bridge.get_mcp_status()

            assert result["totalCount"] == 3
            assert len(result["servers"]) == 3

            servers_by_name = {s["name"]: s for s in result["servers"]}
            assert servers_by_name["my-tool"]["transport"] == "stdio"
            assert servers_by_name["remote-service"]["transport"] == "sse"
            assert servers_by_name["http-service"]["transport"] == "http"

            for server in result["servers"]:
                assert server["status"] == "configured"

    async def test_empty_mcp_servers_returns_zero_count(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            settings_path = Path(tmpdir) / "settings.json"
            settings_path.write_text(json.dumps({"mcpServers": {}}))

            bridge = GeminiCLIBridge()
            bridge._mcp_settings_path = str(settings_path)

            result = await bridge.get_mcp_status()
            assert result["totalCount"] == 0
            assert result["servers"] == []

    async def test_invalid_json_returns_error(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            settings_path = Path(tmpdir) / "settings.json"
            settings_path.write_text("not valid json {{{")

            bridge = GeminiCLIBridge()
            bridge._mcp_settings_path = str(settings_path)

            result = await bridge.get_mcp_status()
            assert "error" in result


# ------------------------------------------------------------------
# run() guard conditions
# ------------------------------------------------------------------


@pytest.mark.asyncio
class TestGeminiCLIBridgeRunGuards:
    """Test run() pre-conditions."""

    async def test_run_raises_without_context(self):
        bridge = GeminiCLIBridge()
        input_data = RunAgentInput(
            thread_id="t1",
            run_id="r1",
            messages=[],
            state={},
            tools=[],
            context=[],
            forwarded_props={},
        )
        with pytest.raises(RuntimeError, match="Context not set"):
            async for _ in bridge.run(input_data):
                pass


# ------------------------------------------------------------------
# interrupt()
# ------------------------------------------------------------------


@pytest.mark.asyncio
class TestGeminiCLIBridgeInterrupt:
    """Test interrupt() behaviour."""

    async def test_interrupt_raises_without_session_manager(self):
        bridge = GeminiCLIBridge()
        with pytest.raises(RuntimeError, match="No active session manager"):
            await bridge.interrupt("t1")

    async def test_interrupt_raises_without_thread_id(self):
        bridge = GeminiCLIBridge()
        from ambient_runner.bridges.gemini_cli.session import GeminiSessionManager

        bridge._session_manager = GeminiSessionManager()
        # No context, no thread_id
        with pytest.raises(RuntimeError, match="No thread_id"):
            await bridge.interrupt()

    async def test_interrupt_forwards_to_session_manager(self):
        bridge = GeminiCLIBridge()
        mock_manager = AsyncMock()
        bridge._session_manager = mock_manager

        await bridge.interrupt("t1")
        mock_manager.interrupt.assert_awaited_once_with("t1")


# ------------------------------------------------------------------
# shutdown()
# ------------------------------------------------------------------


@pytest.mark.asyncio
class TestGeminiCLIBridgeShutdown:
    """Test shutdown behaviour."""

    async def test_shutdown_with_no_resources(self):
        bridge = GeminiCLIBridge()
        await bridge.shutdown()

    async def test_shutdown_calls_session_manager(self):
        bridge = GeminiCLIBridge()
        mock_manager = AsyncMock()
        bridge._session_manager = mock_manager
        await bridge.shutdown()
        mock_manager.shutdown.assert_awaited_once()

    async def test_shutdown_calls_obs_finalize(self):
        bridge = GeminiCLIBridge()
        mock_obs = AsyncMock()
        bridge._obs = mock_obs
        await bridge.shutdown()
        mock_obs.finalize.assert_awaited_once()
