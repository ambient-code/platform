"""Subprocess management for the Gemini CLI bridge.

Key difference from the Claude session layer: Gemini CLI is invoked
**once per turn** (not a long-lived process).  Each ``query()`` call
spawns ``gemini -p <prompt> --output-format stream-json``, reads its
stdout as NDJSON, and tears down the process when the stream ends.
"""

import asyncio
import logging
import os
import signal
from typing import AsyncIterator, Optional

logger = logging.getLogger(__name__)


class GeminiSessionWorker:
    """Spawns the Gemini CLI for a single turn and yields NDJSON lines."""

    def __init__(
        self,
        *,
        model: str,
        api_key: str = "",
        use_vertex: bool = False,
        cwd: str = "",
    ) -> None:
        self._model = model
        self._api_key = api_key
        self._use_vertex = use_vertex
        self._cwd = cwd or os.getenv("WORKSPACE_PATH", "/workspace")
        self._process: Optional[asyncio.subprocess.Process] = None

    async def query(
        self,
        prompt: str,
        session_id: Optional[str] = None,
    ) -> AsyncIterator[str]:
        """Spawn the Gemini CLI and yield NDJSON lines from stdout.

        Args:
            prompt: User prompt to send.
            session_id: Optional session ID from a previous init event
                (passed via ``--resume`` to continue the conversation).

        Yields:
            Raw NDJSON lines (stripped).
        """
        cmd = [
            "gemini",
            "-p",
            prompt,
            "--output-format",
            "stream-json",
            "--yolo",
            "--model",
            self._model,
        ]
        if session_id:
            cmd.extend(["--resume", session_id])

        env = dict(os.environ)
        if self._use_vertex:
            # Vertex AI mode: ensure API keys are NOT set (they take precedence
            # and bypass Vertex). GOOGLE_CLOUD_PROJECT, GOOGLE_CLOUD_LOCATION,
            # and GOOGLE_APPLICATION_CREDENTIALS should already be in os.environ.
            env.pop("GEMINI_API_KEY", None)
            env.pop("GOOGLE_API_KEY", None)
        elif self._api_key:
            # API key mode: Gemini CLI expects GEMINI_API_KEY
            # See: https://github.com/google-gemini/gemini-cli/issues/7557
            env["GEMINI_API_KEY"] = self._api_key
            env["GOOGLE_API_KEY"] = self._api_key

        logger.debug("Spawning Gemini CLI: %s (cwd=%s)", cmd, self._cwd)

        self._process = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
            cwd=self._cwd,
            env=env,
        )

        try:
            assert self._process.stdout is not None  # noqa: S101
            async for raw_line in self._process.stdout:
                stripped = raw_line.decode().strip()
                if stripped:
                    yield stripped

            # Wait for process to finish
            await self._process.wait()

            # Log stderr if the process failed
            if self._process.returncode and self._process.returncode != 0:
                stderr_data = b""
                if self._process.stderr:
                    stderr_data = await self._process.stderr.read()
                logger.warning(
                    "Gemini CLI exited with code %d: %s",
                    self._process.returncode,
                    stderr_data.decode().strip()[:500],
                )
        finally:
            self._process = None

    async def interrupt(self) -> None:
        """Send SIGINT to the running Gemini CLI process."""
        if self._process and self._process.returncode is None:
            try:
                self._process.send_signal(signal.SIGINT)
                logger.info("Sent SIGINT to Gemini CLI process")
            except ProcessLookupError:
                pass

    async def stop(self) -> None:
        """Terminate the running Gemini CLI process."""
        if self._process and self._process.returncode is None:
            try:
                self._process.terminate()
                logger.info("Terminated Gemini CLI process")
            except ProcessLookupError:
                pass


class GeminiSessionManager:
    """Manages Gemini session workers and tracks session IDs for --resume.

    Unlike the Claude ``SessionManager`` (which keeps long-lived SDK
    clients), this manager creates a fresh ``GeminiSessionWorker`` for
    each thread and remembers the ``session_id`` returned by the CLI's
    ``init`` event so subsequent turns can ``--resume``.
    """

    def __init__(self) -> None:
        self._workers: dict[str, GeminiSessionWorker] = {}
        self._session_ids: dict[str, str] = {}
        self._locks: dict[str, asyncio.Lock] = {}

    def get_or_create_worker(
        self,
        thread_id: str,
        *,
        model: str,
        api_key: str = "",
        use_vertex: bool = False,
        cwd: str = "",
    ) -> GeminiSessionWorker:
        """Return a worker for *thread_id*, creating one if needed."""
        if thread_id not in self._workers:
            self._workers[thread_id] = GeminiSessionWorker(
                model=model, api_key=api_key, use_vertex=use_vertex, cwd=cwd
            )
            logger.debug("Created GeminiSessionWorker for thread=%s", thread_id)
        return self._workers[thread_id]

    def get_lock(self, thread_id: str) -> asyncio.Lock:
        """Per-thread serialisation lock."""
        if thread_id not in self._locks:
            self._locks[thread_id] = asyncio.Lock()
        return self._locks[thread_id]

    def get_session_id(self, thread_id: str) -> Optional[str]:
        """Return the last known session_id for a thread."""
        return self._session_ids.get(thread_id)

    def set_session_id(self, thread_id: str, session_id: str) -> None:
        """Record the session_id from an init event."""
        self._session_ids[thread_id] = session_id
        logger.debug("Recorded session_id=%s for thread=%s", session_id, thread_id)

    async def interrupt(self, thread_id: str) -> None:
        """Interrupt the active worker for a thread."""
        worker = self._workers.get(thread_id)
        if worker:
            await worker.interrupt()
        else:
            logger.warning("No worker to interrupt for thread=%s", thread_id)

    async def shutdown(self) -> None:
        """Stop all active workers."""
        for tid, worker in list(self._workers.items()):
            await worker.stop()
        self._workers.clear()
        logger.info("GeminiSessionManager: all workers shut down")
