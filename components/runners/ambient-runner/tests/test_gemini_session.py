"""Tests for GeminiSessionWorker and GeminiSessionManager."""

import asyncio
import time
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from ambient_runner.bridges.gemini_cli.session import (
    GEMINI_CLI_TIMEOUT_SEC,
    SHUTDOWN_TIMEOUT_SEC,
    WORKER_TTL_SEC,
    GeminiSessionManager,
    GeminiSessionWorker,
    _MAX_STDERR_LINES,
)


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------


def _make_mock_process(
    stdout_lines: list[bytes] | None = None,
    stderr_lines: list[bytes] | None = None,
    returncode: int = 0,
):
    """Create a mock asyncio.subprocess.Process.

    Args:
        stdout_lines: Lines to yield from stdout (newline-terminated bytes).
        stderr_lines: Lines to yield from stderr (newline-terminated bytes).
        returncode: Exit code.
    """
    proc = AsyncMock()
    proc.returncode = None  # Process is "running" initially

    # stdout -- yield control to event loop between lines so stderr task can run
    if stdout_lines is not None:

        async def _stdout_iter():
            for line in stdout_lines:
                await asyncio.sleep(0)  # Let other tasks run
                yield line
            # After all lines are yielded, mark process as finished
            proc.returncode = returncode

        proc.stdout = MagicMock()
        proc.stdout.__aiter__ = lambda self: _stdout_iter()
    else:
        proc.stdout = None

    # stderr
    if stderr_lines is not None:

        async def _stderr_iter():
            for line in stderr_lines:
                yield line

        proc.stderr = MagicMock()
        proc.stderr.__aiter__ = lambda self: _stderr_iter()
    else:
        proc.stderr = None

    async def _wait():
        await asyncio.sleep(0)  # Yield to let stderr task finish
        proc.returncode = returncode

    proc.wait = AsyncMock(side_effect=_wait)
    proc.terminate = MagicMock()
    proc.kill = MagicMock()
    proc.send_signal = MagicMock()

    return proc


# ------------------------------------------------------------------
# GeminiSessionWorker -- command construction
# ------------------------------------------------------------------


class TestWorkerCommandConstruction:
    """Verify the CLI command built by query()."""

    @pytest.mark.asyncio
    async def test_basic_command_flags(self):
        """The command must include -p, --output-format, --yolo, --model."""
        worker = GeminiSessionWorker(model="gemini-2.5-flash", api_key="key1")
        proc = _make_mock_process(
            stdout_lines=[b'{"type":"result","status":"success"}\n'],
            stderr_lines=[],
        )

        with patch("asyncio.create_subprocess_exec", return_value=proc) as mock_exec:
            lines = []
            async for line in worker.query("hello"):
                lines.append(line)

            call_args = mock_exec.call_args[0]
            assert call_args[0] == "gemini"
            assert "-p" in call_args
            assert "hello" in call_args
            assert "--output-format" in call_args
            assert "stream-json" in call_args
            assert "--yolo" in call_args
            assert "--model" in call_args
            assert "gemini-2.5-flash" in call_args

    @pytest.mark.asyncio
    async def test_resume_flag_with_session_id(self):
        """--resume <session_id> should be included when session_id is provided."""
        worker = GeminiSessionWorker(model="gemini-2.5-flash")
        proc = _make_mock_process(
            stdout_lines=[b'{"type":"result"}\n'],
            stderr_lines=[],
        )

        with patch("asyncio.create_subprocess_exec", return_value=proc) as mock_exec:
            async for _ in worker.query("hi", session_id="sess-abc"):
                pass

            call_args = mock_exec.call_args[0]
            assert "--resume" in call_args
            idx = list(call_args).index("--resume")
            assert call_args[idx + 1] == "sess-abc"

    @pytest.mark.asyncio
    async def test_no_resume_without_session_id(self):
        """--resume should NOT appear when session_id is None."""
        worker = GeminiSessionWorker(model="gemini-2.5-flash")
        proc = _make_mock_process(
            stdout_lines=[b'{"type":"result"}\n'],
            stderr_lines=[],
        )

        with patch("asyncio.create_subprocess_exec", return_value=proc) as mock_exec:
            async for _ in worker.query("hi"):
                pass

            call_args = mock_exec.call_args[0]
            assert "--resume" not in call_args


# ------------------------------------------------------------------
# GeminiSessionWorker -- environment variables
# ------------------------------------------------------------------


class TestWorkerEnvironment:
    """Verify env var handling for API key vs Vertex modes."""

    @pytest.mark.asyncio
    async def test_api_key_mode_sets_env(self):
        """In API key mode, GEMINI_API_KEY and GOOGLE_API_KEY are set."""
        worker = GeminiSessionWorker(model="m", api_key="my-secret-key")
        proc = _make_mock_process(
            stdout_lines=[b'{"done":true}\n'],
            stderr_lines=[],
        )

        with patch("asyncio.create_subprocess_exec", return_value=proc) as mock_exec:
            async for _ in worker.query("test"):
                pass

            call_kwargs = mock_exec.call_args[1]
            env = call_kwargs["env"]
            assert env["GEMINI_API_KEY"] == "my-secret-key"
            assert env["GOOGLE_API_KEY"] == "my-secret-key"

    @pytest.mark.asyncio
    async def test_vertex_mode_removes_api_keys(self, monkeypatch):
        """In Vertex mode, GEMINI_API_KEY and GOOGLE_API_KEY must be removed."""
        monkeypatch.setenv("GEMINI_API_KEY", "should-be-removed")
        monkeypatch.setenv("GOOGLE_API_KEY", "should-be-removed")
        monkeypatch.setenv("GOOGLE_CLOUD_PROJECT", "proj-1")
        monkeypatch.setenv("GOOGLE_CLOUD_LOCATION", "us-east1")

        worker = GeminiSessionWorker(model="m", use_vertex=True)
        proc = _make_mock_process(
            stdout_lines=[b'{"done":true}\n'],
            stderr_lines=[],
        )

        with patch("asyncio.create_subprocess_exec", return_value=proc) as mock_exec:
            async for _ in worker.query("test"):
                pass

            call_kwargs = mock_exec.call_args[1]
            env = call_kwargs["env"]
            assert "GEMINI_API_KEY" not in env
            assert "GOOGLE_API_KEY" not in env
            # Vertex-related vars should remain from os.environ
            assert env["GOOGLE_CLOUD_PROJECT"] == "proj-1"
            assert env["GOOGLE_CLOUD_LOCATION"] == "us-east1"

    @pytest.mark.asyncio
    async def test_cwd_passed_to_subprocess(self):
        """The subprocess should use the configured cwd."""
        worker = GeminiSessionWorker(model="m", cwd="/my/workspace")
        proc = _make_mock_process(
            stdout_lines=[b'{"done":true}\n'],
            stderr_lines=[],
        )

        with patch("asyncio.create_subprocess_exec", return_value=proc) as mock_exec:
            async for _ in worker.query("test"):
                pass

            call_kwargs = mock_exec.call_args[1]
            assert call_kwargs["cwd"] == "/my/workspace"


# ------------------------------------------------------------------
# GeminiSessionWorker -- stderr streaming
# ------------------------------------------------------------------


class TestWorkerStderrStreaming:
    """Test concurrent stderr capture."""

    @pytest.mark.asyncio
    async def test_stderr_lines_are_captured(self):
        """Stderr output should be captured in worker.stderr_lines."""
        worker = GeminiSessionWorker(model="m")
        proc = _make_mock_process(
            stdout_lines=[b'{"type":"result"}\n'],
            stderr_lines=[
                b"warning: something\n",
                b"info: loading model\n",
            ],
        )

        with patch("asyncio.create_subprocess_exec", return_value=proc):
            async for _ in worker.query("test"):
                pass

        assert "warning: something" in worker.stderr_lines
        assert "info: loading model" in worker.stderr_lines

    @pytest.mark.asyncio
    async def test_stderr_buffer_capped_at_max(self):
        """The stderr buffer should not exceed _MAX_STDERR_LINES."""
        worker = GeminiSessionWorker(model="m")
        # Generate more lines than the cap
        many_lines = [f"line-{i}\n".encode() for i in range(_MAX_STDERR_LINES + 50)]
        proc = _make_mock_process(
            stdout_lines=[b'{"type":"result"}\n'],
            stderr_lines=many_lines,
        )

        with patch("asyncio.create_subprocess_exec", return_value=proc):
            async for _ in worker.query("test"):
                pass

        assert len(worker.stderr_lines) <= _MAX_STDERR_LINES

    @pytest.mark.asyncio
    async def test_stderr_lines_property_returns_copy(self):
        """The stderr_lines property should return a copy."""
        worker = GeminiSessionWorker(model="m")
        worker._stderr_lines = ["line1", "line2"]
        result = worker.stderr_lines
        result.append("extra")
        assert len(worker._stderr_lines) == 2


# ------------------------------------------------------------------
# GeminiSessionWorker -- graceful shutdown
# ------------------------------------------------------------------


class TestWorkerGracefulShutdown:
    """Test the SIGTERM -> wait -> SIGKILL shutdown sequence."""

    @pytest.mark.asyncio
    async def test_stop_sends_sigterm(self):
        """stop() should call terminate() (SIGTERM)."""
        worker = GeminiSessionWorker(model="m")
        proc = MagicMock()
        proc.returncode = None
        proc.terminate = MagicMock()
        proc.wait = AsyncMock()
        proc.kill = MagicMock()
        worker._process = proc

        await worker.stop()

        proc.terminate.assert_called_once()

    @pytest.mark.asyncio
    async def test_stop_sends_sigkill_after_timeout(self):
        """If the process doesn't exit after SIGTERM, SIGKILL should follow."""
        worker = GeminiSessionWorker(model="m")
        proc = MagicMock()
        proc.returncode = None
        proc.terminate = MagicMock()
        proc.kill = MagicMock()

        # wait() times out after SIGTERM
        async def _slow_wait():
            await asyncio.sleep(100)  # Never finishes naturally

        proc.wait = AsyncMock(side_effect=_slow_wait)
        worker._process = proc

        # Patch SHUTDOWN_TIMEOUT_SEC to be very short for the test
        with patch(
            "ambient_runner.bridges.gemini_cli.session.SHUTDOWN_TIMEOUT_SEC", 0.01
        ):
            await worker._kill_process()

        proc.terminate.assert_called_once()
        proc.kill.assert_called_once()

    @pytest.mark.asyncio
    async def test_stop_noop_when_process_already_exited(self):
        """stop() should be a no-op if the process already exited."""
        worker = GeminiSessionWorker(model="m")
        proc = MagicMock()
        proc.returncode = 0  # Already exited
        proc.terminate = MagicMock()
        worker._process = proc

        await worker._kill_process()

        proc.terminate.assert_not_called()

    @pytest.mark.asyncio
    async def test_interrupt_sends_sigint(self):
        """interrupt() should send SIGINT."""
        import signal

        worker = GeminiSessionWorker(model="m")
        proc = MagicMock()
        proc.returncode = None
        proc.send_signal = MagicMock()
        worker._process = proc

        await worker.interrupt()

        proc.send_signal.assert_called_once_with(signal.SIGINT)


# ------------------------------------------------------------------
# GeminiSessionManager -- worker lifecycle
# ------------------------------------------------------------------


class TestSessionManagerWorkerLifecycle:
    """Test worker creation, reuse, and session ID tracking."""

    def test_get_or_create_returns_same_worker_for_same_thread(self):
        """Same thread_id should return the same worker instance."""
        mgr = GeminiSessionManager()
        w1 = mgr.get_or_create_worker("t1", model="m")
        w2 = mgr.get_or_create_worker("t1", model="m")
        assert w1 is w2

    def test_get_or_create_returns_different_workers_for_different_threads(self):
        """Different thread_ids should create different workers."""
        mgr = GeminiSessionManager()
        w1 = mgr.get_or_create_worker("t1", model="m")
        w2 = mgr.get_or_create_worker("t2", model="m")
        assert w1 is not w2

    def test_session_id_set_and_get(self):
        """set_session_id / get_session_id round-trip."""
        mgr = GeminiSessionManager()
        assert mgr.get_session_id("t1") is None
        mgr.set_session_id("t1", "sess-abc")
        assert mgr.get_session_id("t1") == "sess-abc"

    def test_session_id_overwrite(self):
        """set_session_id should overwrite previous values."""
        mgr = GeminiSessionManager()
        mgr.set_session_id("t1", "old")
        mgr.set_session_id("t1", "new")
        assert mgr.get_session_id("t1") == "new"

    def test_get_lock_returns_same_lock(self):
        """Same thread_id should return the same lock."""
        mgr = GeminiSessionManager()
        lock1 = mgr.get_lock("t1")
        lock2 = mgr.get_lock("t1")
        assert lock1 is lock2

    def test_get_lock_different_threads_get_different_locks(self):
        mgr = GeminiSessionManager()
        lock1 = mgr.get_lock("t1")
        lock2 = mgr.get_lock("t2")
        assert lock1 is not lock2


# ------------------------------------------------------------------
# GeminiSessionManager -- TTL eviction
# ------------------------------------------------------------------


class TestSessionManagerTTLEviction:
    """Test stale worker eviction."""

    def test_stale_workers_are_evicted(self):
        """Workers past WORKER_TTL_SEC should be removed on next access."""
        mgr = GeminiSessionManager()
        mgr.get_or_create_worker("stale-thread", model="m")
        mgr.set_session_id("stale-thread", "sess-1")

        # Fake the last access time to be old and reset eviction throttle
        mgr._last_access["stale-thread"] = time.monotonic() - WORKER_TTL_SEC - 1
        mgr._last_eviction = 0.0  # force eviction scan on next call

        # Accessing a different thread triggers eviction
        mgr.get_or_create_worker("new-thread", model="m")

        assert "stale-thread" not in mgr._workers
        assert mgr.get_session_id("stale-thread") is None
        assert "stale-thread" not in mgr._locks
        assert "stale-thread" not in mgr._last_access

    def test_fresh_workers_are_not_evicted(self):
        """Workers within TTL should not be evicted."""
        mgr = GeminiSessionManager()
        mgr.get_or_create_worker("fresh-thread", model="m")

        # Access a different thread -- should NOT evict fresh-thread
        mgr.get_or_create_worker("other-thread", model="m")

        assert "fresh-thread" in mgr._workers

    def test_eviction_updates_last_access_for_accessed_thread(self):
        """The accessed thread should have its last_access updated."""
        mgr = GeminiSessionManager()
        mgr.get_or_create_worker("t1", model="m")

        before = mgr._last_access["t1"]
        # Small sleep to ensure monotonic time advances
        import time as _time

        _time.sleep(0.01)
        mgr.get_or_create_worker("t1", model="m")

        assert mgr._last_access["t1"] >= before


# ------------------------------------------------------------------
# GeminiSessionManager -- stderr and interrupt
# ------------------------------------------------------------------


class TestSessionManagerStderrAndInterrupt:
    """Test stderr retrieval and interrupt forwarding."""

    def test_get_stderr_lines_returns_worker_stderr(self):
        """get_stderr_lines should return the worker's stderr buffer."""
        mgr = GeminiSessionManager()
        worker = mgr.get_or_create_worker("t1", model="m")
        worker._stderr_lines = ["err1", "err2"]

        lines = mgr.get_stderr_lines("t1")
        assert lines == ["err1", "err2"]

    def test_get_stderr_lines_unknown_thread(self):
        """get_stderr_lines for an unknown thread returns empty list."""
        mgr = GeminiSessionManager()
        assert mgr.get_stderr_lines("unknown") == []

    @pytest.mark.asyncio
    async def test_interrupt_forwards_to_worker(self):
        """interrupt() should delegate to the worker's interrupt()."""
        mgr = GeminiSessionManager()
        worker = mgr.get_or_create_worker("t1", model="m")
        worker.interrupt = AsyncMock()

        await mgr.interrupt("t1")
        worker.interrupt.assert_awaited_once()


# ------------------------------------------------------------------
# GeminiSessionManager -- shutdown
# ------------------------------------------------------------------


@pytest.mark.asyncio
class TestSessionManagerShutdown:
    """Test graceful shutdown of all workers."""

    async def test_shutdown_stops_all_workers(self):
        mgr = GeminiSessionManager()
        w1 = mgr.get_or_create_worker("t1", model="m")
        w2 = mgr.get_or_create_worker("t2", model="m")
        w1.stop = AsyncMock()
        w2.stop = AsyncMock()

        await mgr.shutdown()

        w1.stop.assert_awaited_once()
        w2.stop.assert_awaited_once()
        assert len(mgr._workers) == 0

    async def test_shutdown_clears_last_access(self):
        mgr = GeminiSessionManager()
        mgr.get_or_create_worker("t1", model="m")

        await mgr.shutdown()

        assert len(mgr._last_access) == 0

    async def test_shutdown_with_no_workers(self):
        """Shutdown should not raise when there are no workers."""
        mgr = GeminiSessionManager()
        await mgr.shutdown()
