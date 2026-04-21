"""Tests for GRPCSessionListener and GRPCMessageWriter in grpc_transport.py.

Coverage targets:
- GRPCSessionListener: ready event lifecycle, message type filtering,
  fan-out to SSE queues, stop/cancel, bridge.run() called with correct RunnerInput,
  exception in bridge.run() synthesizes RUN_ERROR, invalid JSON fallback
- GRPCMessageWriter: MESSAGES_SNAPSHOT accumulation, RUN_FINISHED/RUN_ERROR push,
  non-terminal events ignored, push offloaded to executor (non-blocking),
  push failure logged without re-raising
- _synthesize_run_error: feeds RUN_ERROR to SSE queue, schedules writer persist
"""

import asyncio
import json
import uuid
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from ag_ui.core import EventType

from tests.conftest import (
    async_event_stream,
    make_run_finished,
    make_text_content,
    make_text_start,
)

from ambient_runner.bridges.claude.grpc_transport import (
    GRPCMessageWriter,
    GRPCSessionListener,
    _synthesize_run_error,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_session_message(event_type: str, payload: str, seq: int = 1):
    msg = MagicMock()
    msg.event_type = event_type
    msg.payload = payload
    msg.seq = seq
    msg.session_id = "sess-1"
    return msg


def _make_runner_payload(
    thread_id: str = "t-1",
    run_id: str = "r-1",
    content: str = "hello",
) -> str:
    return json.dumps(
        {
            "threadId": thread_id,
            "runId": run_id,
            "messages": [{"id": str(uuid.uuid4()), "role": "user", "content": content}],
        }
    )


def _make_grpc_client(messages=None):
    """Return a mock AmbientGRPCClient whose watch() yields the given messages."""
    client = MagicMock()
    client.session_messages.watch.return_value = iter(messages or [])
    client.session_messages.push.return_value = MagicMock(seq=1)
    return client


def _make_bridge(active_streams=None):
    bridge = MagicMock()
    bridge._active_streams = active_streams if active_streams is not None else {}
    return bridge


# ---------------------------------------------------------------------------
# GRPCSessionListener — ready event
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestGRPCSessionListenerReady:
    async def test_ready_set_after_watch_opens(self):
        client = _make_grpc_client(messages=[])
        bridge = _make_bridge()
        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            assert listener.ready.is_set()
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

    async def test_ready_not_set_before_watch(self):
        bridge = _make_bridge()
        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        assert not listener.ready.is_set()

    async def test_ready_set_on_successful_watch(self):
        client = _make_grpc_client(messages=[])
        bridge = _make_bridge()
        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            assert listener.ready.is_set()
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass


# ---------------------------------------------------------------------------
# GRPCSessionListener — message filtering
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestGRPCSessionListenerFiltering:
    async def test_non_user_messages_do_not_trigger_run(self):
        msgs = [
            _make_session_message("assistant", '{"foo": "bar"}', seq=1),
            _make_session_message("system", "{}", seq=2),
        ]
        client = _make_grpc_client(messages=msgs)
        bridge = _make_bridge()
        bridge.run = AsyncMock(return_value=async_event_stream([]))

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.1)
            bridge.run.assert_not_called()
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

    async def test_user_message_triggers_bridge_run(self):
        payload = _make_runner_payload(
            thread_id="t-1", run_id="r-1", content="do the thing"
        )
        msgs = [_make_session_message("user", payload, seq=1)]
        client = _make_grpc_client(messages=msgs)
        bridge = _make_bridge()

        run_inputs = []

        async def fake_run(input_data):
            run_inputs.append(input_data)
            yield make_text_start()
            yield make_run_finished()

        bridge.run = fake_run
        bridge._active_streams = {}

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.3)
            assert len(run_inputs) == 1
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

    async def test_user_message_run_called_with_correct_thread_id(self):
        """bridge.run() must receive input_data with thread_id from the message payload."""
        payload = _make_runner_payload(
            thread_id="t-specific", run_id="r-42", content="hello"
        )
        msgs = [_make_session_message("user", payload, seq=5)]
        client = _make_grpc_client(messages=msgs)
        bridge = _make_bridge()

        run_inputs = []

        async def fake_run(input_data):
            run_inputs.append(input_data)
            yield make_run_finished()

        bridge.run = fake_run
        bridge._active_streams = {}

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.3)
            assert len(run_inputs) == 1
            assert run_inputs[0].thread_id == "t-specific"
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

    async def test_invalid_json_payload_uses_raw_as_content_fallback(self):
        """Invalid JSON in payload falls back to creating a message with raw payload as content."""
        msgs = [_make_session_message("user", "not-json", seq=1)]
        client = _make_grpc_client(messages=msgs)
        bridge = _make_bridge()

        run_inputs = []

        async def fake_run(input_data):
            run_inputs.append(input_data)
            yield make_run_finished()

        bridge.run = fake_run
        bridge._active_streams = {}

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.3)
            assert len(run_inputs) == 1
            msgs_in_input = run_inputs[0].messages
            assert len(msgs_in_input) == 1
            msg = msgs_in_input[0]
            role = msg["role"] if isinstance(msg, dict) else getattr(msg, "role", None)
            content = (
                msg["content"]
                if isinstance(msg, dict)
                else getattr(msg, "content", None)
            )
            assert role == "user"
            assert content == "not-json"
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

    async def test_bridge_run_exception_synthesizes_run_error_to_sse_queue(self):
        """If bridge.run() raises, a RUN_ERROR event must be fed to the SSE tap queue."""
        payload = _make_runner_payload(thread_id="t-err", run_id="r-err")
        msgs = [_make_session_message("user", payload, seq=1)]
        client = _make_grpc_client(messages=msgs)

        tap_queue: asyncio.Queue = asyncio.Queue(maxsize=100)
        active_streams = {"t-err": tap_queue}
        bridge = _make_bridge(active_streams=active_streams)

        async def exploding_run(input_data):
            raise RuntimeError("boom")
            yield  # make it a generator

        bridge.run = exploding_run

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.5)

            run_error_events = []
            while not tap_queue.empty():
                ev = tap_queue.get_nowait()
                raw = getattr(ev, "type", None)
                ev_str = raw.value if hasattr(raw, "value") else str(raw)
                if "RUN_ERROR" in ev_str:
                    run_error_events.append(ev)
            assert len(run_error_events) >= 1
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass


# ---------------------------------------------------------------------------
# GRPCSessionListener — fan-out
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestGRPCSessionListenerFanOut:
    async def test_events_fed_to_active_streams_queue(self):
        payload = _make_runner_payload(thread_id="t-fanout", run_id="r-1")
        msgs = [_make_session_message("user", payload, seq=1)]
        client = _make_grpc_client(messages=msgs)

        received_events = []
        tap_queue: asyncio.Queue = asyncio.Queue(maxsize=100)

        bridge = _make_bridge(active_streams={"t-fanout": tap_queue})
        events = [make_text_start(), make_text_content(), make_run_finished()]

        async def fake_run(input_data):
            for e in events:
                yield e

        bridge.run = fake_run

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.3)
            while not tap_queue.empty():
                received_events.append(tap_queue.get_nowait())
            assert len(received_events) == len(events)
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

    async def test_no_active_stream_fan_out_skipped_silently(self):
        payload = _make_runner_payload(thread_id="t-1", run_id="r-1")
        msgs = [_make_session_message("user", payload, seq=1)]
        client = _make_grpc_client(messages=msgs)
        bridge = _make_bridge(active_streams={})

        events = [make_text_start(), make_run_finished()]

        async def fake_run(input_data):
            for e in events:
                yield e

        bridge.run = fake_run

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.3)
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

    async def test_full_queue_drops_event_without_raising(self):
        payload = _make_runner_payload(thread_id="t-full", run_id="r-1")
        msgs = [_make_session_message("user", payload, seq=1)]
        client = _make_grpc_client(messages=msgs)

        full_queue: asyncio.Queue = asyncio.Queue(maxsize=1)
        full_queue.put_nowait(make_text_start())

        bridge = _make_bridge(active_streams={"t-full": full_queue})
        events = [make_text_start(), make_run_finished()]

        async def fake_run(input_data):
            for e in events:
                yield e

        bridge.run = fake_run

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.3)
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

    async def test_active_streams_entry_removed_after_turn(self):
        payload = _make_runner_payload(thread_id="t-cleanup", run_id="r-1")
        msgs = [_make_session_message("user", payload, seq=1)]
        client = _make_grpc_client(messages=msgs)

        tap_queue: asyncio.Queue = asyncio.Queue(maxsize=100)
        active_streams = {"t-cleanup": tap_queue}
        bridge = _make_bridge(active_streams=active_streams)

        async def fake_run(input_data):
            yield make_run_finished()

        bridge.run = fake_run

        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client

        task = asyncio.create_task(listener._listen_loop())
        try:
            await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
            await asyncio.sleep(0.3)
            assert "t-cleanup" not in active_streams
        finally:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass


# ---------------------------------------------------------------------------
# GRPCSessionListener — stop
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestGRPCSessionListenerStop:
    async def test_stop_cancels_task(self):
        client = _make_grpc_client(messages=[])
        bridge = _make_bridge()
        listener = GRPCSessionListener(
            bridge=bridge, session_id="s-1", grpc_url="localhost:9000"
        )
        listener._grpc_client = client
        listener._task = asyncio.create_task(listener._listen_loop())

        await asyncio.wait_for(listener.ready.wait(), timeout=2.0)
        await listener.stop()
        assert listener._task.done()


# ---------------------------------------------------------------------------
# GRPCMessageWriter — consume
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestGRPCMessageWriterConsume:
    def _make_messages_snapshot(self, messages):
        event = MagicMock()
        event.type = EventType.MESSAGES_SNAPSHOT
        event.messages = messages
        return event

    def _make_run_finished_event(self):
        event = MagicMock()
        event.type = EventType.RUN_FINISHED
        return event

    def _make_run_error_event(self):
        event = MagicMock()
        event.type = EventType.RUN_ERROR
        return event

    def _make_text_event(self):
        event = MagicMock()
        event.type = EventType.TEXT_MESSAGE_CONTENT
        return event

    def _writer(self):
        client = MagicMock()
        client.session_messages.push.return_value = MagicMock(seq=1)
        return GRPCMessageWriter(
            session_id="s-1", run_id="r-1", grpc_client=client
        ), client

    async def test_messages_snapshot_accumulated(self):
        writer, _ = self._writer()
        msg = MagicMock()
        msg.model_dump.return_value = {"role": "assistant", "content": "hi"}
        snap = self._make_messages_snapshot([msg])
        await writer.consume(snap)
        assert len(writer._accumulated_messages) == 1

    async def test_run_finished_pushes_completed(self):
        writer, client = self._writer()
        msg = MagicMock()
        msg.model_dump.return_value = {"role": "assistant", "content": "done"}
        snap = self._make_messages_snapshot([msg])
        await writer.consume(snap)
        await writer.consume(self._make_run_finished_event())

        client.session_messages.push.assert_called_once()
        call = client.session_messages.push.call_args
        assert call[0][0] == "s-1"
        assert call[1]["event_type"] == "assistant"
        assert call[1]["payload"] == "done"

    async def test_run_error_pushes_error_status(self):
        writer, client = self._writer()
        await writer.consume(self._make_run_error_event())

        client.session_messages.push.assert_called_once()
        assert client.session_messages.push.call_args[1]["event_type"] == "assistant"

    async def test_non_terminal_events_do_not_push(self):
        writer, client = self._writer()
        await writer.consume(self._make_text_event())
        client.session_messages.push.assert_not_called()

    async def test_unknown_event_type_ignored(self):
        writer, client = self._writer()
        event = MagicMock()
        event.type = None
        await writer.consume(event)
        client.session_messages.push.assert_not_called()

    async def test_latest_snapshot_replaces_previous(self):
        writer, client = self._writer()
        msg1 = MagicMock()
        msg1.model_dump.return_value = {"role": "assistant", "content": "first"}
        msg2 = MagicMock()
        msg2.model_dump.return_value = {"role": "assistant", "content": "second"}

        await writer.consume(self._make_messages_snapshot([msg1]))
        await writer.consume(self._make_messages_snapshot([msg2]))
        await writer.consume(self._make_run_finished_event())

        assert client.session_messages.push.call_args[1]["payload"] == "second"

    async def test_no_grpc_client_write_skipped(self):
        writer = GRPCMessageWriter(session_id="s-1", run_id="r-1", grpc_client=None)
        event = MagicMock()
        event.type = EventType.RUN_FINISHED
        await writer.consume(event)

    async def test_push_includes_correct_session_id(self):
        writer, client = self._writer()
        await writer.consume(self._make_run_finished_event())
        assert client.session_messages.push.call_args[0][0] == "s-1"
        assert client.session_messages.push.call_args[1]["event_type"] == "assistant"

    async def test_push_offloaded_to_executor_not_inline(self):
        """The synchronous gRPC push must be run via run_in_executor, not inline."""
        writer, client = self._writer()

        executor_calls = []
        real_loop = asyncio.get_event_loop()
        original = real_loop.run_in_executor

        async def capturing(executor, fn, *args):
            executor_calls.append(fn)
            return await original(executor, fn, *args)

        with patch.object(real_loop, "run_in_executor", side_effect=capturing):
            await writer.consume(self._make_run_finished_event())

        assert len(executor_calls) == 1

    async def test_push_failure_does_not_raise(self):
        """If the gRPC push in executor fails, _write_message must not re-raise."""
        writer, client = self._writer()
        client.session_messages.push.side_effect = RuntimeError("rpc unavailable")

        await writer.consume(self._make_run_finished_event())


# ---------------------------------------------------------------------------
# _synthesize_run_error — standalone helper
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestSynthesizeRunError:
    async def test_feeds_run_error_event_to_sse_queue(self):
        """_synthesize_run_error must put a RUN_ERROR event into the SSE tap queue."""
        tap_queue: asyncio.Queue = asyncio.Queue(maxsize=100)
        active_streams = {"t-synth": tap_queue}

        client = MagicMock()
        client.session_messages.push.return_value = MagicMock(seq=1)
        writer = GRPCMessageWriter(session_id="s-1", run_id="r-1", grpc_client=client)

        _synthesize_run_error("t-synth", "test error", active_streams, writer)

        await asyncio.sleep(0.1)

        assert not tap_queue.empty()
        ev = tap_queue.get_nowait()
        raw = getattr(ev, "type", None)
        ev_str = raw.value if hasattr(raw, "value") else str(raw)
        assert "RUN_ERROR" in ev_str

    async def test_no_sse_queue_does_not_raise(self):
        """When no SSE queue is registered, _synthesize_run_error must not raise."""
        active_streams: dict = {}

        client = MagicMock()
        client.session_messages.push.return_value = MagicMock(seq=1)
        writer = GRPCMessageWriter(session_id="s-1", run_id="r-1", grpc_client=client)

        _synthesize_run_error("t-missing", "test error", active_streams, writer)
        await asyncio.sleep(0.1)

    async def test_schedules_writer_error_persist(self):
        """_synthesize_run_error must schedule writer._write_message(status='error')."""
        tap_queue: asyncio.Queue = asyncio.Queue(maxsize=100)
        active_streams = {"t-wr": tap_queue}

        client = MagicMock()
        client.session_messages.push.return_value = MagicMock(seq=1)
        writer = GRPCMessageWriter(session_id="s-1", run_id="r-1", grpc_client=client)

        write_calls = []
        original_write = writer._write_message

        async def tracking_write(status):
            write_calls.append(status)
            return await original_write(status)

        writer._write_message = tracking_write

        _synthesize_run_error("t-wr", "boom", active_streams, writer)
        await asyncio.sleep(0.2)

        assert "error" in write_calls
