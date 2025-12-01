"""Unit tests for duplicate turn prevention in observability module."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from observability import ObservabilityManager


class TestDuplicateTurnPrevention:
    """Tests for preventing duplicate trace creation."""

    @pytest.mark.asyncio
    async def test_multiple_assistant_messages_same_turn_no_duplicates(self):
        """Test that multiple AssistantMessages for the same turn don't create duplicate traces."""
        manager = ObservabilityManager(
            session_id="test-session", user_id="user-1", user_name="Test User"
        )

        # Mock Langfuse client
        mock_client = Mock()
        mock_ctx = Mock()
        mock_generation = Mock()
        mock_ctx.__enter__ = Mock(return_value=mock_generation)
        mock_ctx.__exit__ = Mock()
        mock_client.start_as_current_observation = Mock(return_value=mock_ctx)

        manager.langfuse_client = mock_client

        # Simulate first AssistantMessage - should create trace
        manager.start_turn("claude-sonnet-4-5")
        assert manager._current_turn_generation is not None
        assert mock_client.start_as_current_observation.call_count == 1

        # Simulate second AssistantMessage for SAME turn - should skip
        manager.start_turn("claude-sonnet-4-5")
        # Should still be 1 call (no new trace created)
        assert mock_client.start_as_current_observation.call_count == 1

        # Simulate third AssistantMessage for SAME turn - should skip
        manager.start_turn("claude-sonnet-4-5")
        # Should still be 1 call
        assert mock_client.start_as_current_observation.call_count == 1

    @pytest.mark.asyncio
    async def test_sequential_turns_create_separate_traces(self):
        """Test that sequential turns create separate traces."""
        manager = ObservabilityManager(
            session_id="test-session", user_id="user-1", user_name="Test User"
        )

        # Mock Langfuse client
        mock_client = Mock()
        mock_ctx = Mock()
        mock_generation = Mock()
        mock_ctx.__enter__ = Mock(return_value=mock_generation)
        mock_ctx.__exit__ = Mock()
        mock_client.start_as_current_observation = Mock(return_value=mock_ctx)

        manager.langfuse_client = mock_client

        # Turn 1
        manager.start_turn("claude-sonnet-4-5")
        assert manager._current_turn_generation is not None
        assert mock_client.start_as_current_observation.call_count == 1

        # End turn 1 (clear generation)
        manager._current_turn_generation = None
        manager._current_turn_ctx = None

        # Turn 2 - should create new trace
        manager.start_turn("claude-sonnet-4-5")
        assert manager._current_turn_generation is not None
        assert mock_client.start_as_current_observation.call_count == 2

        # End turn 2
        manager._current_turn_generation = None
        manager._current_turn_ctx = None

        # Turn 3 - should create new trace
        manager.start_turn("claude-sonnet-4-5")
        assert manager._current_turn_generation is not None
        assert mock_client.start_as_current_observation.call_count == 3

    @pytest.mark.asyncio
    async def test_end_turn_adds_turn_number_to_metadata(self):
        """Test that end_turn adds SDK's authoritative turn number to metadata."""
        manager = ObservabilityManager(
            session_id="test-session", user_id="user-1", user_name="Test User"
        )

        # Mock Langfuse client and generation
        mock_client = Mock()
        mock_generation = Mock()
        mock_ctx = Mock()
        mock_ctx.__exit__ = Mock()

        manager.langfuse_client = mock_client
        manager._current_turn_generation = mock_generation
        manager._current_turn_ctx = mock_ctx

        # Create mock AssistantMessage
        mock_message = MagicMock()
        mock_message.content = []

        # End turn with SDK's turn number
        manager.end_turn(5, mock_message, usage={"input_tokens": 100, "output_tokens": 50})

        # Check that update was called with turn number in metadata
        mock_generation.update.assert_called_once()
        call_kwargs = mock_generation.update.call_args[1]
        assert "metadata" in call_kwargs
        assert call_kwargs["metadata"]["turn"] == 5

    @pytest.mark.asyncio
    async def test_no_prediction_just_sdk_turn_count(self):
        """Test that we use SDK's authoritative turn count, not predictions."""
        manager = ObservabilityManager(
            session_id="test-session", user_id="user-1", user_name="Test User"
        )

        # Mock Langfuse client
        mock_client = Mock()
        mock_ctx = Mock()
        mock_generation = Mock()
        mock_ctx.__enter__ = Mock(return_value=mock_generation)
        mock_ctx.__exit__ = Mock()
        mock_client.start_as_current_observation = Mock(return_value=mock_ctx)
        mock_client.flush = Mock()

        manager.langfuse_client = mock_client

        # Start turn without specifying turn number
        manager.start_turn("claude-sonnet-4-5")
        assert manager._current_turn_generation is not None
        assert mock_client.start_as_current_observation.call_count == 1

        # Second AssistantMessage arrives
        manager.start_turn("claude-sonnet-4-5")
        # Should be skipped - turn already active
        assert mock_client.start_as_current_observation.call_count == 1

        # SDK ResultMessage arrives with authoritative num_turns=2
        mock_message = MagicMock()
        mock_message.content = []

        manager.end_turn(2, mock_message, usage={"input_tokens": 100, "output_tokens": 50})

        # Check turn number was added to metadata
        call_kwargs = mock_generation.update.call_args[1]
        assert call_kwargs["metadata"]["turn"] == 2

        # Should have called flush
        assert mock_client.flush.call_count == 1
