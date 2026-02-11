"""Unit tests for the tracing middleware."""

import pytest

from ag_ui.core import CustomEvent, EventType

from ambient_runner.middleware.tracing import tracing_middleware
from tests.conftest import (
    MockObservabilityManager,
    async_event_stream,
    make_run_finished,
    make_run_started,
    make_text_content,
    make_text_end,
    make_text_start,
)


@pytest.mark.asyncio
class TestTracingMiddlewarePassthrough:
    """With obs=None, middleware should pass events through unchanged."""

    async def test_yields_all_events(self):
        events_in = [make_run_started(), make_text_start(), make_run_finished()]
        events_out = [e async for e in tracing_middleware(async_event_stream(events_in))]
        assert len(events_out) == 3

    async def test_events_unchanged(self):
        events_in = [make_run_started(), make_text_content(delta="Hello")]
        events_out = [e async for e in tracing_middleware(async_event_stream(events_in))]
        assert events_out[0] is events_in[0]
        assert events_out[1] is events_in[1]

    async def test_empty_stream(self):
        events_out = [e async for e in tracing_middleware(async_event_stream([]))]
        assert events_out == []


@pytest.mark.asyncio
class TestTracingMiddlewareObservability:
    """With obs provided, middleware should track events and emit trace ID."""

    async def test_calls_init_event_tracking(self):
        obs = MockObservabilityManager()
        events = [make_run_started()]
        _ = [e async for e in tracing_middleware(
            async_event_stream(events), obs=obs, model="claude-4", prompt="Hello"
        )]
        assert obs.init_event_tracking_calls == [("claude-4", "Hello")]

    async def test_tracks_all_events(self):
        obs = MockObservabilityManager()
        events = [make_run_started(), make_text_start(), make_text_content(), make_run_finished()]
        _ = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        assert len(obs.tracked_events) == 4

    async def test_emits_trace_id_custom_event_after_assistant_start(self):
        """Trace ID CustomEvent should appear after the first assistant TEXT_MESSAGE_START."""
        obs = MockObservabilityManager(trace_id="trace-abc")
        events = [
            make_run_started(),
            make_text_start(role="assistant"),
            make_text_content(),
            make_text_end(),
            make_run_finished(),
        ]
        events_out = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]

        # Original 5 events + 1 CustomEvent = 6
        assert len(events_out) == 6

        custom_events = [e for e in events_out if isinstance(e, CustomEvent)]
        assert len(custom_events) == 1
        assert custom_events[0].name == "ambient:langfuse_trace"
        assert custom_events[0].value == {"traceId": "trace-abc"}

    async def test_trace_id_emitted_only_once(self):
        """Even with multiple assistant messages, trace ID is emitted once."""
        obs = MockObservabilityManager(trace_id="trace-abc")
        events = [
            make_run_started(),
            make_text_start(msg_id="m1", role="assistant"),
            make_text_end(msg_id="m1"),
            make_text_start(msg_id="m2", role="assistant"),
            make_text_end(msg_id="m2"),
            make_run_finished(),
        ]
        events_out = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        custom_events = [e for e in events_out if isinstance(e, CustomEvent)]
        assert len(custom_events) == 1

    async def test_no_trace_id_when_none(self):
        """If obs has no trace ID, no CustomEvent should be emitted."""
        obs = MockObservabilityManager(trace_id=None)
        events = [make_run_started(), make_text_start(role="assistant"), make_run_finished()]
        events_out = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        custom_events = [e for e in events_out if isinstance(e, CustomEvent)]
        assert len(custom_events) == 0

    async def test_no_trace_id_before_assistant_message(self):
        """Trace ID should not appear before an assistant message."""
        obs = MockObservabilityManager(trace_id="trace-abc")
        events = [make_run_started(), make_run_finished()]
        events_out = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        custom_events = [e for e in events_out if isinstance(e, CustomEvent)]
        assert len(custom_events) == 0

    async def test_finalize_called(self):
        obs = MockObservabilityManager()
        events = [make_run_started(), make_run_finished()]
        _ = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]
        assert obs.finalize_called is True

    async def test_preserves_event_order(self):
        """Original events should maintain their order around the injected CustomEvent."""
        obs = MockObservabilityManager(trace_id="trace-abc")
        events = [
            make_run_started(),
            make_text_start(role="assistant"),
            make_text_content(delta="Hi"),
            make_text_end(),
            make_run_finished(),
        ]
        events_out = [e async for e in tracing_middleware(async_event_stream(events), obs=obs)]

        # Filter out the custom event
        non_custom = [e for e in events_out if not isinstance(e, CustomEvent)]
        assert len(non_custom) == 5
        assert non_custom[0].type == EventType.RUN_STARTED
        assert non_custom[1].type == EventType.TEXT_MESSAGE_START
        assert non_custom[2].type == EventType.TEXT_MESSAGE_CONTENT
        assert non_custom[3].type == EventType.TEXT_MESSAGE_END
        assert non_custom[4].type == EventType.RUN_FINISHED
