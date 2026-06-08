"""
gRPC transport for ClaudeBridge (additive — only active when AMBIENT_GRPC_ENABLED=true).

GRPCSessionListener — pod-lifetime WatchSessionMessages subscriber.
  Active alongside the existing HTTP/SSE path when AMBIENT_GRPC_ENABLED=true.
  Calls bridge.run() directly for each inbound user message (no HTTP round-trip).
  Fans out each event to:
    (a) bridge._active_streams[thread_id] queue — feeds the /events SSE tap
    (b) GRPCMessageWriter — assembles and writes the durable DB record

GRPCMessageWriter — real-time assistant text writer.
  Accumulates TEXT_MESSAGE_CONTENT deltas, pushes on TEXT_MESSAGE_END.
  Each assistant turn gets a server-assigned seq at its correct chronological
  position relative to tool events. push_error() handles crash recovery.

When AMBIENT_GRPC_ENABLED is not set, none of this code is instantiated or called.
"""

import asyncio
import logging
import os
import time
import uuid
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from typing import TYPE_CHECKING, Any, Optional

import grpc

from ag_ui.core import BaseEvent

from .operational_events import OperationalEventWriter

if TYPE_CHECKING:
    from ambient_runner._grpc_client import AmbientGRPCClient
    from ambient_runner.bridge import PlatformBridge

logger = logging.getLogger(__name__)

_BACKOFF_INITIAL = 1.0
_BACKOFF_MAX = 30.0


def _synthesize_run_error(
    thread_id: str,
    error_message: str,
    active_streams: dict[str, asyncio.Queue],
    writer: "GRPCMessageWriter",
) -> None:
    """Synthesize a terminal RUN_ERROR event when bridge.run() raises.

    Feeds the error event into the SSE tap queue (if registered) and
    schedules the writer to persist an 'error' status record so neither
    the SSE consumer nor the DB writer is left hanging.
    """
    from ag_ui.core import RunErrorEvent

    try:
        error_event = RunErrorEvent(message=error_message, code="RUNNER_ERROR")
    except Exception:
        error_event = None

    stream_queue = active_streams.get(thread_id)
    if stream_queue is not None and error_event is not None:
        try:
            stream_queue.put_nowait(error_event)
        except asyncio.QueueFull:
            logger.warning(
                "[GRPC LISTENER] SSE tap queue full while synthesising RUN_ERROR: thread=%s",
                thread_id,
            )

    task = asyncio.ensure_future(writer.push_error(error_message))

    def _log_write_error(f: asyncio.Future) -> None:
        if not f.cancelled() and f.exception() is not None:
            logger.warning(
                "[GRPC LISTENER] push_error failed: %s", f.exception()
            )

    task.add_done_callback(_log_write_error)


class GRPCSessionListener:
    """Pod-lifetime gRPC session listener for ClaudeBridge.

    Subscribes to WatchSessionMessages for this session. For each inbound
    message with event_type=="user", parses the payload as RunnerInput and
    calls bridge.run() directly.

    ready: asyncio.Event — set once the WatchSessionMessages stream is open.
    Callers should await self.ready.wait() before sending the first message.
    """

    def __init__(
        self,
        bridge: "PlatformBridge",
        session_id: str,
        grpc_url: str,
    ) -> None:
        self._bridge = bridge
        self._session_id = session_id
        self._grpc_url = grpc_url
        self._grpc_client: Optional["AmbientGRPCClient"] = None
        self.ready = asyncio.Event()
        self._task: Optional[asyncio.Task] = None

    def start(self) -> None:
        from ambient_runner._grpc_client import AmbientGRPCClient

        self._grpc_client = AmbientGRPCClient.from_env()
        self._task = asyncio.create_task(
            self._listen_loop(), name="grpc-session-listener"
        )
        logger.info(
            "[GRPC LISTENER] Started: session=%s url=%s",
            self._session_id,
            self._grpc_url,
        )

    async def stop(self) -> None:
        if self._task and not self._task.done():
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        if self._grpc_client:
            self._grpc_client.close()
        logger.info("[GRPC LISTENER] Stopped: session=%s", self._session_id)

    def _watch_in_thread(
        self,
        msg_queue: asyncio.Queue,
        loop: asyncio.AbstractEventLoop,
        stop_event: asyncio.Event,
        last_seq: int,
    ) -> None:
        """Blocking gRPC watch — runs in a ThreadPoolExecutor.

        Sets self.ready after watch() returns the stream iterator (stream open,
        server will deliver messages from this point). Puts each received
        SessionMessage onto msg_queue via run_coroutine_threadsafe.
        """
        if self._grpc_client is None:
            return
        try:
            stream = self._grpc_client.session_messages.watch(
                self._session_id, after_seq=last_seq
            )
            loop.call_soon_threadsafe(self.ready.set)
            logger.info(
                "[GRPC LISTENER] WatchSessionMessages stream open: session=%s after_seq=%d",
                self._session_id,
                last_seq,
            )
            for msg in stream:
                if loop.is_closed() or stop_event.is_set():
                    break
                logger.info(
                    "[GRPC LISTENER] Received: session=%s seq=%d event_type=%s",
                    self._session_id,
                    msg.seq,
                    msg.event_type,
                )
                asyncio.run_coroutine_threadsafe(msg_queue.put(msg), loop)
        except grpc.RpcError as exc:
            logger.warning(
                "[GRPC LISTENER] gRPC stream error: session=%s code=%s details=%s",
                self._session_id,
                exc.code(),
                exc.details(),
            )
            if (
                exc.code() == grpc.StatusCode.UNAUTHENTICATED
                and self._grpc_client is not None
            ):
                logger.warning(
                    "[GRPC LISTENER] UNAUTHENTICATED — reconnecting with fresh token: session=%s",
                    self._session_id,
                )
                self._grpc_client.reconnect()
        except Exception as exc:
            logger.error(
                "[GRPC LISTENER] Unexpected watch error: session=%s error=%s",
                self._session_id,
                exc,
                exc_info=True,
            )

    async def _listen_loop(self) -> None:
        is_resume = os.getenv("IS_RESUME", "").strip().lower() == "true"
        # Record start time for filtering historical user messages on resume.
        # Use a 5-second grace window to handle clock skew between the runner
        # pod and the API server that stamps created_at.
        resume_cutoff: float | None = None
        if is_resume:
            resume_cutoff = time.time() - 5.0
            logger.info(
                "[GRPC LISTENER] Resume mode: will skip user messages created before %.3f: session=%s",
                resume_cutoff,
                self._session_id,
            )

        last_seq = 0
        backoff = _BACKOFF_INITIAL

        while True:
            msg_queue: asyncio.Queue = asyncio.Queue()
            stop_event = asyncio.Event()
            loop = asyncio.get_running_loop()
            executor = ThreadPoolExecutor(max_workers=1)

            watch_future = loop.run_in_executor(
                executor,
                self._watch_in_thread,
                msg_queue,
                loop,
                stop_event,
                last_seq,
            )

            try:
                while True:
                    try:
                        msg = await asyncio.wait_for(msg_queue.get(), timeout=30.0)
                    except asyncio.TimeoutError:
                        if watch_future.done():
                            break
                        continue

                    last_seq = max(last_seq, msg.seq)

                    if msg.event_type != "user":
                        logger.debug(
                            "[GRPC LISTENER] Skipping event_type=%s seq=%d",
                            msg.event_type,
                            msg.seq,
                        )
                        continue

                    # On resume, skip historical user messages that existed
                    # before this runner pod started.
                    if resume_cutoff is not None and msg.created_at is not None:
                        msg_epoch = msg.created_at.timestamp()
                        if msg_epoch < resume_cutoff:
                            logger.info(
                                "[GRPC LISTENER] Skipping historical user message: seq=%d created_at=%.3f < cutoff=%.3f session=%s",
                                msg.seq,
                                msg_epoch,
                                resume_cutoff,
                                self._session_id,
                            )
                            continue

                    logger.info(
                        "[GRPC LISTENER] User message seq=%d — triggering run: session=%s",
                        msg.seq,
                        self._session_id,
                    )
                    await self._handle_user_message(msg)

            except asyncio.CancelledError:
                stop_event.set()
                executor.shutdown(wait=False)
                logger.info("[GRPC LISTENER] Cancelled: session=%s", self._session_id)
                raise
            except Exception as exc:
                stop_event.set()
                executor.shutdown(wait=False)
                logger.warning(
                    "[GRPC LISTENER] Error, reconnecting in %.1fs: session=%s error=%s",
                    backoff,
                    self._session_id,
                    exc,
                )
                await asyncio.sleep(backoff)
                backoff = min(backoff * 2, _BACKOFF_MAX)
                continue

            stop_event.set()
            executor.shutdown(wait=False)
            backoff = _BACKOFF_INITIAL
            logger.info(
                "[GRPC LISTENER] Stream ended cleanly, reconnecting: session=%s last_seq=%d",
                self._session_id,
                last_seq,
            )

    async def _handle_user_message(self, msg: Any) -> None:
        """Parse a user message payload and drive a full bridge.run() turn."""
        from ambient_runner.endpoints.run import RunnerInput

        try:
            runner_input = RunnerInput.model_validate_json(msg.payload)
        except Exception:
            runner_input = RunnerInput(
                messages=[
                    {"id": str(uuid.uuid4()), "role": "user", "content": msg.payload}
                ],
                thread_id=self._session_id,
            )

        try:
            input_data = runner_input.to_run_agent_input()
        except Exception as exc:
            logger.warning(
                "[GRPC LISTENER] Failed to build run agent input: seq=%d error=%s",
                msg.seq,
                exc,
            )
            return

        thread_id = input_data.thread_id or self._session_id
        run_id = str(input_data.run_id) if input_data.run_id else str(uuid.uuid4())

        writer = GRPCMessageWriter(
            session_id=self._session_id,
            run_id=run_id,
            grpc_client=self._grpc_client,
        )
        ops_writer = OperationalEventWriter(
            session_id=self._session_id,
            grpc_client=self._grpc_client,
        )

        logger.info(
            "[GRPC LISTENER] bridge.run() starting: session=%s thread=%s run=%s",
            self._session_id,
            thread_id,
            run_id,
        )

        active_streams: dict[str, asyncio.Queue] = getattr(
            self._bridge, "_active_streams", {}
        )
        run_queue = active_streams.get(thread_id)

        async def _run_once():
            async for event in self._bridge.run(input_data):
                stream_queue = active_streams.get(thread_id)
                if stream_queue is not None:
                    try:
                        stream_queue.put_nowait(event)
                    except asyncio.QueueFull:
                        logger.warning(
                            "[GRPC LISTENER] SSE tap queue full, dropping event: thread=%s",
                            thread_id,
                        )
                await writer.consume(event)
                await ops_writer.consume(event)

        try:
            await _run_once()
        except PermissionError as exc:
            logger.warning(
                "[GRPC LISTENER] Credential auth failure, refreshing token and retrying: session=%s error=%s",
                self._session_id,
                exc,
            )
            try:
                from ambient_runner.platform.utils import refresh_bot_token

                await asyncio.get_running_loop().run_in_executor(
                    None, refresh_bot_token
                )
            except Exception as refresh_exc:
                logger.warning(
                    "[GRPC LISTENER] Token refresh failed: session=%s error=%s",
                    self._session_id,
                    refresh_exc,
                )
            try:
                writer = GRPCMessageWriter(
                    session_id=self._session_id,
                    run_id=run_id,
                    grpc_client=self._grpc_client,
                )
                ops_writer = OperationalEventWriter(
                    session_id=self._session_id,
                    grpc_client=self._grpc_client,
                )
                await _run_once()
            except Exception as retry_exc:
                logger.error(
                    "[GRPC LISTENER] bridge.run() failed after token refresh: session=%s error=%s",
                    self._session_id,
                    retry_exc,
                    exc_info=True,
                )
                _synthesize_run_error(thread_id, str(retry_exc), active_streams, writer)
        except Exception as exc:
            logger.error(
                "[GRPC LISTENER] bridge.run() failed: session=%s error=%s",
                self._session_id,
                exc,
                exc_info=True,
            )
            _synthesize_run_error(thread_id, str(exc), active_streams, writer)
        finally:
            if run_queue is not None and active_streams.get(thread_id) is run_queue:
                active_streams.pop(thread_id, None)
            logger.info(
                "[GRPC LISTENER] Turn complete: session=%s thread=%s",
                self._session_id,
                thread_id,
            )


class GRPCMessageWriter:
    """Pushes assistant text to the session messages API in real-time.

    Accumulates TEXT_MESSAGE_CONTENT deltas and pushes the complete text
    on TEXT_MESSAGE_END. This ensures each assistant turn gets a server-
    assigned seq number at the correct chronological position — after any
    preceding tool events and before any subsequent ones.
    """

    def __init__(
        self,
        session_id: str,
        run_id: str,
        grpc_client: Optional["AmbientGRPCClient"],
    ) -> None:
        self._session_id = session_id
        self._run_id = run_id
        self._grpc_client = grpc_client
        self._text_buffer: str = ""

    async def consume(self, event: BaseEvent) -> None:
        raw_type = getattr(event, "type", None)
        if raw_type is None:
            return
        event_type_str = raw_type.value if hasattr(raw_type, "value") else str(raw_type)

        if event_type_str == "TEXT_MESSAGE_START":
            self._text_buffer = ""

        elif event_type_str == "TEXT_MESSAGE_CONTENT":
            delta = getattr(event, "delta", None) or ""
            self._text_buffer += delta

        elif event_type_str == "TEXT_MESSAGE_END":
            text = self._text_buffer.strip()
            self._text_buffer = ""
            if text:
                await self._push_assistant(text)

    async def _push_assistant(self, text: str) -> None:
        if self._grpc_client is None:
            logger.warning(
                "[GRPC WRITER] No gRPC client — cannot push assistant text: session=%s",
                self._session_id,
            )
            return

        logger.info(
            "[GRPC WRITER] PushAssistant: session=%s run=%s text_len=%d",
            self._session_id,
            self._run_id,
            len(text),
        )

        client = self._grpc_client
        session_id = self._session_id

        def _do_push() -> None:
            client.session_messages.push(
                session_id,
                event_type="assistant",
                payload=text,
            )

        try:
            await asyncio.get_running_loop().run_in_executor(None, _do_push)
        except Exception as exc:
            logger.warning(
                "[GRPC WRITER] Push failed: session=%s error=%s",
                self._session_id,
                exc,
            )

    async def push_error(self, error_message: str) -> None:
        """Push any buffered text plus the error as an assistant message."""
        text = self._text_buffer.strip()
        self._text_buffer = ""
        if text:
            await self._push_assistant(text)

        if not self._grpc_client:
            return

        client = self._grpc_client
        session_id = self._session_id

        def _do_push() -> None:
            client.session_messages.push(
                session_id,
                event_type="error",
                payload=error_message,
            )

        try:
            await asyncio.get_running_loop().run_in_executor(None, _do_push)
        except Exception as exc:
            logger.warning(
                "[GRPC WRITER] Error push failed: session=%s error=%s",
                self._session_id,
                exc,
            )
