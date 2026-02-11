"""Unit tests for the developer events middleware."""

import pytest

from ag_ui.core import EventType

from middleware.developer_events import emit_developer_message


@pytest.mark.asyncio
class TestEmitDeveloperMessage:
    """Tests for emit_developer_message async generator."""

    async def test_yields_three_events(self):
        """Should emit START, CONTENT, END for every message."""
        events = [e async for e in emit_developer_message("Auth connected")]
        assert len(events) == 3

    async def test_event_types_in_order(self):
        events = [e async for e in emit_developer_message("Hello")]
        assert events[0].type == EventType.TEXT_MESSAGE_START
        assert events[1].type == EventType.TEXT_MESSAGE_CONTENT
        assert events[2].type == EventType.TEXT_MESSAGE_END

    async def test_role_is_developer(self):
        events = [e async for e in emit_developer_message("test")]
        assert events[0].role == "developer"

    async def test_content_matches_input(self):
        text = "MCP servers initialised (3 connected)"
        events = [e async for e in emit_developer_message(text)]
        assert events[1].delta == text

    async def test_message_ids_consistent(self):
        """All three events should share the same message_id."""
        events = [e async for e in emit_developer_message("test")]
        msg_id = events[0].message_id
        assert msg_id  # not empty
        assert events[1].message_id == msg_id
        assert events[2].message_id == msg_id

    async def test_different_calls_get_different_ids(self):
        events_a = [e async for e in emit_developer_message("first")]
        events_b = [e async for e in emit_developer_message("second")]
        assert events_a[0].message_id != events_b[0].message_id

    async def test_single_char_text(self):
        """Minimal text should produce valid events (AG-UI requires min_length=1 for delta)."""
        events = [e async for e in emit_developer_message("x")]
        assert len(events) == 3
        assert events[1].delta == "x"
