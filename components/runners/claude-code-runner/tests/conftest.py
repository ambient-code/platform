"""
Shared fixtures for runner unit tests.
"""

import asyncio
from dataclasses import dataclass, field
from typing import Any, AsyncIterator, Optional
from unittest.mock import AsyncMock, MagicMock, Mock

import pytest

from ag_ui.core import (
    BaseEvent,
    CustomEvent,
    EventType,
    RunFinishedEvent,
    RunStartedEvent,
    TextMessageContentEvent,
    TextMessageEndEvent,
    TextMessageStartEvent,
    ToolCallArgsEvent,
    ToolCallEndEvent,
    ToolCallStartEvent,
)


# ------------------------------------------------------------------
# AG-UI event factories
# ------------------------------------------------------------------


def make_run_started(thread_id: str = "t-1", run_id: str = "r-1") -> RunStartedEvent:
    return RunStartedEvent(
        type=EventType.RUN_STARTED,
        thread_id=thread_id,
        run_id=run_id,
    )


def make_text_start(msg_id: str = "m-1", role: str = "assistant") -> TextMessageStartEvent:
    return TextMessageStartEvent(
        type=EventType.TEXT_MESSAGE_START,
        message_id=msg_id,
        role=role,
    )


def make_text_content(msg_id: str = "m-1", delta: str = "Hello") -> TextMessageContentEvent:
    return TextMessageContentEvent(
        type=EventType.TEXT_MESSAGE_CONTENT,
        message_id=msg_id,
        delta=delta,
    )


def make_text_end(msg_id: str = "m-1") -> TextMessageEndEvent:
    return TextMessageEndEvent(
        type=EventType.TEXT_MESSAGE_END,
        message_id=msg_id,
    )


def make_tool_start(tool_id: str = "tc-1", name: str = "Read") -> ToolCallStartEvent:
    return ToolCallStartEvent(
        type=EventType.TOOL_CALL_START,
        tool_call_id=tool_id,
        tool_call_name=name,
    )


def make_tool_args(tool_id: str = "tc-1", delta: str = '{"file":"x"}') -> ToolCallArgsEvent:
    return ToolCallArgsEvent(
        type=EventType.TOOL_CALL_ARGS,
        tool_call_id=tool_id,
        delta=delta,
    )


def make_tool_end(tool_id: str = "tc-1") -> ToolCallEndEvent:
    return ToolCallEndEvent(
        type=EventType.TOOL_CALL_END,
        tool_call_id=tool_id,
    )


def make_run_finished(thread_id: str = "t-1", run_id: str = "r-1") -> RunFinishedEvent:
    return RunFinishedEvent(
        type=EventType.RUN_FINISHED,
        thread_id=thread_id,
        run_id=run_id,
    )


async def async_event_stream(events: list[BaseEvent]) -> AsyncIterator[BaseEvent]:
    """Turn a list of events into an async iterator (simulates adapter.run())."""
    for event in events:
        yield event


# ------------------------------------------------------------------
# Mock ObservabilityManager
# ------------------------------------------------------------------


class MockObservabilityManager:
    """Lightweight mock of ObservabilityManager for middleware tests."""

    def __init__(self, trace_id: str | None = "trace-abc-123"):
        self._trace_id = trace_id
        self.init_event_tracking_calls: list[tuple[str, str]] = []
        self.tracked_events: list[BaseEvent] = []
        self.finalize_called = False
        # Simulate: trace ID only available after a TEXT_MESSAGE_START (assistant)
        self._turn_started = False

    def init_event_tracking(self, model: str, prompt: str) -> None:
        self.init_event_tracking_calls.append((model, prompt))

    def track_agui_event(self, event: BaseEvent) -> None:
        self.tracked_events.append(event)
        if getattr(event, "type", None) == EventType.TEXT_MESSAGE_START:
            if getattr(event, "role", "") == "assistant":
                self._turn_started = True

    def get_current_trace_id(self) -> str | None:
        return self._trace_id if self._turn_started else None

    def finalize_event_tracking(self) -> None:
        self.finalize_called = True
