"""GET /events/{thread_id} — read-only SSE tap for live AG-UI events.

Registers an asyncio.Queue into bridge._active_streams[thread_id] before the
turn starts (or while it is running) and streams every AG-UI event to the
client as Server-Sent Events until RUN_FINISHED / RUN_ERROR or client
disconnect.

Zero side effects: this endpoint never calls bridge.run() or modifies any
state.  It is a pure observer of events already being produced by
GRPCSessionListener.
"""

import asyncio
import logging

from ag_ui.encoder import EventEncoder
from fastapi import APIRouter, Request
from fastapi.responses import StreamingResponse

logger = logging.getLogger(__name__)

router = APIRouter()

_SENTINEL = object()
_QUEUE_MAX = 256


@router.get("/events/{thread_id}")
async def stream_events(thread_id: str, request: Request):
    """SSE stream of all AG-UI events for a running turn on thread_id.

    Registers into bridge._active_streams so GRPCSessionListener fans events
    here in real-time.  Ends when RUN_FINISHED / RUN_ERROR is received or the
    client disconnects.
    """
    bridge = request.app.state.bridge
    active_streams: dict = getattr(bridge, "_active_streams", None)

    if active_streams is None:
        return {"error": "bridge does not support event streaming"}, 501

    accept_header = request.headers.get("accept", "text/event-stream")
    encoder = EventEncoder(accept=accept_header)

    queue: asyncio.Queue = asyncio.Queue(maxsize=_QUEUE_MAX)
    active_streams[thread_id] = queue

    logger.info("[SSE TAP] Client subscribed: thread=%s", thread_id)

    async def event_stream():
        try:
            while True:
                if await request.is_disconnected():
                    logger.info("[SSE TAP] Client disconnected: thread=%s", thread_id)
                    break

                try:
                    event = await asyncio.wait_for(queue.get(), timeout=30.0)
                except asyncio.TimeoutError:
                    yield ": keepalive\n\n"
                    continue

                if event is _SENTINEL:
                    break

                try:
                    yield encoder.encode(event)
                except Exception as enc_err:
                    logger.warning(
                        "[SSE TAP] Failed to encode event %s: %s",
                        type(event).__name__,
                        enc_err,
                    )

                raw_type = getattr(event, "type", None)
                if raw_type is not None:
                    type_str = raw_type.value if hasattr(raw_type, "value") else str(raw_type)
                    if type_str in ("RUN_FINISHED", "RUN_ERROR"):
                        logger.info(
                            "[SSE TAP] Turn ended (%s): thread=%s", type_str, thread_id
                        )
                        break
        finally:
            active_streams.pop(thread_id, None)
            logger.info("[SSE TAP] Stream closed: thread=%s", thread_id)

    return StreamingResponse(
        event_stream(),
        media_type=encoder.get_content_type(),
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
