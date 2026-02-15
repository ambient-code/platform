"""
AG-UI Tracing Middleware — Langfuse observability for the event stream.

Wraps an adapter's ``async for event in adapter.run(...)`` stream to:

1. Track turns, tool calls, and usage in Langfuse (via ObservabilityManager)
2. Emit a ``CustomEvent`` with the Langfuse trace ID once available
3. Finalise the turn trace on ``RUN_FINISHED``

Usage::

    from ambient_runner.middleware import tracing_middleware

    async for event in tracing_middleware(adapter.run(input), obs=obs, model=model, prompt=prompt):
        yield encoder.encode(event)

The middleware is transparent — if ``obs`` is ``None`` it simply yields events
unchanged with zero overhead.
"""

import logging
from typing import Any, AsyncIterator, Optional

from ag_ui.core import BaseEvent, CustomEvent, EventType

logger = logging.getLogger(__name__)


async def tracing_middleware(
    event_stream: AsyncIterator[BaseEvent],
    *,
    obs: Optional[Any] = None,
    model: str = "",
    prompt: str = "",
) -> AsyncIterator[BaseEvent]:
    """Wrap an AG-UI event stream with Langfuse tracing.

    Args:
        event_stream: The upstream adapter's event stream (``adapter.run(input)``).
        obs: An ``ObservabilityManager`` instance, or ``None`` to skip tracing.
        model: Model name for the Langfuse generation.
        prompt: User prompt (used as input for the first turn trace).

    Yields:
        The original events plus an ``ambient:langfuse_trace`` ``CustomEvent``
        once the Langfuse trace ID becomes available.
    """
    # Fast path: no observability — just pass through
    if obs is None:
        async for event in event_stream:
            yield event
        return

    # Initialise event-level tracking state
    obs.init_event_tracking(model, prompt)
    trace_id_emitted = False

    try:
        async for event in event_stream:
            # Side-channel: track for Langfuse (no event emission)
            obs.track_agui_event(event)

            # Yield the original event unchanged
            yield event

            # Emit trace ID as a CustomEvent once it becomes available.
            # The trace ID appears after the first TEXT_MESSAGE_START with
            # role=assistant triggers start_turn() inside the ObservabilityManager.
            if not trace_id_emitted:
                trace_id = obs.get_current_trace_id()
                if trace_id:
                    yield CustomEvent(
                        type=EventType.CUSTOM,
                        name="ambient:langfuse_trace",
                        value={"traceId": trace_id},
                    )
                    trace_id_emitted = True
                    logger.info(f"Tracing middleware: emitted trace ID {trace_id}")

    finally:
        # Safety-net: close any open turn that wasn't ended by RUN_FINISHED
        obs.finalize_event_tracking()
