"""GET /events/{thread_id} — real-time SSE tap for an in-progress bridge.run() turn.

The backend opens this endpoint BEFORE calling PushSessionMessage (see §2.1 of
the gRPC message transport design). Opening first registers the queue in
bridge._active_streams[thread_id] so the gRPC fan-out cannot fire before a
subscriber exists — eliminating the race condition by ordering, not polling.

The GRPCSessionListener's fan-out loop feeds events into the queue.
This endpoint reads from the queue and yields them as SSE, filtering out
MESSAGES_SNAPSHOT (internal only — used by GRPCMessageWriter for the DB write).
The stream closes when RUN_FINISHED or RUN_ERROR is received.

Defensive fallback: if the queue is not yet registered when a client connects
(edge case: very slow lifespan startup), the endpoint polls _active_streams
with 100ms sleep intervals up to EVENTS_TAP_TIMEOUT_SEC (default 2s) before
returning 404.
"""

import asyncio
import logging
import os
from typing import AsyncIterator

from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import StreamingResponse

logger = logging.getLogger(__name__)

router = APIRouter()

_POLL_INTERVAL = 0.1
_TAP_TIMEOUT_SEC = float(os.getenv("EVENTS_TAP_TIMEOUT_SEC", "2"))

_CLOSE_TYPES = frozenset(["RUN_FINISHED", "RUN_ERROR"])
_FILTER_TYPES = frozenset(["MESSAGES_SNAPSHOT"])


def _event_type_str(event) -> str:
    raw = getattr(event, "type", None)
    if raw is None:
        return ""
    return raw.value if hasattr(raw, "value") else str(raw)


@router.get("/events/{thread_id}")
async def get_events(thread_id: str, request: Request):
    """SSE tap for an in-progress bridge.run() turn.

    Creates a bounded asyncio.Queue and registers it in
    bridge._active_streams[thread_id] before returning the SSE response.
    The GRPCSessionListener fan-out loop feeds events into the queue.
    """
    bridge = request.app.state.bridge
    active_streams: dict = getattr(bridge, "_active_streams", None)

    if active_streams is None:
        raise HTTPException(
            status_code=503, detail="Bridge does not support active streams"
        )

    queue: asyncio.Queue = asyncio.Queue(maxsize=100)
    active_streams[thread_id] = queue

    logger.info(
        "[SSE TAP] Queue registered: thread=%s (active_streams count=%d)",
        thread_id,
        len(active_streams),
    )

    async def event_stream() -> AsyncIterator[str]:
        try:
            while True:
                if await request.is_disconnected():
                    logger.info("[SSE TAP] Client disconnected: thread=%s", thread_id)
                    break

                try:
                    event = await asyncio.wait_for(queue.get(), timeout=30.0)
                except asyncio.TimeoutError:
                    yield ": heartbeat\n\n"
                    continue

                et = _event_type_str(event)

                if et in _FILTER_TYPES:
                    logger.debug("[SSE TAP] Filtered %s: thread=%s", et, thread_id)
                    continue

                try:
                    from ag_ui.encoder import EventEncoder

                    encoder = EventEncoder(accept="text/event-stream")
                    encoded = encoder.encode(event)
                    logger.info(
                        "[SSE TAP] Yielding event: thread=%s type=%s", thread_id, et
                    )
                    yield encoded
                except Exception as enc_err:
                    logger.warning(
                        "[SSE TAP] Encode error: thread=%s type=%s error=%s",
                        thread_id,
                        et,
                        enc_err,
                    )

                if et in _CLOSE_TYPES:
                    logger.info("[SSE TAP] Turn ended (%s): thread=%s", et, thread_id)
                    break
        finally:
            active_streams.pop(thread_id, None)
            logger.info("[SSE TAP] Queue removed: thread=%s", thread_id)

    return StreamingResponse(
        event_stream(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )


@router.get("/events/{thread_id}/wait")
async def wait_for_events(thread_id: str, request: Request):
    """Defensive fallback variant: polls _active_streams until the queue is
    registered (for edge cases where the listener hasn't started yet).
    Returns 404 after EVENTS_TAP_TIMEOUT_SEC.

    The primary path is GET /events/{thread_id} which registers the queue
    immediately on connection (before PushSessionMessage is called).
    """
    bridge = request.app.state.bridge
    active_streams: dict = getattr(bridge, "_active_streams", None)

    if active_streams is None:
        raise HTTPException(
            status_code=503, detail="Bridge does not support active streams"
        )

    elapsed = 0.0
    while elapsed < _TAP_TIMEOUT_SEC:
        if thread_id in active_streams:
            break
        await asyncio.sleep(_POLL_INTERVAL)
        elapsed += _POLL_INTERVAL

    if thread_id not in active_streams:
        logger.warning(
            "[SSE TAP WAIT] Timeout after %.1fs: thread=%s", elapsed, thread_id
        )
        raise HTTPException(
            status_code=404, detail=f"No active stream for thread {thread_id!r}"
        )

    queue = active_streams[thread_id]
    logger.info("[SSE TAP WAIT] Queue found after %.1fs: thread=%s", elapsed, thread_id)

    async def event_stream() -> AsyncIterator[str]:
        try:
            while True:
                if await request.is_disconnected():
                    break
                try:
                    event = await asyncio.wait_for(queue.get(), timeout=30.0)
                except asyncio.TimeoutError:
                    yield ": heartbeat\n\n"
                    continue

                et = _event_type_str(event)
                if et in _FILTER_TYPES:
                    continue

                try:
                    from ag_ui.encoder import EventEncoder

                    encoder = EventEncoder(accept="text/event-stream")
                    yield encoder.encode(event)
                except Exception as enc_err:
                    logger.warning("[SSE TAP WAIT] Encode error: %s", enc_err)

                if et in _CLOSE_TYPES:
                    break
        finally:
            active_streams.pop(thread_id, None)

    return StreamingResponse(
        event_stream(),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
