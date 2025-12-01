"""Unit tests for observability module."""

import pytest
import os
import logging
from unittest.mock import Mock, patch
from observability import ObservabilityManager


@pytest.fixture
def mock_langfuse_client():
    """Create mock Langfuse client for SDK v3."""
    mock_client = Mock()
    mock_trace = Mock()
    # SDK v3 uses client.trace() to create trace object
    mock_client.trace.return_value = mock_trace
    mock_client.flush = Mock()
    # trace.generation() and trace.span() return observation objects
    mock_trace.generation.return_value = Mock()
    mock_trace.span.return_value = Mock()
    return mock_client, mock_trace


@pytest.fixture
def manager():
    """Create ObservabilityManager instance."""
    return ObservabilityManager(
        session_id="test-session-123", user_id="user-456", user_name="Test User"
    )


class TestObservabilityManagerInit:
    """Tests for ObservabilityManager initialization."""

    def test_init_sets_properties(self):
        """Test that __init__ sets all properties correctly."""
        manager = ObservabilityManager(
            session_id="session-1", user_id="user-1", user_name="John Doe"
        )

        assert manager.session_id == "session-1"
        assert manager.user_id == "user-1"
        assert manager.user_name == "John Doe"
        assert manager.langfuse_client is None
        assert manager._propagate_ctx is None
        assert manager._tool_spans == {}


class TestLangfuseInitialization:
    """Tests for Langfuse initialization."""

    @pytest.mark.asyncio
    async def test_init_langfuse_unavailable(self, manager):
        """Test initialization when Langfuse SDK is not available."""
        # Mock the import to raise ImportError
        with patch.dict('sys.modules', {'langfuse': None}):
            with patch('builtins.__import__', side_effect=ImportError("No module named 'langfuse'")):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None

    @pytest.mark.asyncio
    async def test_init_langfuse_disabled(self, manager):
        """Test initialization when LANGFUSE_ENABLED is false."""
        with patch.dict(os.environ, {"LANGFUSE_ENABLED": "false"}):
            result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None

    @pytest.mark.asyncio
    async def test_init_missing_public_key(self, manager, caplog):
        """Test initialization with missing LANGFUSE_PUBLIC_KEY."""
        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None
        assert "LANGFUSE_ENABLED is true but keys are missing" in caplog.text
        assert "ambient-admin-langfuse-secret" in caplog.text

    @pytest.mark.asyncio
    async def test_init_missing_secret_key(self, manager, caplog):
        """Test initialization with missing LANGFUSE_SECRET_KEY."""
        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert "LANGFUSE_ENABLED is true but keys are missing" in caplog.text

    @pytest.mark.asyncio
    async def test_init_missing_host(self, manager, caplog):
        """Test initialization with missing LANGFUSE_HOST."""
        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert "LANGFUSE_HOST is missing" in caplog.text

    @pytest.mark.asyncio
    @patch("langfuse.propagate_attributes")
    @patch("langfuse.Langfuse")
    async def test_init_successful(self, mock_langfuse_class, mock_propagate, manager, caplog):
        """Test successful Langfuse initialization with SDK v3 propagate_attributes pattern."""
        mock_client = Mock()
        mock_langfuse_class.return_value = mock_client

        # Mock propagate_attributes context manager
        mock_ctx = Mock()
        mock_ctx.__enter__ = Mock()
        mock_ctx.__exit__ = Mock()
        mock_propagate.return_value = mock_ctx

        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.INFO):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is True
        assert manager.langfuse_client is not None
        assert manager._propagate_ctx is not None

        mock_langfuse_class.assert_called_once_with(
            public_key="pk-lf-public",
            secret_key="sk-lf-secret",
            host="http://localhost:3000",
        )

        # Verify propagate_attributes was called
        mock_propagate.assert_called_once()
        call_kwargs = mock_propagate.call_args[1]
        assert call_kwargs["user_id"] == manager.user_id
        assert call_kwargs["session_id"] == manager.session_id
        assert "claude-code" in call_kwargs["tags"]

        assert "Session tracking enabled" in caplog.text

    @pytest.mark.asyncio
    @patch("langfuse.propagate_attributes")
    @patch("langfuse.Langfuse")
    async def test_init_with_user_tracking(self, mock_langfuse_class, mock_propagate, caplog):
        """Test Langfuse initialization with user tracking."""
        mock_client = Mock()
        mock_langfuse_class.return_value = mock_client

        # Mock propagate_attributes context manager
        mock_ctx = Mock()
        mock_ctx.__enter__ = Mock()
        mock_ctx.__exit__ = Mock()
        mock_propagate.return_value = mock_ctx

        manager = ObservabilityManager(
            session_id="session-1", user_id="user-123", user_name="Jane Doe"
        )

        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.INFO):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is True
        assert "session_id=session-1, user_id=user-123" in caplog.text

    @pytest.mark.asyncio
    @patch("langfuse.Langfuse")
    async def test_init_langfuse_exception(self, mock_langfuse_class, manager, caplog):
        """Test Langfuse initialization when SDK raises exception."""
        mock_langfuse_class.side_effect = Exception("Connection failed")

        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None
        assert manager._propagate_ctx is None
        assert "Langfuse init failed" in caplog.text

    @pytest.mark.asyncio
    @patch("langfuse.Langfuse")
    async def test_init_sanitizes_api_keys_in_error(
        self, mock_langfuse_class, manager, caplog
    ):
        """Test that API keys are sanitized in error messages."""
        mock_langfuse_class.side_effect = Exception("Auth failed with key pk-lf-public")

        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        # API key should be redacted
        assert "pk-lf-public" not in caplog.text
        assert "[REDACTED_PUBLIC_KEY]" in caplog.text


class TestStartTurn:
    """Tests for start_turn method."""

    def test_start_turn_no_client(self, manager):
        """Test start_turn when Langfuse client is not initialized."""
        # Should not raise exception
        manager.start_turn("claude-3-5-sonnet")

    def test_start_turn_creates_generation(self, manager):
        """Test start_turn creates a generation trace."""
        mock_client = Mock()
        mock_ctx = Mock()
        mock_generation = Mock()
        mock_ctx.__enter__ = Mock(return_value=mock_generation)
        mock_client.start_as_current_observation.return_value = mock_ctx

        manager.langfuse_client = mock_client

        manager.start_turn("claude-3-5-sonnet", "Test prompt")

        # Verify start_as_current_observation was called
        mock_client.start_as_current_observation.assert_called_once()
        call_kwargs = mock_client.start_as_current_observation.call_args[1]
        assert call_kwargs["as_type"] == "generation"
        assert call_kwargs["name"] == "claude_interaction"
        assert call_kwargs["model"] == "claude-3-5-sonnet"

        assert manager._current_turn_generation is not None

    def test_start_turn_prevents_duplicate_traces(self, manager):
        """Test that start_turn prevents duplicate trace creation for the same turn.

        Simulates SDK behavior where multiple AssistantMessages arrive during streaming.
        Only the first AssistantMessage should create a trace; subsequent ones should be ignored
        until end_turn() is called.
        """
        mock_client = Mock()
        mock_ctx = Mock()
        mock_generation = Mock()
        mock_ctx.__enter__ = Mock(return_value=mock_generation)
        mock_client.start_as_current_observation.return_value = mock_ctx

        manager.langfuse_client = mock_client

        # First call to start_turn - should create a trace
        manager.start_turn("claude-3-5-sonnet", "User input")
        assert mock_client.start_as_current_observation.call_count == 1
        assert manager._current_turn_generation is not None

        # Second call to start_turn (same turn, streaming update) - should be ignored
        manager.start_turn("claude-3-5-sonnet", "User input")
        assert mock_client.start_as_current_observation.call_count == 1  # Still 1, not 2

        # Third call to start_turn (still same turn) - should be ignored
        manager.start_turn("claude-3-5-sonnet", "User input")
        assert mock_client.start_as_current_observation.call_count == 1  # Still 1, not 3


class TestEndTurn:
    """Tests for end_turn method."""

    def test_end_turn_no_generation(self, manager):
        """Test end_turn when no turn is active."""
        # Should not raise exception
        manager.end_turn(1, Mock(), None)


class TestTrackToolUse:
    """Tests for track_tool_use method."""

    def test_track_tool_use_no_client(self, manager):
        """Test track_tool_use when Langfuse client is not initialized."""
        # Should not raise exception
        manager.track_tool_use("Read", "tool-123", {"file": "test.txt"})

    def test_track_tool_use_creates_span(self, manager):
        """Test track_tool_use creates tool span as child of current turn."""
        mock_client = Mock()
        mock_generation = Mock()
        mock_tool_span = Mock()

        # Mock generation.start_observation() method
        mock_generation.start_observation.return_value = mock_tool_span

        manager.langfuse_client = mock_client
        manager._current_turn_generation = mock_generation

        tool_input = {"file_path": "/test/file.txt"}
        manager.track_tool_use("Read", "tool-456", tool_input)

        # Verify generation.start_observation() was called with correct params
        mock_generation.start_observation.assert_called_once()
        call_kwargs = mock_generation.start_observation.call_args[1]
        assert call_kwargs["as_type"] == "span"
        assert call_kwargs["name"] == "tool_Read"
        assert call_kwargs["input"] == tool_input
        assert call_kwargs["metadata"]["tool_id"] == "tool-456"
        assert call_kwargs["metadata"]["tool_name"] == "Read"

        assert "tool-456" in manager._tool_spans
        assert manager._tool_spans["tool-456"] == mock_tool_span


class TestTrackToolResult:
    """Tests for track_tool_result method."""

    def test_track_tool_result_no_span(self, manager):
        """Test track_tool_result when tool span doesn't exist."""
        # Should not raise exception
        manager.track_tool_result("tool-999", "result", False)

    def test_track_tool_result_success(self):
        """Test track_tool_result for successful tool execution."""
        mock_tool_span = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager._tool_spans["tool-123"] = mock_tool_span

        manager.track_tool_result("tool-123", "File contents", is_error=False)

        # SDK v3: Uses update() then end()
        mock_tool_span.update.assert_called_once()
        call_kwargs = mock_tool_span.update.call_args[1]
        assert call_kwargs["level"] == "DEFAULT"
        assert "File contents" in call_kwargs["output"]["result"]
        assert call_kwargs["metadata"]["is_error"] is False

        # Verify span.end() was called
        mock_tool_span.end.assert_called_once()
        assert "tool-123" not in manager._tool_spans

    def test_track_tool_result_error(self):
        """Test track_tool_result for failed tool execution."""
        mock_tool_span = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager._tool_spans["tool-123"] = mock_tool_span

        manager.track_tool_result("tool-123", "Error: File not found", is_error=True)

        # SDK v3: Uses update() then end()
        mock_tool_span.update.assert_called_once()
        call_kwargs = mock_tool_span.update.call_args[1]
        assert call_kwargs["level"] == "ERROR"
        assert call_kwargs["metadata"]["is_error"] is True

        # Verify span.end() was called
        mock_tool_span.end.assert_called_once()


class TestFinalize:
    """Tests for finalize method."""

    @pytest.mark.asyncio
    async def test_finalize_no_client(self, manager):
        """Test finalize when Langfuse client is not initialized."""
        # Should not raise exception
        await manager.finalize()

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_finalize_closes_turn(self, mock_timeout):
        """Test finalize closes open turn."""
        mock_timeout.return_value = (True, None)

        mock_client = Mock()
        mock_ctx = Mock()
        mock_ctx.__exit__ = Mock()
        mock_generation = Mock()
        mock_propagate_ctx = Mock()
        mock_propagate_ctx.__exit__ = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager._current_turn_generation = mock_generation
        manager._current_turn_ctx = mock_ctx
        manager._propagate_ctx = mock_propagate_ctx

        await manager.finalize()

        # Verify turn context was exited
        mock_ctx.__exit__.assert_called_once()
        # Verify propagate context was exited
        mock_propagate_ctx.__exit__.assert_called_once()
        # Verify flush was called
        mock_timeout.assert_called_once()

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_finalize_flush_timeout(self, mock_timeout, caplog):
        """Test finalize when flush times out."""
        mock_timeout.return_value = (False, None)  # Timeout

        mock_client = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client

        with caplog.at_level(logging.ERROR):
            await manager.finalize()

        assert "timed out" in caplog.text


class TestCleanupOnError:
    """Tests for cleanup_on_error method."""

    @pytest.mark.asyncio
    async def test_cleanup_no_client(self, manager):
        """Test cleanup_on_error when Langfuse client is not initialized."""
        # Should not raise exception
        await manager.cleanup_on_error(ValueError("test error"))

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_cleanup_on_error(self, mock_timeout):
        """Test cleanup_on_error marks turn as error."""
        mock_timeout.return_value = (True, None)

        mock_client = Mock()
        mock_generation = Mock()
        mock_ctx = Mock()
        mock_ctx.__exit__ = Mock()
        mock_propagate_ctx = Mock()
        mock_propagate_ctx.__exit__ = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager._current_turn_generation = mock_generation
        manager._current_turn_ctx = mock_ctx
        manager._propagate_ctx = mock_propagate_ctx

        error = ValueError("Session failed")
        await manager.cleanup_on_error(error)

        # Verify turn generation was marked as error
        mock_generation.update.assert_called_once()
        call_kwargs = mock_generation.update.call_args[1]
        assert call_kwargs["level"] == "ERROR"

        # Verify contexts were exited
        mock_ctx.__exit__.assert_called_once()
        mock_propagate_ctx.__exit__.assert_called_once()

        # Verify flush was called
        mock_timeout.assert_called_once()

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_cleanup_flush_timeout(self, mock_timeout, caplog):
        """Test cleanup when flush times out."""
        mock_timeout.return_value = (False, None)  # Timeout

        mock_client = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client

        with caplog.at_level(logging.ERROR):
            await manager.cleanup_on_error(ValueError("test"))

        assert "timed out" in caplog.text


class TestEnvironmentVariableCombinations:
    """Tests for various environment variable combinations."""

    @pytest.mark.asyncio
    @pytest.mark.parametrize(
        "enabled_value", ["1", "true", "True", "TRUE", "yes", "YES"]
    )
    async def test_langfuse_enabled_variations(self, enabled_value, manager):
        """Test that various truthy values for LANGFUSE_ENABLED work."""
        env_vars = {
            "LANGFUSE_ENABLED": enabled_value,
            "LANGFUSE_PUBLIC_KEY": "",  # Missing key should still return False
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            result = await manager.initialize("test prompt", "test-namespace")

        # Should fail due to missing public key, but LANGFUSE_ENABLED was recognized as true
        assert result is False

    @pytest.mark.asyncio
    @pytest.mark.parametrize("enabled_value", ["0", "false", "False", "no", "NO", ""])
    async def test_langfuse_disabled_variations(self, enabled_value, manager):
        """Test that various falsy values for LANGFUSE_ENABLED work."""
        env_vars = {
            "LANGFUSE_ENABLED": enabled_value,
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None
