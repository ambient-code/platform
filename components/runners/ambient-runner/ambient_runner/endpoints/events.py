"""GET /events — persistent SSE for between-run AG-UI events."""

import asyncio
import logging

from ag_ui.encoder import EventEncoder
from fastapi import APIRouter, Request
from fastapi.responses import StreamingResponse

logger = logging.getLogger(__name__)

router = APIRouter()

# Heartbeat interval to keep the SSE connection alive (seconds).
_HEARTBEAT_INTERVAL = 15


@router.get("/events")
async def stream_events(request: Request):
    """Persistent SSE endpoint for between-run events.

    Streams AG-UI events that arrive outside of user-initiated runs
    (background task completions, hook notifications, agent responses
    to task results).
    """
    bridge = request.app.state.bridge
    ctx = getattr(bridge, "_context", None)
    thread_id = ctx.session_id if ctx else ""

    encoder = EventEncoder(accept="text/event-stream")

    async def event_stream():
        try:
            event_iter = bridge.stream_between_run_events(thread_id)

            while True:
                if await request.is_disconnected():
                    break

                try:
                    event = await asyncio.wait_for(
                        event_iter.__anext__(),
                        timeout=_HEARTBEAT_INTERVAL,
                    )
                    yield encoder.encode(event)
                except StopAsyncIteration:
                    break
                except asyncio.TimeoutError:
                    # No event within heartbeat interval — send keepalive
                    yield ": heartbeat\n\n"
                except Exception as e:
                    logger.error(f"Error in between-run event stream: {e}")
                    break
        except Exception as e:
            logger.error(f"Fatal error in /events stream: {e}", exc_info=True)

    return StreamingResponse(
        event_stream(),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
