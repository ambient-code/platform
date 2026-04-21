"""Unit tests for PlatformBridge ABC and ClaudeBridge.

Coverage targets:
- ClaudeBridge initial gRPC state (None listener, empty active_streams)
- shutdown stops listener / safe when None
- start_grpc_listener creates and starts GRPCSessionListener with correct args,
  guards against duplicate starts, raises when no context
- inject_message raises NotImplementedError on PlatformBridge base
- PlatformBridge ABC contract
- FrameworkCapabilities dataclass defaults
- ClaudeBridge capabilities, lifecycle, run guards, shutdown, observability setup
"""

import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from ag_ui.core import RunAgentInput

from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
    setup_bridge_observability,
)
from ambient_runner.bridges.claude import ClaudeBridge
from ambient_runner.platform.context import RunnerContext


# ------------------------------------------------------------------
# ClaudeBridge gRPC transport tests
# ------------------------------------------------------------------


class TestClaudeBridgeGRPCState:
    """Verify gRPC state is initialized correctly on ClaudeBridge."""

    def test_grpc_listener_none_by_default(self):
        bridge = ClaudeBridge()
        assert bridge._grpc_listener is None

    def test_active_streams_empty_dict_by_default(self):
        bridge = ClaudeBridge()
        assert bridge._active_streams == {}
        assert isinstance(bridge._active_streams, dict)


@pytest.mark.asyncio
class TestClaudeBridgeShutdownGRPC:
    """Test shutdown stops the gRPC listener when present."""

    async def test_shutdown_stops_grpc_listener(self):
        bridge = ClaudeBridge()
        mock_listener = AsyncMock()
        bridge._grpc_listener = mock_listener
        await bridge.shutdown()
        mock_listener.stop.assert_awaited_once()

    async def test_shutdown_without_grpc_listener_does_not_raise(self):
        bridge = ClaudeBridge()
        assert bridge._grpc_listener is None
        await bridge.shutdown()


@pytest.mark.asyncio
class TestClaudeBridgeStartGRPCListener:
    """Test the dedicated start_grpc_listener hook (separate from _setup_platform)."""

    async def test_start_creates_listener_with_correct_args(self):
        bridge = ClaudeBridge()
        ctx = RunnerContext(session_id="sess-grpc", workspace_path="/workspace")
        bridge.set_context(ctx)

        mock_listener_instance = MagicMock()
        mock_listener_instance.start = MagicMock()
        mock_listener_cls = MagicMock(return_value=mock_listener_instance)

        with patch(
            "ambient_runner.bridges.claude.grpc_transport.GRPCSessionListener",
            mock_listener_cls,
        ):
            await bridge.start_grpc_listener("localhost:9000")

        mock_listener_instance.start.assert_called_once()
        assert bridge._grpc_listener is mock_listener_instance

    async def test_start_raises_without_context(self):
        bridge = ClaudeBridge()
        with pytest.raises(RuntimeError, match="context not set"):
            await bridge.start_grpc_listener("localhost:9000")

    async def test_duplicate_start_is_idempotent(self):
        bridge = ClaudeBridge()
        ctx = RunnerContext(session_id="sess-dup", workspace_path="/workspace")
        bridge.set_context(ctx)

        first_listener = MagicMock()
        first_listener.start = MagicMock()
        bridge._grpc_listener = first_listener

        mock_listener_cls = MagicMock()
        with patch(
            "ambient_runner.bridges.claude.grpc_transport.GRPCSessionListener",
            mock_listener_cls,
        ):
            await bridge.start_grpc_listener("localhost:9000")

        mock_listener_cls.assert_not_called()
        assert bridge._grpc_listener is first_listener

    async def test_listener_started_and_ready_event_available(self):
        bridge = ClaudeBridge()
        ctx = RunnerContext(session_id="sess-ready", workspace_path="/workspace")
        bridge.set_context(ctx)

        ready_event = asyncio.Event()
        ready_event.set()

        mock_listener = MagicMock()
        mock_listener.ready = ready_event
        mock_listener.start = MagicMock()
        mock_listener_cls = MagicMock(return_value=mock_listener)

        with patch(
            "ambient_runner.bridges.claude.grpc_transport.GRPCSessionListener",
            mock_listener_cls,
        ):
            await bridge.start_grpc_listener("localhost:9000")

        assert bridge._grpc_listener.ready.is_set()


@pytest.mark.asyncio
class TestClaudeBridgeStartGRPCListenerRealPath:
    """start_grpc_listener only patches GRPCSessionListener — no _setup_platform mock."""

    async def test_listener_class_receives_bridge_and_session_id(self):
        """Verify GRPCSessionListener is constructed with the correct bridge and session_id."""
        bridge = ClaudeBridge()
        ctx = RunnerContext(session_id="sess-realpath", workspace_path="/workspace")
        bridge.set_context(ctx)

        captured_kwargs = {}

        def capturing_init(self_inner, *, bridge, session_id, grpc_url):
            captured_kwargs["bridge"] = bridge
            captured_kwargs["session_id"] = session_id
            captured_kwargs["grpc_url"] = grpc_url
            self_inner._bridge = bridge
            self_inner._session_id = session_id
            self_inner._grpc_url = grpc_url
            self_inner._grpc_client = None
            self_inner.ready = asyncio.Event()
            self_inner._task = None

        mock_listener_cls = MagicMock()
        mock_instance = MagicMock()
        mock_instance.start = MagicMock()
        mock_listener_cls.return_value = mock_instance

        with patch(
            "ambient_runner.bridges.claude.grpc_transport.GRPCSessionListener",
            mock_listener_cls,
        ):
            await bridge.start_grpc_listener("grpc.example.com:9000")

        call_kwargs = mock_listener_cls.call_args[1]
        assert call_kwargs["bridge"] is bridge
        assert call_kwargs["session_id"] == "sess-realpath"
        assert call_kwargs["grpc_url"] == "grpc.example.com:9000"
        mock_instance.start.assert_called_once()

    async def test_listener_not_started_without_context(self):
        """start_grpc_listener raises RuntimeError when no context is set."""
        bridge = ClaudeBridge()
        mock_listener_cls = MagicMock()

        with patch(
            "ambient_runner.bridges.claude.grpc_transport.GRPCSessionListener",
            mock_listener_cls,
        ):
            with pytest.raises(RuntimeError, match="context not set"):
                await bridge.start_grpc_listener("grpc.example.com:9000")

        mock_listener_cls.assert_not_called()


# ------------------------------------------------------------------
# inject_message — base class raises NotImplementedError
# ------------------------------------------------------------------


@pytest.mark.asyncio
class TestPlatformBridgeInjectMessage:
    """inject_message must raise NotImplementedError on the base class and any
    subclass that doesn't override it."""

    async def test_base_class_raises_not_implemented(self):
        class MinimalBridge(PlatformBridge):
            def capabilities(self):
                return FrameworkCapabilities(framework="test")

            async def run(self, input_data):
                yield

            async def interrupt(self, thread_id=None):
                pass

        bridge = MinimalBridge()
        with pytest.raises(NotImplementedError):
            await bridge.inject_message("sess-1", "user", "{}")

    async def test_error_includes_bridge_class_name(self):
        class MyBridge(PlatformBridge):
            def capabilities(self):
                return FrameworkCapabilities(framework="test")

            async def run(self, input_data):
                yield

            async def interrupt(self, thread_id=None):
                pass

        bridge = MyBridge()
        with pytest.raises(NotImplementedError, match="MyBridge"):
            await bridge.inject_message("s1", "user", "{}")


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


@pytest.mark.asyncio
class TestClaudeBridgeSetupObservability:
    """Test observability setup wiring via setup_bridge_observability."""

    async def test_forwards_workflow_env_vars_to_initialize(self):
        """Verify the three ACTIVE_WORKFLOW_* env vars are read from context and forwarded."""
        ctx = RunnerContext(
            session_id="sess-1",
            workspace_path="/workspace",
            environment={
                "AGENTIC_SESSION_NAMESPACE": "my-project",
                "ACTIVE_WORKFLOW_GIT_URL": "https://github.com/org/my-wf.git",
                "ACTIVE_WORKFLOW_BRANCH": "develop",
                "ACTIVE_WORKFLOW_PATH": "workflows/analysis",
                "USER_ID": "u1",
                "USER_NAME": "Test",
            },
        )

        mock_obs_instance = AsyncMock()
        mock_obs_instance.initialize = AsyncMock(return_value=False)

        with patch(
            "ambient_runner.observability.ObservabilityManager",
            return_value=mock_obs_instance,
        ) as mock_obs_cls:
            await setup_bridge_observability(ctx, "claude-sonnet-4-5")

        mock_obs_cls.assert_called_once()
        mock_obs_instance.initialize.assert_awaited_once()
        call_kwargs = mock_obs_instance.initialize.call_args[1]

        assert call_kwargs["namespace"] == "my-project"
        assert call_kwargs["model"] == "claude-sonnet-4-5"
        assert call_kwargs["workflow_url"] == "https://github.com/org/my-wf.git"
        assert call_kwargs["workflow_branch"] == "develop"
        assert call_kwargs["workflow_path"] == "workflows/analysis"

    async def test_forwards_empty_defaults_when_workflow_vars_unset(self):
        """Verify empty-string defaults are forwarded when workflow env vars are absent."""
        ctx = RunnerContext(
            session_id="sess-2",
            workspace_path="/workspace",
            environment={
                "AGENTIC_SESSION_NAMESPACE": "ns",
                "USER_ID": "u1",
                "USER_NAME": "Test",
            },
        )

        mock_obs_instance = AsyncMock()
        mock_obs_instance.initialize = AsyncMock(return_value=False)

        with patch(
            "ambient_runner.observability.ObservabilityManager",
            return_value=mock_obs_instance,
        ):
            await setup_bridge_observability(ctx, "claude-sonnet-4-5")

        call_kwargs = mock_obs_instance.initialize.call_args[1]

        assert call_kwargs["workflow_url"] == ""
        assert call_kwargs["workflow_branch"] == ""
        assert call_kwargs["workflow_path"] == ""
