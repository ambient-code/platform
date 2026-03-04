"""Unit tests for the developer events middleware."""

import pytest

from ag_ui.core import EventType

from ambient_runner.middleware.developer_events import emit_developer_message


@pytest.mark.asyncio
class TestEmitDeveloperMessage:
    """Test emit_developer_message yields correct AG-UI events."""

    async def test_yields_three_events(self):
        events = [e async for e in emit_developer_message("Hello")]
        assert len(events) == 3

    async def test_event_types_in_order(self):
        events = [e async for e in emit_developer_message("Hello")]
        assert events[0].type == EventType.TEXT_MESSAGE_START
        assert events[1].type == EventType.TEXT_MESSAGE_CONTENT
        assert events[2].type == EventType.TEXT_MESSAGE_END

    async def test_role_is_developer(self):
        events = [e async for e in emit_developer_message("Hello")]
        assert events[0].role == "developer"

    async def test_content_matches_input(self):
        text = "MCP servers initialised (3 connected)"
        events = [e async for e in emit_developer_message(text)]
        assert events[1].delta == text

    async def test_message_ids_consistent(self):
        events = [e async for e in emit_developer_message("Hello")]
        msg_id = events[0].message_id
        assert events[1].message_id == msg_id
        assert events[2].message_id == msg_id

    async def test_different_calls_get_different_ids(self):
        events_a = [e async for e in emit_developer_message("A")]
        events_b = [e async for e in emit_developer_message("B")]
        assert events_a[0].message_id != events_b[0].message_id

    async def test_single_char_text(self):
        events = [e async for e in emit_developer_message("x")]
        assert events[1].delta == "x"

    async def test_whitespace_text(self):
        events = [e async for e in emit_developer_message(" ")]
        assert events[1].delta == " "
