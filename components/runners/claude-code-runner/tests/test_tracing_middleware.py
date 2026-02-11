"""Unit tests for the tracing middleware."""

import pytest

from ag_ui.core import CustomEvent, EventType

from middleware.tracing import tracing_middleware
from tests.conftest import (
    MockObservabilityManager,
    async_event_stream,
    make_run_finished,
    make_run_started,
    make_text_content,
    make_text_end,
    make_text_start,
    make_tool_args,
    make_tool_end,
    make_tool_start,
)


@pytest.mark.asyncio
class TestTracingMiddlewarePassthrough:
    """When obs=None the middleware is a transparent pass-through."""

    async def test_yields_all_events_unchanged(self):
        events = [make_run_started(), make_text_start(), make_text_content(), make_text_end(), make_run_finished()]
        result = [e async for e in tracing_middleware(async_event_stream(events), obs=None)]
        assert len(result) == len(events)
        for orig, got in zip(events, result):
            assert orig is got  # same object reference

    async def test_empty_stream(self):
        result = [e async for e in tracing_middleware(async_event_stream([]), obs=None)]
        assert result == []


@pytest.mark.asyncio
class TestTracingMiddlewareObservability:
    """When obs is provided it should track events and emit a trace ID."""

    async def test_initialises_event_tracking(self):
        obs = MockObservabilityManager()
        events = [make_run_started()]
        _ = [e async for e in tracing_middleware(async_event_stream(events), obs=obs, model="claude-4", prompt="hi")]
        assert obs.init_event_tracking_calls == [("claude-4", "hi")]

    async def test_tracks_all_events(self):
        obs = MockObservabilityManager()
        events = [make_run_started(), make_text_start(), make_text_content(delta="yo"), make_text_end(), make_run_finished()]
        _ = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        assert len(obs.tracked_events) == len(events)

    async def test_emits_trace_id_custom_event_after_assistant_start(self):
        """Trace ID should appear as a CustomEvent right after the first assistant TEXT_MESSAGE_START."""
        obs = MockObservabilityManager(trace_id="trace-xyz")
        events = [
            make_run_started(),
            make_text_start(role="assistant"),
            make_text_content(delta="Hello"),
            make_text_end(),
            make_run_finished(),
        ]
        result = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]

        # Original 5 events + 1 trace CustomEvent
        assert len(result) == 6

        custom_events = [e for e in result if isinstance(e, CustomEvent)]
        assert len(custom_events) == 1
        assert custom_events[0].name == "ambient:langfuse_trace"
        assert custom_events[0].value == {"traceId": "trace-xyz"}

    async def test_trace_id_emitted_only_once(self):
        """Even with multiple assistant messages, the trace ID event should appear only once."""
        obs = MockObservabilityManager(trace_id="trace-once")
        events = [
            make_run_started(),
            make_text_start(msg_id="m1", role="assistant"),
            make_text_content(msg_id="m1", delta="First"),
            make_text_end(msg_id="m1"),
            make_text_start(msg_id="m2", role="assistant"),
            make_text_content(msg_id="m2", delta="Second"),
            make_text_end(msg_id="m2"),
            make_run_finished(),
        ]
        result = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        custom_events = [e for e in result if isinstance(e, CustomEvent)]
        assert len(custom_events) == 1

    async def test_no_trace_id_when_none(self):
        """If the ObservabilityManager never provides a trace ID, no custom event is emitted."""
        obs = MockObservabilityManager(trace_id=None)
        events = [
            make_run_started(),
            make_text_start(role="assistant"),
            make_text_content(delta="Hello"),
            make_text_end(),
            make_run_finished(),
        ]
        result = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        custom_events = [e for e in result if isinstance(e, CustomEvent)]
        assert len(custom_events) == 0

    async def test_no_trace_id_before_assistant_message(self):
        """The trace ID should not be emitted before the first assistant message."""
        obs = MockObservabilityManager(trace_id="trace-early")
        events = [
            make_run_started(),
            make_text_start(role="user"),  # user message, not assistant
            make_text_content(delta="Hello"),
            make_text_end(),
            make_run_finished(),
        ]
        result = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        custom_events = [e for e in result if isinstance(e, CustomEvent)]
        # trace_id is not emitted because the mock only returns it after assistant message
        assert len(custom_events) == 0

    async def test_finalizes_on_normal_completion(self):
        obs = MockObservabilityManager()
        events = [make_run_started(), make_run_finished()]
        _ = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        assert obs.finalize_called is True

    async def test_finalizes_on_error(self):
        """finalize_event_tracking is called even if the stream raises."""
        obs = MockObservabilityManager()

        async def failing_stream():
            yield make_run_started()
            raise RuntimeError("boom")

        with pytest.raises(RuntimeError, match="boom"):
            _ = [e async for e in tracing_middleware(failing_stream(), obs=obs)]

        assert obs.finalize_called is True

    async def test_preserves_event_order(self):
        obs = MockObservabilityManager(trace_id="trace-order")
        events = [
            make_run_started(),
            make_text_start(role="assistant"),
            make_tool_start(tool_id="tc-1", name="Read"),
            make_tool_args(tool_id="tc-1"),
            make_tool_end(tool_id="tc-1"),
            make_text_content(delta="Done"),
            make_text_end(),
            make_run_finished(),
        ]
        result = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]

        # Filter out the injected CustomEvent to verify original order
        original_events = [e for e in result if not isinstance(e, CustomEvent)]
        assert len(original_events) == len(events)
        for orig, got in zip(events, original_events):
            assert orig is got
