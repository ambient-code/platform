"""
Platform-layer session workers for the Claude Agent SDK.

Each ``SessionWorker`` owns a single ``ClaudeSDKClient`` running inside its
own ``asyncio.Task``.  HTTP request handlers never touch the SDK's internal
anyio task group — they communicate through plain ``asyncio.Queue`` objects.

This side-steps the anyio task-group context-mismatch bug that prevents
persistent ``ClaudeSDKClient`` instances from working inside FastAPI/SSE
handlers (see SDK issues #454 and #378).

Graceful shutdown closes stdin and waits for the CLI to persist the session
to ``.claude/`` so that ``--resume`` works on pod restart.

Usage::

    manager = SessionManager()
    worker  = await manager.get_or_create("thread-1", options, api_key)

    # From a request handler (any async context):
    async for msg in worker.query("Hello", session_id="thread-1"):
        ...

    # On server shutdown:
    await manager.shutdown()
"""

import asyncio
import json
import logging
import os
from contextlib import suppress
from pathlib import Path
from typing import Any, AsyncIterator, Callable, Optional

logger = logging.getLogger(__name__)

# Sentinel that tells the worker loop to shut down.
_SHUTDOWN = object()


class WorkerError:
    """Wrapper for exceptions forwarded through the output queue.

    Using a dedicated wrapper type avoids the fragile
    ``isinstance(item, Exception)`` check — callers match on the wrapper
    class instead, which is unambiguous even if the underlying exception
    happens to be a subclass of something else in the queue.
    """

    __slots__ = ("exception",)

    def __init__(self, exception: Exception) -> None:
        self.exception = exception


class SessionWorker:
    """Owns one ``ClaudeSDKClient`` in a long-lived background task.

    The task is created by :meth:`start` and runs until :meth:`stop` is
    called (or the client errors out).  Request handlers call :meth:`query`
    which bridges to the background task via a pair of asyncio queues.

    Uses a persistent ``receive_messages()`` reader that routes SDK messages
    to the active run's output queue during a turn, and to a between-run
    queue otherwise.  This ensures late SDK events (background task
    completions, hook notifications) are delivered in real-time.
    """

    def __init__(
        self,
        thread_id: str,
        options: Any,
        api_key: str,
        on_session_id: Optional[Callable[[str, str], None]] = None,
    ):
        self.thread_id = thread_id
        self._options = options
        self._api_key = api_key
        self._on_session_id = on_session_id

        # Inbound: (prompt, session_id, output_queue) | _SHUTDOWN
        self._input_queue: asyncio.Queue = asyncio.Queue()
        self._task: Optional[asyncio.Task] = None
        self._client: Optional[Any] = None  # ClaudeSDKClient once connected

        # Persistent reader routing state
        self._active_output_queue: Optional[asyncio.Queue] = None
        self._turn_done: asyncio.Event = asyncio.Event()
        self._between_run_queue: asyncio.Queue = asyncio.Queue(maxsize=1000)
        self._reader_task: Optional[asyncio.Task] = None

        # Session ID returned by the CLI (for resume on restart)
        self.session_id: Optional[str] = None

    # ── lifecycle ──

    @property
    def is_alive(self) -> bool:
        """True if the background task is still running."""
        return self._task is not None and not self._task.done()

    async def start(self) -> None:
        """Spawn the background task that owns the SDK client."""
        if self._task is not None:
            return
        self._task = asyncio.create_task(
            self._run(), name=f"session-worker-{self.thread_id}"
        )
        logger.info(f"[SessionWorker] Started worker for thread={self.thread_id}")

    async def _run(self) -> None:
        """Main loop — runs entirely inside one stable async context.

        Connects the SDK client, starts a persistent reader task that
        routes messages, and loops on the input queue for new queries.
        """
        from claude_agent_sdk import ClaudeSDKClient

        os.environ["ANTHROPIC_API_KEY"] = self._api_key

        from ambient_runner.bridges.claude.mock_client import (
            MOCK_API_KEY,
            MockClaudeSDKClient,
        )

        if self._api_key == MOCK_API_KEY:
            logger.info("[SessionWorker] Using MockClaudeSDKClient (replay mode)")
            client: Any = MockClaudeSDKClient(options=self._options)
        else:
            client = ClaudeSDKClient(options=self._options)
        self._client = client

        try:
            await client.connect()
            logger.info(f"[SessionWorker] Connected for thread={self.thread_id}")

            # Start persistent reader
            self._reader_task = asyncio.create_task(
                self._read_messages_forever(client),
                name=f"reader-{self.thread_id}",
            )

            while True:
                # Check if reader died between turns
                if self._reader_task.done():
                    exc = self._reader_task.exception()
                    logger.error(
                        "[SessionWorker] Reader task died for thread=%s: %s",
                        self.thread_id,
                        exc,
                    )
                    break

                item = await self._input_queue.get()

                if item is _SHUTDOWN:
                    logger.info(
                        f"[SessionWorker] Shutdown signal for thread={self.thread_id}"
                    )
                    break

                prompt, session_id, output_queue = item

                try:
                    self._active_output_queue = output_queue
                    self._turn_done.clear()

                    await client.query(prompt, session_id=session_id)

                    # Wait for reader to signal ResultMessage received,
                    # but also bail if the reader task dies mid-turn.
                    reader_done = asyncio.ensure_future(
                        asyncio.shield(self._reader_task)
                    ) if self._reader_task else None
                    turn_wait = asyncio.ensure_future(self._turn_done.wait())

                    waiters = [turn_wait]
                    if reader_done:
                        waiters.append(reader_done)

                    done, _ = await asyncio.wait(
                        waiters, return_when=asyncio.FIRST_COMPLETED
                    )

                    # Cancel the loser
                    for fut in waiters:
                        if fut not in done:
                            fut.cancel()

                    if reader_done and reader_done in done:
                        logger.error(
                            "[SessionWorker] Reader died mid-turn for "
                            "thread=%s",
                            self.thread_id,
                        )
                        await output_queue.put(WorkerError(
                            RuntimeError("SDK message reader died")
                        ))
                        break

                except Exception as exc:
                    logger.error(
                        "[SessionWorker] Error during query for "
                        "thread=%s, stopping worker: %s",
                        self.thread_id,
                        exc,
                    )
                    await output_queue.put(WorkerError(exc))
                    break
                finally:
                    # Set None BEFORE sentinel so the reader routes any
                    # trailing messages to the between-run queue.
                    self._active_output_queue = None
                    await output_queue.put(None)

        except Exception as exc:
            logger.error(
                f"[SessionWorker] Fatal error for thread={self.thread_id}: {exc}"
            )
        finally:
            self._client = None
            # Cancel reader task
            if self._reader_task and not self._reader_task.done():
                self._reader_task.cancel()
                with suppress(asyncio.CancelledError):
                    await self._reader_task
            self._reader_task = None
            # Drain between-run queue to make room for shutdown sentinel
            while not self._between_run_queue.empty():
                try:
                    self._between_run_queue.get_nowait()
                except asyncio.QueueEmpty:
                    break
            self._between_run_queue.put_nowait(_SHUTDOWN)
            # Graceful shutdown: close stdin so the CLI saves the session
            # to .claude/ before being terminated.  This enables --resume
            # on pod restart.
            await self._graceful_disconnect(client)
            logger.info(f"[SessionWorker] Disconnected for thread={self.thread_id}")

    async def _read_messages_forever(self, client: Any) -> None:
        """Persistent reader — routes messages to active run or between-run queue."""
        from claude_agent_sdk import SystemMessage, ResultMessage

        try:
            async for msg in client.receive_messages():
                msg_type = type(msg).__name__
                subtype = getattr(msg, "subtype", "")
                route = "run" if self._active_output_queue is not None else "between-run"
                logger.debug(
                    "[Reader] %s (subtype=%s) → %s queue",
                    msg_type, subtype, route,
                )

                # Capture session_id from init message (for resume)
                if isinstance(msg, SystemMessage):
                    data = getattr(msg, "data", {}) or {}
                    if subtype == "init":
                        sid = data.get("session_id")
                        if sid:
                            self.session_id = sid
                            if self._on_session_id:
                                self._on_session_id(self.thread_id, sid)

                if self._active_output_queue is not None:
                    await self._active_output_queue.put(msg)
                    if isinstance(msg, ResultMessage):
                        logger.debug("[Reader] ResultMessage → turn done")
                        self._turn_done.set()
                else:
                    # Between runs — non-blocking put, drop if full
                    try:
                        self._between_run_queue.put_nowait(msg)
                    except asyncio.QueueFull:
                        logger.warning("Between-run queue full, dropping event")
        except Exception as exc:
            logger.error(
                "[SessionWorker] Reader stream error for thread=%s: %s",
                self.thread_id,
                exc,
            )
            # Unblock the main loop if it's waiting on a turn
            self._turn_done.set()

    async def between_run_queue_get(self) -> Any:
        """Get the next message from the between-run queue. Returns None on shutdown."""
        msg = await self._between_run_queue.get()
        if msg is _SHUTDOWN:
            return None
        return msg

    async def between_run_events(self) -> AsyncIterator[Any]:
        """Yield SDK messages arriving outside user-initiated runs."""
        while True:
            msg = await self._between_run_queue.get()
            if msg is _SHUTDOWN:
                return
            yield msg

    async def _graceful_disconnect(self, client: Any) -> None:
        """Close stdin, wait for CLI to save, then disconnect."""
        try:
            t = getattr(client, "_transport", None)
            if t:
                s = getattr(t, "_stdin_stream", None)
                if s:
                    with suppress(Exception):
                        await s.aclose()
                    t._stdin_stream = None
                p = getattr(t, "_process", None)
                if p and p.returncode is None:
                    try:
                        await asyncio.wait_for(asyncio.shield(p.wait()), timeout=5.0)
                    except asyncio.TimeoutError:
                        pass
        except Exception:
            pass
        finally:
            try:
                await client.disconnect()
            except Exception:
                pass

    async def stop(self) -> None:
        """Signal the worker to shut down and wait for it to finish."""
        if self._task is None:
            return
        await self._input_queue.put(_SHUTDOWN)
        try:
            await asyncio.wait_for(self._task, timeout=15.0)
        except asyncio.TimeoutError:
            logger.warning(
                f"[SessionWorker] Worker for thread={self.thread_id} "
                "did not stop in time, cancelling"
            )
            self._task.cancel()
            with suppress(asyncio.CancelledError):
                await self._task
        self._task = None

    # ── called from request handlers ──

    async def query(
        self, prompt: str, session_id: str = "default"
    ) -> AsyncIterator[Any]:
        """Send *prompt* to the worker and yield SDK ``Message`` objects.

        Safe to call from any async context (e.g. a FastAPI handler).
        """
        output_queue: asyncio.Queue = asyncio.Queue()
        await self._input_queue.put((prompt, session_id, output_queue))

        while True:
            item = await output_queue.get()
            if item is None:
                return
            if isinstance(item, WorkerError):
                raise item.exception
            yield item

    async def interrupt(self) -> None:
        """Forward an interrupt signal to the underlying SDK client."""
        if self._client is not None:
            try:
                await self._client.interrupt()
            except Exception as exc:
                logger.warning(f"[SessionWorker] Interrupt failed: {exc}")
        else:
            logger.warning("[SessionWorker] Interrupt requested but no active client")

    async def stop_task(self, task_id: str) -> None:
        """Stop a background task (subagent) by ID."""
        if self._client is not None:
            try:
                await self._client.stop_task(task_id)
            except Exception as exc:
                logger.warning(f"[SessionWorker] stop_task({task_id}) failed: {exc}")
                raise
        else:
            raise RuntimeError("No active client")


class SessionManager:
    """Creates, caches, and tears down :class:`SessionWorker` instances.

    One worker per ``thread_id``.  A per-thread ``asyncio.Lock`` prevents
    concurrent requests to the *same* thread from overlapping (which would
    mix messages on the single underlying SDK client).

    Tracks session IDs returned by the CLI so that workers can be recreated
    with ``--resume`` after a pod restart.  Session IDs are persisted to disk
    so they survive pod restarts.
    """

    _SESSION_IDS_FILE = "claude_session_ids.json"

    def __init__(self, state_dir: str = "") -> None:
        self._workers: dict[str, SessionWorker] = {}
        self._locks: dict[str, asyncio.Lock] = {}
        self._session_ids: dict[str, str] = {}  # thread_id -> CLI session_id
        self._state_dir = state_dir
        self._restore_session_ids()

    async def get_or_create(
        self,
        thread_id: str,
        options: Any,
        api_key: str,
    ) -> SessionWorker:
        """Return the worker for *thread_id*, creating one if needed.

        If an existing worker's background task has exited (e.g. after a
        fatal SDK error), it is destroyed and a fresh worker is created so
        that the session can recover.
        """
        if thread_id in self._workers:
            existing = self._workers[thread_id]
            if existing.is_alive:
                return existing
            logger.warning(
                "[SessionManager] Worker for thread=%s is dead, recreating",
                thread_id,
            )
            await self.destroy(thread_id)

        worker = SessionWorker(
            thread_id, options, api_key, on_session_id=self._on_session_id
        )
        await worker.start()
        self._workers[thread_id] = worker
        self._locks[thread_id] = asyncio.Lock()
        logger.debug(f"[SessionManager] Created worker for thread={thread_id}")
        return worker

    def get_existing(self, thread_id: str) -> Optional[SessionWorker]:
        """Return the worker for *thread_id* if it exists, else ``None``."""
        return self._workers.get(thread_id)

    def get_lock(self, thread_id: str) -> asyncio.Lock:
        """Return the per-thread serialisation lock."""
        if thread_id not in self._locks:
            self._locks[thread_id] = asyncio.Lock()
        return self._locks[thread_id]

    def get_session_id(self, thread_id: str) -> Optional[str]:
        """Return the CLI session ID for *thread_id*, if known."""
        worker = self._workers.get(thread_id)
        if worker and worker.session_id:
            return worker.session_id
        return self._session_ids.get(thread_id)

    def get_all_session_ids(self) -> dict[str, str]:
        """Return a snapshot of all known session IDs (live workers + cached)."""
        result = dict(self._session_ids)
        for tid, worker in self._workers.items():
            if worker.session_id:
                result[tid] = worker.session_id
        return result

    async def destroy(self, thread_id: str) -> None:
        """Stop and remove the worker for *thread_id*.

        Captures the session ID before destruction so the session can be
        resumed later (e.g. after pod restart).
        """
        worker = self._workers.pop(thread_id, None)
        if worker is not None:
            if worker.session_id:
                self._session_ids[thread_id] = worker.session_id
                self._persist_session_ids()
            await worker.stop()
        self._locks.pop(thread_id, None)
        logger.debug(f"[SessionManager] Destroyed worker for thread={thread_id}")

    async def shutdown(self) -> None:
        """Stop all workers gracefully.  Call on server shutdown."""
        thread_ids = list(self._workers.keys())
        for tid in thread_ids:
            await self.destroy(tid)
        logger.info("[SessionManager] All workers shut down")

    # ── session ID persistence ──

    def _on_session_id(self, thread_id: str, session_id: str) -> None:
        """Called by workers as soon as the CLI returns a session ID.

        Persists immediately so the mapping survives even if the current
        turn never completes (e.g. runner OOM during tool execution).
        """
        self._session_ids[thread_id] = session_id
        self._persist_session_ids()

    def _session_ids_path(self) -> Path | None:
        if not self._state_dir:
            return None
        return Path(self._state_dir) / self._SESSION_IDS_FILE

    def _persist_session_ids(self) -> None:
        """Save session IDs to disk for --resume across pod restarts."""
        path = self._session_ids_path()
        if not path or not self._session_ids:
            return
        try:
            path.parent.mkdir(parents=True, exist_ok=True)
            with open(path, "w") as f:
                json.dump(self._session_ids, f)
            logger.info(
                "Persisted %d session ID(s) to %s", len(self._session_ids), path
            )
        except OSError:
            logger.debug("Could not persist session IDs to %s", path, exc_info=True)

    def _restore_session_ids(self) -> None:
        """Restore session IDs from disk (written by a previous pod)."""
        path = self._session_ids_path()
        if not path or not path.exists():
            return
        try:
            with open(path) as f:
                restored = json.load(f)
            if isinstance(restored, dict):
                self._session_ids.update(restored)
                logger.info(
                    "Restored %d Claude session ID(s) from %s", len(restored), path
                )
        except (OSError, json.JSONDecodeError):
            logger.debug("Could not restore session IDs from %s", path, exc_info=True)
