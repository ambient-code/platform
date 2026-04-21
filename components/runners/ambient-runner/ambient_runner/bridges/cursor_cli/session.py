# components/runners/ambient-runner/ambient_runner/bridges/cursor_cli/session.py
"""Subprocess management for the Cursor CLI bridge.

The Cursor CLI `cursor-agent` binary is invoked once per turn.
Each query() call spawns `cursor-agent --print --force ... "<prompt>"`,
reads NDJSON from stdout, and tears down the process when done.
"""

import asyncio
import json
import logging
import os
import signal
import time
from collections import deque
from pathlib import Path
from typing import AsyncIterator

logger = logging.getLogger(__name__)

CURSOR_CLI_TIMEOUT_SEC = int(os.getenv("CURSOR_CLI_TIMEOUT_SEC", "300"))
SHUTDOWN_TIMEOUT_SEC = int(os.getenv("SHUTDOWN_TIMEOUT_SEC", "10"))
WORKER_TTL_SEC = int(os.getenv("WORKER_TTL_SEC", "3600"))

_MAX_STDERR_LINES = 100

_CURSOR_ENV_BLOCKLIST = frozenset(
    {
        "ANTHROPIC_API_KEY",
        "GEMINI_API_KEY",
        "GOOGLE_API_KEY",
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


class CursorSessionWorker:
    """Spawns the Cursor CLI for a single turn and yields NDJSON lines."""

    def __init__(
        self,
        *,
        model: str,
        api_key: str = "",
        cwd: str = "",
    ) -> None:
        self._model = model
        self._api_key = api_key
        self._cwd = cwd or os.getenv("WORKSPACE_PATH", "/workspace")
        self._process: asyncio.subprocess.Process | None = None
        self._stderr_lines: deque[str] = deque(maxlen=_MAX_STDERR_LINES)
        self._stderr_task: asyncio.Task | None = None

    @property
    def stderr_lines(self) -> list[str]:
        return list(self._stderr_lines)

    async def _stream_stderr(self) -> None:
        if self._process is None or self._process.stderr is None:
            return
        try:
            async for raw_line in self._process.stderr:
                line = raw_line.decode().rstrip()
                if line:
                    self._stderr_lines.append(line)
                    logger.debug("[Cursor stderr] %s", line)
        except asyncio.CancelledError:
            raise
        except Exception:
            logger.debug("stderr stream ended with error", exc_info=True)

    async def query(
        self,
        prompt: str,
        session_id: str | None = None,
    ) -> AsyncIterator[str]:
        """Spawn the Cursor CLI and yield NDJSON lines from stdout."""
        cmd = [
            "cursor-agent",
            "--print",
            "--force",
            "--approve-mcps",
            "--output-format",
            "stream-json",
            "--model",
            self._model,
        ]
        if session_id:
            cmd.extend(["--resume", session_id])
        cmd.extend(["--workspace", self._cwd])
        cmd.append(prompt)

        env = {k: v for k, v in os.environ.items() if k not in _CURSOR_ENV_BLOCKLIST}
        if self._api_key:
            env["CURSOR_API_KEY"] = self._api_key

        logger.debug("Spawning Cursor CLI: %s (cwd=%s)", cmd, self._cwd)

        self._process = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
            cwd=self._cwd,
            env=env,
            limit=10 * 1024 * 1024,
        )

        self._stderr_task = asyncio.create_task(self._stream_stderr())

        try:
            if self._process.stdout is None:
                raise RuntimeError(
                    "Cursor CLI process has no stdout - cannot read NDJSON stream"
                )

            async def _read_lines() -> AsyncIterator[str]:
                async for raw_line in self._process.stdout:
                    stripped = raw_line.decode().strip()
                    if stripped:
                        yield stripped

            loop = asyncio.get_running_loop()
            deadline = loop.time() + CURSOR_CLI_TIMEOUT_SEC
            async for line in _read_lines():
                yield line
                if loop.time() > deadline:
                    logger.warning(
                        "Cursor CLI timed out after %d seconds, killing process",
                        CURSOR_CLI_TIMEOUT_SEC,
                    )
                    await self._kill_process()
                    raise TimeoutError(
                        f"Cursor CLI timed out after {CURSOR_CLI_TIMEOUT_SEC}s"
                    )

            await self._process.wait()

            if self._process.returncode not in (None, 0):
                stderr_tail = " | ".join(list(self._stderr_lines)[-5:])
                logger.warning(
                    "Cursor CLI exited with code %d; recent stderr: %s",
                    self._process.returncode,
                    stderr_tail,
                )
                raise RuntimeError(
                    f"Cursor CLI exited with code {self._process.returncode}"
                    + (f": {stderr_tail}" if stderr_tail else "")
                )
        finally:
            if self._stderr_task and not self._stderr_task.done():
                self._stderr_task.cancel()
                try:
                    await self._stderr_task
                except asyncio.CancelledError:
                    pass
            self._stderr_task = None
            self._process = None

    async def _kill_process(self) -> None:
        if self._process is None or self._process.returncode is not None:
            return
        try:
            self._process.terminate()
            logger.debug("Sent SIGTERM to Cursor CLI process")
        except ProcessLookupError:
            return
        try:
            await asyncio.wait_for(self._process.wait(), timeout=SHUTDOWN_TIMEOUT_SEC)
        except asyncio.TimeoutError:
            logger.warning(
                "Cursor CLI did not exit after %ds SIGTERM, sending SIGKILL",
                SHUTDOWN_TIMEOUT_SEC,
            )
            try:
                self._process.kill()
                await self._process.wait()
            except ProcessLookupError:
                pass

    async def interrupt(self) -> None:
        if self._process and self._process.returncode is None:
            try:
                self._process.send_signal(signal.SIGINT)
                logger.info("Sent SIGINT to Cursor CLI process")
            except ProcessLookupError:
                pass

    async def stop(self) -> None:
        await self._kill_process()
        if self._stderr_task and not self._stderr_task.done():
            self._stderr_task.cancel()
            try:
                await self._stderr_task
            except asyncio.CancelledError:
                pass
            self._stderr_task = None


class CursorSessionManager:
    """Manages Cursor session workers and tracks session IDs for --resume."""

    _EVICTION_INTERVAL = 60.0
    _SESSION_IDS_FILE = "cursor_session_ids.json"

    def __init__(self, state_dir: str = "") -> None:
        self._workers: dict[str, CursorSessionWorker] = {}
        self._session_ids: dict[str, str] = {}
        self._locks: dict[str, asyncio.Lock] = {}
        self._last_access: dict[str, float] = {}
        self._last_eviction: float = 0.0
        self._ids_path: Path | None = (
            Path(state_dir) / self._SESSION_IDS_FILE if state_dir else None
        )
        self._restore_session_ids()

    def _evict_stale(self) -> None:
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
        cwd: str = "",
    ) -> CursorSessionWorker:
        self._evict_stale()
        self._last_access[thread_id] = time.monotonic()

        if thread_id not in self._workers:
            self._workers[thread_id] = CursorSessionWorker(
                model=model,
                api_key=api_key,
                cwd=cwd,
            )
            logger.debug("Created CursorSessionWorker for thread=%s", thread_id)
        return self._workers[thread_id]

    def get_lock(self, thread_id: str) -> asyncio.Lock:
        if thread_id not in self._locks:
            self._locks[thread_id] = asyncio.Lock()
        return self._locks[thread_id]

    def get_session_id(self, thread_id: str) -> str | None:
        return self._session_ids.get(thread_id)

    def set_session_id(self, thread_id: str, session_id: str) -> None:
        self._session_ids[thread_id] = session_id
        self._persist_session_ids()
        logger.debug("Recorded session_id=%s for thread=%s", session_id, thread_id)

    def _persist_session_ids(self) -> None:
        if not self._ids_path or not self._session_ids:
            return
        try:
            self._ids_path.parent.mkdir(parents=True, exist_ok=True)
            with open(self._ids_path, "w") as f:
                json.dump(self._session_ids, f)
        except OSError:
            logger.debug(
                "Could not persist session IDs to %s", self._ids_path, exc_info=True
            )

    def _restore_session_ids(self) -> None:
        if not self._ids_path:
            return
        try:
            with open(self._ids_path) as f:
                restored = json.load(f)
            if isinstance(restored, dict):
                self._session_ids.update(restored)
                logger.info(
                    "Restored %d Cursor session ID(s) from %s",
                    len(restored),
                    self._ids_path,
                )
        except FileNotFoundError:
            pass
        except (OSError, json.JSONDecodeError):
            logger.debug(
                "Could not restore session IDs from %s", self._ids_path, exc_info=True
            )

    def clear_session_ids(self) -> None:
        self._session_ids.clear()
        if self._ids_path:
            try:
                self._ids_path.unlink()
                logger.info("Cleared stale Cursor session IDs from %s", self._ids_path)
            except FileNotFoundError:
                pass
            except OSError:
                logger.debug(
                    "Could not remove session IDs at %s", self._ids_path, exc_info=True
                )

    async def interrupt(self, thread_id: str) -> None:
        worker = self._workers.get(thread_id)
        if worker:
            await worker.interrupt()
        else:
            logger.warning("No worker to interrupt for thread=%s", thread_id)

    def get_stderr_lines(self, thread_id: str) -> list[str]:
        worker = self._workers.get(thread_id)
        if worker:
            return worker.stderr_lines
        return []

    def get_all_stderr(self, max_per_worker: int = 10) -> list[str]:
        all_lines: list[str] = []
        for worker in self._workers.values():
            lines = worker.stderr_lines
            if lines:
                all_lines.extend(lines[-max_per_worker:])
        return all_lines

    async def shutdown(self) -> None:
        async def _stop_all() -> None:
            tasks = [worker.stop() for worker in self._workers.values()]
            if tasks:
                await asyncio.gather(*tasks, return_exceptions=True)

        try:
            await asyncio.wait_for(_stop_all(), timeout=SHUTDOWN_TIMEOUT_SEC * 2)
        except asyncio.TimeoutError:
            logger.warning(
                "CursorSessionManager: shutdown timed out after %ds",
                SHUTDOWN_TIMEOUT_SEC * 2,
            )

        self._workers.clear()
        self._last_access.clear()
        logger.info("CursorSessionManager: all workers shut down")
