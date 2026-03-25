"""
Tests for GRPCMessageWriter.

Covers the event-accumulation and push logic, including edge cases that
caused production failures:

  - assistant message with content=None (tool-call-only turns where Claude
    emits no text; MESSAGES_SNAPSHOT contains {"role":"assistant","content":null})
  - no assistant message in snapshot at all
  - normal happy-path with text content
  - RUN_ERROR triggers push with status="error"
"""

import pytest
from unittest.mock import MagicMock

from ag_ui.core import EventType, RunFinishedEvent, RunErrorEvent

from ambient_runner.bridges.claude.grpc_transport import GRPCMessageWriter


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def make_writer(grpc_client=None):
    if grpc_client is None:
        grpc_client = MagicMock()
    return GRPCMessageWriter(
        session_id="sess-1",
        run_id="run-1",
        grpc_client=grpc_client,
    )


def make_snapshot_event(messages: list) -> MagicMock:
    evt = MagicMock()
    evt.type = EventType.MESSAGES_SNAPSHOT
    evt.messages = [_dict_to_mock(m) for m in messages]
    return evt


def _dict_to_mock(d: dict) -> MagicMock:
    m = MagicMock()
    m.model_dump.return_value = d
    return m


def make_run_finished() -> RunFinishedEvent:
    return RunFinishedEvent(
        type=EventType.RUN_FINISHED,
        thread_id="t-1",
        run_id="run-1",
    )


def make_run_error() -> RunErrorEvent:
    return RunErrorEvent(
        type=EventType.RUN_ERROR,
        message="something went wrong",
    )


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_write_message_none_content_raises_without_fix():
    """
    Regression: assistant message with content=None causes TypeError: object of
    type 'NoneType' has no len().

    Snapshot has role=assistant but content=null (tool-call-only turn).
    Before the fix this crashes; after the fix it should push an empty string.
    """
    client = MagicMock()
    writer = make_writer(client)

    snapshot = make_snapshot_event(
        [
            {"role": "user", "content": "i'm sending you a message"},
            {"role": "assistant", "content": None},
        ]
    )
    await writer.consume(snapshot)
    await writer.consume(make_run_finished())

    client.session_messages.push.assert_called_once_with(
        "sess-1",
        event_type="assistant",
        payload="",
    )


@pytest.mark.asyncio
async def test_write_message_no_assistant_in_snapshot():
    """No assistant message at all — push should still succeed with empty payload."""
    client = MagicMock()
    writer = make_writer(client)

    snapshot = make_snapshot_event(
        [
            {"role": "user", "content": "hello"},
        ]
    )
    await writer.consume(snapshot)
    await writer.consume(make_run_finished())

    client.session_messages.push.assert_called_once_with(
        "sess-1",
        event_type="assistant",
        payload="",
    )


@pytest.mark.asyncio
async def test_write_message_happy_path():
    """Normal turn: assistant has text content — push uses that text."""
    client = MagicMock()
    writer = make_writer(client)

    snapshot = make_snapshot_event(
        [
            {"role": "user", "content": "hi"},
            {"role": "assistant", "content": "Hello! I'm here."},
        ]
    )
    await writer.consume(snapshot)
    await writer.consume(make_run_finished())

    client.session_messages.push.assert_called_once_with(
        "sess-1",
        event_type="assistant",
        payload="Hello! I'm here.",
    )


@pytest.mark.asyncio
async def test_run_error_pushes_with_error_status():
    """RUN_ERROR triggers push with status='error'."""
    client = MagicMock()
    writer = make_writer(client)

    snapshot = make_snapshot_event(
        [
            {"role": "assistant", "content": "partial"},
        ]
    )
    await writer.consume(snapshot)
    await writer.consume(make_run_error())

    client.session_messages.push.assert_called_once_with(
        "sess-1",
        event_type="assistant",
        payload="partial",
    )


@pytest.mark.asyncio
async def test_latest_snapshot_wins():
    """Multiple MESSAGES_SNAPSHOT events — only the last one counts."""
    client = MagicMock()
    writer = make_writer(client)

    await writer.consume(
        make_snapshot_event(
            [
                {"role": "assistant", "content": "stale"},
            ]
        )
    )
    await writer.consume(
        make_snapshot_event(
            [
                {"role": "assistant", "content": "fresh"},
            ]
        )
    )
    await writer.consume(make_run_finished())

    client.session_messages.push.assert_called_once_with(
        "sess-1",
        event_type="assistant",
        payload="fresh",
    )


@pytest.mark.asyncio
async def test_no_push_without_run_finished():
    """Events before RUN_FINISHED/RUN_ERROR don't trigger a push."""
    client = MagicMock()
    writer = make_writer(client)

    await writer.consume(
        make_snapshot_event(
            [
                {"role": "assistant", "content": "something"},
            ]
        )
    )

    client.session_messages.push.assert_not_called()


@pytest.mark.asyncio
async def test_no_grpc_client_does_not_raise():
    """Writer with no gRPC client logs a warning and returns cleanly."""
    writer = make_writer(grpc_client=None)
    await writer.consume(make_snapshot_event([{"role": "assistant", "content": "x"}]))
    await writer.consume(make_run_finished())
