"""POST / — AG-UI run endpoint (delegates to bridge)."""

import asyncio
import json
import logging
import uuid
from concurrent.futures import ThreadPoolExecutor
from typing import Any, Dict, List, Optional, Union

from ag_ui.core import EventType, RunAgentInput, RunErrorEvent, ToolCallResultEvent
from ag_ui_claude_sdk.utils import now_ms
from ag_ui.encoder import EventEncoder
from fastapi import APIRouter, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel

logger = logging.getLogger(__name__)

# gRPC is opt-in via AMBIENT_GRPC_URL.
#
# The existing Operator path is completely unaffected: operator-created Job pods
# never have AMBIENT_GRPC_URL injected, so _grpc_client is always None for them.
# Every _push_event() call below checks `if _grpc_client is None: return` first,
# making it a guaranteed no-op on the Operator path.
#
# gRPC is only active when the ambient-control-plane injects AMBIENT_GRPC_URL,
# which is the new CP-managed session path introduced in this PR.
try:
    from ambient_runner._grpc_client import AmbientGRPCClient

    _grpc_client: Optional[AmbientGRPCClient] = AmbientGRPCClient.from_env()
except Exception as _grpc_init_err:
    logger.warning(
        "gRPC client unavailable, session messages will not be pushed: %s",
        _grpc_init_err,
    )
    _grpc_client = None


def _push_event(session_id: Optional[str], event: Any) -> None:
    """Push an AG-UI MESSAGES_SNAPSHOT event to the session messages stream.

    Only MESSAGES_SNAPSHOT and RUN_FINISHED events are persisted.
    MESSAGES_SNAPSHOT contains the full structured conversation after each run.
    RUN_FINISHED is the completion signal consumers (e.g. acpctl) wait on.
    All other AG-UI events (per-token deltas, RUN_STARTED, etc.) are streamed to SSE
    consumers only and are not stored.

    Best-effort; never raises.
    """
    if _grpc_client is None or not session_id:
        return
    try:
        event_type = getattr(event, "type", None)
        if event_type is None:
            return
        event_type_str = (
            event_type.value if hasattr(event_type, "value") else str(event_type)
        )
        if event_type_str not in ("MESSAGES_SNAPSHOT", "RUN_FINISHED"):
            logger.debug(
                "[OUTBOUND SSE] Skipping gRPC push for event_type=%s (SSE only)",
                event_type_str,
            )
            return
        try:
            messages = getattr(event, "messages", None)
            if messages is not None:
                serializable = [
                    m.model_dump(exclude_none=True)
                    if hasattr(m, "model_dump")
                    else m.dict(exclude_none=True)
                    if hasattr(m, "dict")
                    else m
                    if isinstance(m, dict)
                    else str(m)
                    for m in messages
                ]
                payload = json.dumps(serializable, default=str)
            elif hasattr(event, "model_dump"):
                payload = json.dumps(event.model_dump(exclude_none=True), default=str)
            elif hasattr(event, "dict"):
                payload = json.dumps(event.dict(exclude_none=True), default=str)
            else:
                payload = json.dumps(event, default=str)
        except Exception:
            payload = str(event)
        logger.info(
            "[OUTBOUND GRPC] Pushing event: session=%s event_type=%s payload_len=%d",
            session_id,
            event_type_str,
            len(payload),
        )
        _grpc_client.session_messages.push(session_id, event_type_str, payload)
    except Exception as exc:
        logger.warning("[OUTBOUND GRPC] _push_event failed: %s", exc)


def _watch_inbound_messages(
    session_id: str,
    queue: asyncio.Queue,
    loop: asyncio.AbstractEventLoop,
    stop_event: asyncio.Event,
) -> None:
    """Run the blocking gRPC WatchSessionMessages stream in a thread.

    Puts incoming SessionMessage objects onto the asyncio queue.
    Stops when stop_event is set or the stream ends.
    """
    if _grpc_client is None:
        return
    try:
        for msg in _grpc_client.session_messages.watch(session_id):
            if loop.is_closed():
                break
            asyncio.run_coroutine_threadsafe(queue.put(msg), loop)
            if stop_event.is_set():
                break
    except Exception as exc:
        logger.debug(
            "WatchSessionMessages stream ended (session=%s): %s", session_id, exc
        )


router = APIRouter()


class RunnerInput(BaseModel):
    """Input model with optional AG-UI fields."""

    threadId: Optional[str] = None
    thread_id: Optional[str] = None
    runId: Optional[str] = None
    run_id: Optional[str] = None
    parentRunId: Optional[str] = None
    parent_run_id: Optional[str] = None
    messages: List[Dict[str, Any]]
    state: Optional[Dict[str, Any]] = None
    tools: Optional[List[Any]] = None
    context: Optional[Union[List[Any], Dict[str, Any]]] = None
    forwardedProps: Optional[Dict[str, Any]] = None
    environment: Optional[Dict[str, str]] = None
    metadata: Optional[Dict[str, Any]] = None

    def to_run_agent_input(self) -> RunAgentInput:
        thread_id = self.threadId or self.thread_id
        run_id = self.runId or self.run_id or str(uuid.uuid4())
        parent_run_id = self.parentRunId or self.parent_run_id
        context_list = self.context if isinstance(self.context, list) else []

        return RunAgentInput(
            thread_id=thread_id,
            run_id=run_id,
            parent_run_id=parent_run_id,
            messages=self.messages,
            state=self.state or {},
            tools=self.tools or [],
            context=context_list,
            forwarded_props=self.forwardedProps or {},
        )


@router.post("/")
async def run_agent(input_data: RunnerInput, request: Request):
    """AG-UI run endpoint — delegates to the bridge."""
    bridge = request.app.state.bridge

    run_agent_input = input_data.to_run_agent_input()
    accept_header = request.headers.get("accept", "text/event-stream")
    encoder = EventEncoder(accept=accept_header)
    session_id = run_agent_input.thread_id

    msg_count = len(input_data.messages)
    last_role = (
        input_data.messages[-1].get("role", "?") if input_data.messages else "(none)"
    )
    last_content_preview = ""
    if input_data.messages:
        raw = input_data.messages[-1].get("content", "")
        text = raw if isinstance(raw, str) else str(raw)
        last_content_preview = text[:100] + "..." if len(text) > 100 else text
    logger.info(
        "[► RUN START] thread_id=%s run_id=%s msg_count=%d last_role=%s last_content=%r",
        run_agent_input.thread_id,
        run_agent_input.run_id,
        msg_count,
        last_role,
        last_content_preview,
    )

    inbound_queue: asyncio.Queue = asyncio.Queue()
    stop_watch = asyncio.Event()
    async def event_stream():
        executor = ThreadPoolExecutor(max_workers=1)
        event_count = 0
        grpc_pushed = 0

        # Per-request inbound gRPC watch is intentionally disabled.
        #
        # Previously this opened a second concurrent WatchSessionMessages stream
        # alongside the pod-lifetime watcher in app.py, causing a race: both
        # watchers received the same inbound "user" messages simultaneously —
        # app.py would POST to / (triggering a new run) while run.py would call
        # bridge.inject_message() (a no-op in all bridges) for the same message.
        #
        # Inbound message routing is now owned exclusively by app.py:
        #   - "user" messages  → app.py background watcher → POST / (new run)
        #   - interrupt signal → POST /interrupt (direct HTTP, bypasses gRPC)
        #
        # If mid-run gRPC injection is needed in the future, implement
        # bridge.inject_message() in the bridge subclass first, then re-enable
        # this watcher — but filter by a dedicated event_type (e.g. "interrupt")
        # so it does not race with app.py's "user" handler.
        #
        # if _grpc_client is not None and session_id:
        #     watch_future = loop.run_in_executor(
        #         executor,
        #         _watch_inbound_messages,
        #         session_id,
        #         inbound_queue,
        #         loop,
        #         stop_watch,
        #     )

        async def drain_inbound():
            while True:
                try:
                    msg = inbound_queue.get_nowait()
                    try:
                        await bridge.inject_message(
                            msg.session_id, msg.event_type, msg.payload
                        )
                    except Exception as exc:
                        logger.debug("inject_message failed: %s", exc)
                except asyncio.QueueEmpty:
                    break

        try:
            logger.info("[RUN] bridge.run() starting for session=%s", session_id)
            async for event in bridge.run(run_agent_input):
                await drain_inbound()
                event_count += 1
                event_type_str = ""
                raw_et = getattr(event, "type", None)
                if raw_et is not None:
                    event_type_str = (
                        raw_et.value if hasattr(raw_et, "value") else str(raw_et)
                    )
                is_persistent = event_type_str in ("MESSAGES_SNAPSHOT", "RUN_FINISHED")
                if is_persistent:
                    grpc_pushed += 1
                logger.info(
                    "[OUTBOUND SSE] event #%d type=%s persistent=%s session=%s",
                    event_count,
                    event_type_str,
                    is_persistent,
                    session_id,
                )
                _push_event(session_id, event)
                try:
                    yield encoder.encode(event)
                except Exception as encode_err:
                    logger.warning(
                        "Failed to encode %s event: %s",
                        type(event).__name__,
                        encode_err,
                    )
                    tool_call_id = getattr(event, "tool_call_id", None)
                    if tool_call_id:
                        fallback = ToolCallResultEvent(
                            type=EventType.TOOL_CALL_RESULT,
                            thread_id=getattr(event, "thread_id", "") or "",
                            run_id=getattr(event, "run_id", "") or "",
                            message_id=f"{tool_call_id}-result",
                            tool_call_id=tool_call_id,
                            role="tool",
                            content=(
                                f"[Tool result too large to display: {encode_err}]"
                            ),
                        )
                        yield encoder.encode(fallback)
                    else:
                        yield encoder.encode(
                            RunErrorEvent(
                                type=EventType.RUN_ERROR,
                                thread_id=getattr(event, "thread_id", "")
                                or run_agent_input.thread_id
                                or "",
                                run_id=getattr(event, "run_id", "")
                                or run_agent_input.run_id
                                or "unknown",
                                message=f"An event was too large to send ({type(event).__name__}: {encode_err})",
                                timestamp=now_ms(),
                            )
                        )
        except Exception as e:
            logger.error(
                "[◄ RUN ERROR] session=%s error=%s", session_id, e, exc_info=True
            )

            error_msg = str(e)
            extra = bridge.get_error_context()
            if extra:
                error_msg = f"{error_msg}\n\n{extra}"

            yield encoder.encode(
                RunErrorEvent(
                    type=EventType.RUN_ERROR,
                    thread_id=run_agent_input.thread_id or "",
                    run_id=run_agent_input.run_id or "unknown",
                    message=error_msg,
                    timestamp=now_ms(),
                )
            )
        finally:
            logger.info(
                "[◄ RUN COMPLETE] session=%s total_events=%d grpc_pushed=%d",
                session_id,
                event_count,
                grpc_pushed,
            )
            stop_watch.set()
            executor.shutdown(wait=False)

    return StreamingResponse(
        event_stream(),
        media_type=encoder.get_content_type(),
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
