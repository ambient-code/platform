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
import logging
import os
from contextlib import suppress
from typing import Any, AsyncIterator, Optional

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
    """

    def __init__(
        self,
        thread_id: str,
        options: Any,
        api_key: str,
    ):
        self.thread_id = thread_id
        self._options = options
        self._api_key = api_key

        # Inbound: (prompt, session_id, output_queue) | _SHUTDOWN
        self._input_queue: asyncio.Queue = asyncio.Queue()
        self._task: Optional[asyncio.Task] = None
        self._client: Optional[Any] = None  # ClaudeSDKClient once connected

        # Session ID returned by the CLI (for resume on restart)
        self.session_id: Optional[str] = None

    # ── lifecycle ──

    async def start(self) -> None:
        """Spawn the background task that owns the SDK client."""
        if self._task is not None:
            return
        self._task = asyncio.create_task(
            self._run(), name=f"session-worker-{self.thread_id}"
        )
        logger.info(f"[SessionWorker] Started worker for thread={self.thread_id}")

    async def _run(self) -> None:
        """Main loop — runs entirely inside one stable async context."""
        from claude_agent_sdk import ClaudeSDKClient, SystemMessage

        os.environ["ANTHROPIC_API_KEY"] = self._api_key

        client = ClaudeSDKClient(options=self._options)
        self._client = client

        try:
            await client.connect()
            logger.info(f"[SessionWorker] Connected for thread={self.thread_id}")

            while True:
                item = await self._input_queue.get()

                if item is _SHUTDOWN:
                    logger.info(
                        f"[SessionWorker] Shutdown signal for thread={self.thread_id}"
                    )
                    break

                prompt, session_id, output_queue = item

                try:
                    await client.query(prompt, session_id=session_id)

                    async for msg in client.receive_response():
                        # Capture session_id from init message (for resume)
                        if isinstance(msg, SystemMessage):
                            data = getattr(msg, "data", {}) or {}
                            if getattr(msg, "subtype", "") == "init":
                                sid = data.get("session_id")
                                if sid:
                                    self.session_id = sid

                        await output_queue.put(msg)

                except Exception as exc:
                    logger.error(
                        f"[SessionWorker] Error during query for "
                        f"thread={self.thread_id}: {exc}"
                    )
                    await output_queue.put(WorkerError(exc))
                finally:
                    # Sentinel: this turn is done (success or error).
                    await output_queue.put(None)

        except Exception as exc:
            logger.error(f"[SessionWorker] Fatal error for thread={self.thread_id}: {exc}")
        finally:
            self._client = None
            # Graceful shutdown: close stdin so the CLI saves the session
            # to .claude/ before being terminated.  This enables --resume
            # on pod restart.
            await self._graceful_disconnect(client)
            logger.info(f"[SessionWorker] Disconnected for thread={self.thread_id}")

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
                        await asyncio.wait_for(
                            asyncio.shield(p.wait()), timeout=5.0
                        )
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


class SessionManager:
    """Creates, caches, and tears down :class:`SessionWorker` instances.

    One worker per ``thread_id``.  A per-thread ``asyncio.Lock`` prevents
    concurrent requests to the *same* thread from overlapping (which would
    mix messages on the single underlying SDK client).

    Tracks session IDs returned by the CLI so that workers can be recreated
    with ``--resume`` after a pod restart.
    """

    def __init__(self) -> None:
        self._workers: dict[str, SessionWorker] = {}
        self._locks: dict[str, asyncio.Lock] = {}
        self._session_ids: dict[str, str] = {}  # thread_id -> CLI session_id

    async def get_or_create(
        self,
        thread_id: str,
        options: Any,
        api_key: str,
    ) -> SessionWorker:
        """Return the worker for *thread_id*, creating one if needed."""
        if thread_id not in self._workers:
            worker = SessionWorker(thread_id, options, api_key)
            await worker.start()
            self._workers[thread_id] = worker
            self._locks[thread_id] = asyncio.Lock()
            logger.debug(f"[SessionManager] Created worker for thread={thread_id}")
        return self._workers[thread_id]

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

    async def destroy(self, thread_id: str) -> None:
        """Stop and remove the worker for *thread_id*.

        Captures the session ID before destruction so the session can be
        resumed later (e.g. after pod restart).
        """
        worker = self._workers.pop(thread_id, None)
        if worker is not None:
            if worker.session_id:
                self._session_ids[thread_id] = worker.session_id
            await worker.stop()
        self._locks.pop(thread_id, None)
        logger.debug(f"[SessionManager] Destroyed worker for thread={thread_id}")

    async def shutdown(self) -> None:
        """Stop all workers gracefully.  Call on server shutdown."""
        thread_ids = list(self._workers.keys())
        for tid in thread_ids:
            await self.destroy(tid)
        logger.info("[SessionManager] All workers shut down")
