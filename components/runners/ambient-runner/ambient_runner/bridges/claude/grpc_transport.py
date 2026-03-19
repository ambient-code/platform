"""
gRPC transport for ClaudeBridge.

GRPCSessionListener — pod-lifetime WatchSessionMessages subscriber.
  Replaces _background_inbound_watcher in app.py.
  Calls bridge.run() directly for each inbound user message (no HTTP round-trip).
  Fans out each event to:
    (a) bridge._active_streams[thread_id] queue — feeds the /events SSE tap
    (b) GRPCMessageWriter — assembles and writes the durable DB record

GRPCMessageWriter — per-turn event consumer.
  Accumulates MESSAGES_SNAPSHOT content.
  Pushes one PushSessionMessage(event_type="assistant") on RUN_FINISHED / RUN_ERROR.
"""

import asyncio
import json
import logging
import uuid
from concurrent.futures import ThreadPoolExecutor
from typing import TYPE_CHECKING, Any, Optional

from ag_ui.core import BaseEvent

if TYPE_CHECKING:
    from ambient_runner._grpc_client import AmbientGRPCClient
    from ambient_runner.bridge import PlatformBridge

logger = logging.getLogger(__name__)

_BACKOFF_INITIAL = 1.0
_BACKOFF_MAX = 30.0


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
        except Exception as exc:
            logger.info(
                "[GRPC LISTENER] Watch stream ended: session=%s reason=%s",
                self._session_id,
                exc,
            )

    async def _listen_loop(self) -> None:
        last_seq = 0
        backoff = _BACKOFF_INITIAL

        while True:
            msg_queue: asyncio.Queue = asyncio.Queue()
            stop_event = asyncio.Event()
            loop = asyncio.get_event_loop()
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
                "[GRPC LISTENER] Failed to build run agent input: seq=%d error=%s preview=%r",
                msg.seq,
                exc,
                (msg.payload or "")[:120],
            )
            return

        thread_id = input_data.thread_id or self._session_id
        run_id = str(input_data.run_id) if input_data.run_id else str(uuid.uuid4())

        writer = GRPCMessageWriter(
            session_id=self._session_id,
            run_id=run_id,
            grpc_client=self._grpc_client,
        )

        logger.info(
            "[GRPC LISTENER] bridge.run() starting: session=%s thread=%s run=%s",
            self._session_id,
            thread_id,
            run_id,
        )

        try:
            async for event in self._bridge.run(input_data):
                active_streams: dict = getattr(self._bridge, "_active_streams", {})
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
        except Exception as exc:
            logger.error(
                "[GRPC LISTENER] bridge.run() failed: session=%s error=%s",
                self._session_id,
                exc,
                exc_info=True,
            )
        finally:
            active_streams = getattr(self._bridge, "_active_streams", {})
            active_streams.pop(thread_id, None)
            logger.info(
                "[GRPC LISTENER] Turn complete: session=%s thread=%s",
                self._session_id,
                thread_id,
            )


class GRPCMessageWriter:
    """Per-turn event consumer. Writes one PushSessionMessage on turn end.

    Accumulates messages from MESSAGES_SNAPSHOT events (storing only the
    latest snapshot — each MESSAGES_SNAPSHOT is a complete replacement).
    On RUN_FINISHED or RUN_ERROR, pushes the assembled payload as a single
    durable DB record with event_type="assistant".
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
        self._accumulated_messages: list = []

    async def consume(self, event: BaseEvent) -> None:
        """Process one event from bridge.run(). Called by the listener fan-out loop."""
        raw_type = getattr(event, "type", None)
        if raw_type is None:
            return
        event_type_str = raw_type.value if hasattr(raw_type, "value") else str(raw_type)

        if event_type_str == "MESSAGES_SNAPSHOT":
            messages = getattr(event, "messages", None) or []
            self._accumulated_messages = [
                m.model_dump() if hasattr(m, "model_dump") else m for m in messages
            ]
            logger.debug(
                "[GRPC WRITER] MESSAGES_SNAPSHOT accumulated: session=%s count=%d",
                self._session_id,
                len(self._accumulated_messages),
            )

        elif event_type_str == "RUN_FINISHED":
            await self._write_message(status="completed")

        elif event_type_str == "RUN_ERROR":
            await self._write_message(status="error")

    async def _write_message(self, status: str) -> None:
        if self._grpc_client is None:
            logger.warning(
                "[GRPC WRITER] No gRPC client — cannot push assembled message: session=%s",
                self._session_id,
            )
            return

        payload = json.dumps(
            {
                "run_id": self._run_id,
                "status": status,
                "messages": self._accumulated_messages,
            }
        )

        logger.info(
            "[GRPC WRITER] PushSessionMessage: session=%s run=%s status=%s messages=%d payload_len=%d",
            self._session_id,
            self._run_id,
            status,
            len(self._accumulated_messages),
            len(payload),
        )

        self._grpc_client.session_messages.push(
            self._session_id,
            event_type="assistant",
            payload=payload,
        )
