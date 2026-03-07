"""Subprocess management for the Gemini CLI bridge.

Key difference from the Claude session layer: Gemini CLI is invoked
**once per turn** (not a long-lived process).  Each ``query()`` call
spawns ``gemini -p <prompt> --output-format stream-json``, reads its
stdout as NDJSON, and tears down the process when the stream ends.
"""

import asyncio
import json
import logging
import os
import signal
import time
from collections import deque
from pathlib import Path
from typing import AsyncIterator, Optional

logger = logging.getLogger(__name__)

# Configurable via environment
GEMINI_CLI_TIMEOUT_SEC = int(os.getenv("GEMINI_CLI_TIMEOUT_SEC", "300"))
SHUTDOWN_TIMEOUT_SEC = int(os.getenv("SHUTDOWN_TIMEOUT_SEC", "10"))
WORKER_TTL_SEC = int(os.getenv("WORKER_TTL_SEC", "3600"))

# Maximum stderr lines kept in ring buffer per worker
_MAX_STDERR_LINES = 100

# Env vars that should NOT be passed to the Gemini CLI subprocess.
# These are runner-internal secrets that the CLI doesn't need.
_GEMINI_ENV_BLOCKLIST = frozenset(
    {
        "ANTHROPIC_API_KEY",
        "BOT_TOKEN",
        "LANGFUSE_SECRET_KEY",
        "LANGFUSE_PUBLIC_KEY",
        "LANGFUSE_HOST",
        "AWS_ACCESS_KEY_ID",
        "AWS_SECRET_ACCESS_KEY",
        "S3_ENDPOINT",
        "S3_BUCKET",
        "GOOGLE_OAUTH_CLIENT_ID",
        "GOOGLE_OAUTH_CLIENT_SECRET",
    }
)


class GeminiSessionWorker:
    """Spawns the Gemini CLI for a single turn and yields NDJSON lines."""

    def __init__(
        self,
        *,
        model: str,
        api_key: str = "",
        use_vertex: bool = False,
        cwd: str = "",
        include_directories: list[str] | None = None,
    ) -> None:
        self._model = model
        self._api_key = api_key
        self._use_vertex = use_vertex
        self._cwd = cwd or os.getenv("WORKSPACE_PATH", "/workspace")
        self._include_directories = include_directories or []
        self._process: Optional[asyncio.subprocess.Process] = None
        self._stderr_lines: deque[str] = deque(maxlen=_MAX_STDERR_LINES)
        self._stderr_task: Optional[asyncio.Task] = None

    @property
    def stderr_lines(self) -> list[str]:
        """Return the buffered stderr lines."""
        return list(self._stderr_lines)

    async def _stream_stderr(self) -> None:
        """Read stderr line by line into a capped ring buffer."""
        if self._process is None or self._process.stderr is None:
            return
        try:
            async for raw_line in self._process.stderr:
                line = raw_line.decode().rstrip()
                if line:
                    self._stderr_lines.append(line)
                    logger.debug("[Gemini stderr] %s", line)
        except asyncio.CancelledError:
            raise
        except Exception:
            logger.debug("stderr stream ended with error", exc_info=True)

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
        for d in self._include_directories:
            if os.path.isdir(d):
                cmd.extend(["--include-directories", d])

        env = {k: v for k, v in os.environ.items() if k not in _GEMINI_ENV_BLOCKLIST}
        if self._use_vertex:
            # Vertex AI mode: Gemini CLI requires GOOGLE_GENAI_USE_VERTEXAI=true
            # to use Vertex instead of AI Studio. API keys must be unset (they
            # take precedence and bypass Vertex).
            # See: https://geminicli.com/docs/get-started/authentication/
            env["GOOGLE_GENAI_USE_VERTEXAI"] = "true"
            # Map platform Vertex env vars to Gemini CLI's expected names if not
            # already set. The platform uses ANTHROPIC_VERTEX_PROJECT_ID and
            # CLOUD_ML_REGION for Claude; Gemini CLI needs GOOGLE_CLOUD_PROJECT
            # and GOOGLE_CLOUD_LOCATION.
            if not env.get("GOOGLE_CLOUD_PROJECT"):
                project = env.get("ANTHROPIC_VERTEX_PROJECT_ID", "")
                if project:
                    env["GOOGLE_CLOUD_PROJECT"] = project
            if not env.get("GOOGLE_CLOUD_LOCATION"):
                location = env.get("CLOUD_ML_REGION", "")
                if location:
                    env["GOOGLE_CLOUD_LOCATION"] = location
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

        # Start concurrent stderr streaming
        self._stderr_task = asyncio.create_task(self._stream_stderr())

        try:
            if self._process.stdout is None:
                raise RuntimeError(
                    "Gemini CLI process has no stdout - cannot read NDJSON stream"
                )

            async def _read_lines() -> AsyncIterator[str]:
                async for raw_line in self._process.stdout:
                    stripped = raw_line.decode().strip()
                    if stripped:
                        yield stripped

            deadline = asyncio.get_event_loop().time() + GEMINI_CLI_TIMEOUT_SEC
            async for line in _read_lines():
                yield line
                if asyncio.get_event_loop().time() > deadline:
                    logger.warning(
                        "Gemini CLI timed out after %d seconds, killing process",
                        GEMINI_CLI_TIMEOUT_SEC,
                    )
                    await self._kill_process()
                    raise TimeoutError(
                        f"Gemini CLI timed out after {GEMINI_CLI_TIMEOUT_SEC}s"
                    )

            # Wait for process to finish
            await self._process.wait()

            # If the process failed, raise so adapter emits RUN_ERROR
            if self._process.returncode and self._process.returncode != 0:
                stderr_tail = " | ".join(list(self._stderr_lines)[-5:])
                logger.warning(
                    "Gemini CLI exited with code %d; recent stderr: %s",
                    self._process.returncode,
                    stderr_tail,
                )
                raise RuntimeError(
                    f"Gemini CLI exited with code {self._process.returncode}"
                    + (f": {stderr_tail}" if stderr_tail else "")
                )
        finally:
            # Ensure stderr task is cleaned up
            if self._stderr_task and not self._stderr_task.done():
                self._stderr_task.cancel()
                try:
                    await self._stderr_task
                except asyncio.CancelledError:
                    pass
            self._stderr_task = None
            self._process = None

    async def _kill_process(self) -> None:
        """Send SIGTERM, wait, then SIGKILL if needed."""
        if self._process is None or self._process.returncode is not None:
            return
        try:
            self._process.terminate()
            logger.debug("Sent SIGTERM to Gemini CLI process")
        except ProcessLookupError:
            return
        try:
            await asyncio.wait_for(self._process.wait(), timeout=SHUTDOWN_TIMEOUT_SEC)
        except asyncio.TimeoutError:
            logger.warning(
                "Gemini CLI did not exit after %ds SIGTERM, sending SIGKILL",
                SHUTDOWN_TIMEOUT_SEC,
            )
            try:
                self._process.kill()
                await self._process.wait()
            except ProcessLookupError:
                pass

    async def interrupt(self) -> None:
        """Send SIGINT to the running Gemini CLI process."""
        if self._process and self._process.returncode is None:
            try:
                self._process.send_signal(signal.SIGINT)
                logger.info("Sent SIGINT to Gemini CLI process")
            except ProcessLookupError:
                pass

    async def stop(self) -> None:
        """Terminate the running Gemini CLI process with graceful shutdown."""
        await self._kill_process()
        # Ensure stderr task is cleaned up
        if self._stderr_task and not self._stderr_task.done():
            self._stderr_task.cancel()
            try:
                await self._stderr_task
            except asyncio.CancelledError:
                pass
            self._stderr_task = None


class GeminiSessionManager:
    """Manages Gemini session workers and tracks session IDs for --resume.

    Unlike the Claude ``SessionManager`` (which keeps long-lived SDK
    clients), this manager creates a fresh ``GeminiSessionWorker`` for
    each thread and remembers the ``session_id`` returned by the CLI's
    ``init`` event so subsequent turns can ``--resume``.
    """

    _EVICTION_INTERVAL = 60.0  # seconds between eviction scans
    _SESSION_IDS_FILE = "gemini_session_ids.json"

    def __init__(self, state_dir: str = "") -> None:
        self._workers: dict[str, GeminiSessionWorker] = {}
        self._session_ids: dict[str, str] = {}
        self._locks: dict[str, asyncio.Lock] = {}
        self._last_access: dict[str, float] = {}
        self._last_eviction: float = 0.0
        self._state_dir = state_dir
        # Restore session IDs from disk (persisted by state-sync for --resume across restarts)
        self._restore_session_ids()

    def _evict_stale(self) -> None:
        """Remove workers idle longer than WORKER_TTL_SEC (runs at most every 60s)."""
        now = time.monotonic()
        if now - self._last_eviction < self._EVICTION_INTERVAL:
            return
        self._last_eviction = now
        stale = [
            tid for tid, ts in self._last_access.items() if now - ts > WORKER_TTL_SEC
        ]
        for tid in stale:
            worker = self._workers.pop(tid, None)
            self._session_ids.pop(tid, None)
            self._locks.pop(tid, None)
            self._last_access.pop(tid, None)
            if worker:
                # Fire-and-forget stop -- best effort cleanup
                try:
                    loop = asyncio.get_running_loop()
                    loop.create_task(worker.stop())
                except RuntimeError:
                    pass
            logger.debug("Evicted stale worker for thread=%s", tid)

    def get_or_create_worker(
        self,
        thread_id: str,
        *,
        model: str,
        api_key: str = "",
        use_vertex: bool = False,
        cwd: str = "",
        include_directories: list[str] | None = None,
    ) -> GeminiSessionWorker:
        """Return a worker for *thread_id*, creating one if needed."""
        self._evict_stale()
        self._last_access[thread_id] = time.monotonic()

        if thread_id not in self._workers:
            self._workers[thread_id] = GeminiSessionWorker(
                model=model,
                api_key=api_key,
                use_vertex=use_vertex,
                cwd=cwd,
                include_directories=include_directories,
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
        """Record the session_id from an init event and persist to disk."""
        self._session_ids[thread_id] = session_id
        self._persist_session_ids()
        logger.debug("Recorded session_id=%s for thread=%s", session_id, thread_id)

    def _session_ids_path(self) -> Path | None:
        """Return the path to the session IDs file, or None if no state dir."""
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
                    "Restored %d Gemini session ID(s) from %s", len(restored), path
                )
        except (OSError, json.JSONDecodeError):
            logger.debug("Could not restore session IDs from %s", path, exc_info=True)

    async def interrupt(self, thread_id: str) -> None:
        """Interrupt the active worker for a thread."""
        worker = self._workers.get(thread_id)
        if worker:
            await worker.interrupt()
        else:
            logger.warning("No worker to interrupt for thread=%s", thread_id)

    def get_stderr_lines(self, thread_id: str) -> list[str]:
        """Return buffered stderr lines for a thread's worker."""
        worker = self._workers.get(thread_id)
        if worker:
            return worker.stderr_lines
        return []

    def get_all_stderr(self, max_per_worker: int = 10) -> list[str]:
        """Collect recent stderr lines from all active workers."""
        all_lines: list[str] = []
        for worker in self._workers.values():
            lines = worker.stderr_lines
            if lines:
                all_lines.extend(lines[-max_per_worker:])
        return all_lines

    async def shutdown(self) -> None:
        """Stop all active workers with an overall timeout."""

        async def _stop_all() -> None:
            tasks = [worker.stop() for worker in self._workers.values()]
            if tasks:
                await asyncio.gather(*tasks, return_exceptions=True)

        try:
            await asyncio.wait_for(_stop_all(), timeout=SHUTDOWN_TIMEOUT_SEC * 2)
        except asyncio.TimeoutError:
            logger.warning(
                "GeminiSessionManager: shutdown timed out after %ds, "
                "some workers may not have stopped cleanly",
                SHUTDOWN_TIMEOUT_SEC * 2,
            )

        self._workers.clear()
        self._last_access.clear()
        logger.info("GeminiSessionManager: all workers shut down")
