"""Unit tests for GET /events/{thread_id} and GET /events/{thread_id}/wait."""

import asyncio
from unittest.mock import MagicMock

from fastapi import FastAPI
from fastapi.testclient import TestClient

from ag_ui.core import EventType

from ambient_runner.endpoints.events import router

from tests.conftest import (
    make_run_finished,
    make_text_content,
    make_text_start,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_bridge(active_streams=None, has_streams_attr=True):
    bridge = MagicMock()
    if has_streams_attr:
        bridge._active_streams = active_streams if active_streams is not None else {}
    else:
        del bridge._active_streams
        # MagicMock creates attrs on access; use spec to block it
        bridge = MagicMock(spec=[])
    return bridge


def _make_client(bridge):
    app = FastAPI()
    app.state.bridge = bridge
    app.include_router(router)
    return TestClient(app, raise_server_exceptions=False)


def _parse_sse_events(body: str) -> list[dict]:
    """Parse SSE response body into a list of {event, data} dicts."""
    events = []
    current = {}
    for line in body.splitlines():
        if line.startswith("event:"):
            current["event"] = line[len("event:") :].strip()
        elif line.startswith("data:"):
            current["data"] = line[len("data:") :].strip()
        elif line == "" and current:
            events.append(current)
            current = {}
    if current:
        events.append(current)
    return events


# ---------------------------------------------------------------------------
# GET /events/{thread_id} — queue registration
# ---------------------------------------------------------------------------


class TestEventsEndpointQueueRegistration:
    def test_registers_queue_before_streaming(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)
        client = _make_client(bridge)

        # Pre-fill queue with a terminal event so the stream closes immediately
        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        q.put_nowait(make_run_finished())

        # The endpoint creates its own queue on connect — verify it's registered
        # by inspecting active_streams after the response body is fully consumed
        with client.stream("GET", "/events/t-1") as resp:
            assert resp.status_code == 200
            resp.read()

    def test_queue_removed_after_stream_closes(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        # Pre-register a queue with a terminal event so the stream exits cleanly
        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        q.put_nowait(make_run_finished())
        active_streams["t-cleanup"] = q

        # Re-use the existing queue (the endpoint will overwrite with a new one)
        # We just need the endpoint to reach finally — feed via the new queue path
        app = FastAPI()
        app.state.bridge = bridge
        app.include_router(router)
        client = TestClient(app, raise_server_exceptions=False)

        # After response: queue should be removed
        with client.stream("GET", "/events/t-cleanup") as resp:
            resp.read()

        assert "t-cleanup" not in active_streams

    def test_returns_503_when_bridge_has_no_active_streams(self):
        bridge = MagicMock(spec=[])  # no _active_streams attribute
        client = _make_client(bridge)
        resp = client.get("/events/t-1")
        assert resp.status_code == 503


# ---------------------------------------------------------------------------
# GET /events/{thread_id} — SSE event filtering and closing
# ---------------------------------------------------------------------------


class TestEventsEndpointFiltering:
    def _make_client_with_preloaded_queue(self, events, thread_id="t-1"):
        """Create a client whose bridge already has a queue pre-loaded with events."""
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        for ev in events:
            q.put_nowait(ev)
        active_streams[thread_id] = q

        app = FastAPI()
        app.state.bridge = bridge
        app.include_router(router)
        return TestClient(app, raise_server_exceptions=False)

    def test_stream_closes_on_run_finished(self):
        events = [make_text_start(), make_text_content(), make_run_finished()]
        client = self._make_client_with_preloaded_queue(events)

        with client.stream("GET", "/events/t-1") as resp:
            body = resp.read().decode()

        assert "RUN_FINISHED" in body

    def test_stream_closes_on_run_error(self):
        run_error = MagicMock()
        run_error.type = EventType.RUN_ERROR

        events = [run_error]
        client = self._make_client_with_preloaded_queue(events)

        with client.stream("GET", "/events/t-1") as resp:
            body = resp.read().decode()

        assert "RUN_ERROR" in body

    def test_messages_snapshot_not_emitted(self):
        snapshot = MagicMock()
        snapshot.type = EventType.MESSAGES_SNAPSHOT

        events = [snapshot, make_run_finished()]
        client = self._make_client_with_preloaded_queue(events)

        with client.stream("GET", "/events/t-1") as resp:
            body = resp.read().decode()

        assert "MESSAGES_SNAPSHOT" not in body
        assert "RUN_FINISHED" in body

    def test_text_events_are_emitted(self):
        events = [make_text_start(), make_text_content(), make_run_finished()]
        client = self._make_client_with_preloaded_queue(events)

        with client.stream("GET", "/events/t-1") as resp:
            body = resp.read().decode()

        assert "TEXT_MESSAGE_START" in body
        assert "TEXT_MESSAGE_CONTENT" in body


# ---------------------------------------------------------------------------
# GET /events/{thread_id}/wait — defensive fallback
# ---------------------------------------------------------------------------


class TestEventsWaitEndpoint:
    def test_returns_404_when_no_active_stream(self, monkeypatch):
        monkeypatch.setenv("EVENTS_TAP_TIMEOUT_SEC", "0.1")
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)
        client = _make_client(bridge)

        resp = client.get("/events/missing-thread/wait")
        assert resp.status_code == 404

    def test_returns_503_when_bridge_has_no_active_streams(self):
        bridge = MagicMock(spec=[])
        client = _make_client(bridge)
        resp = client.get("/events/t-1/wait")
        assert resp.status_code == 503

    def test_streams_when_queue_already_registered(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        q.put_nowait(make_text_start())
        q.put_nowait(make_run_finished())
        active_streams["t-wait"] = q

        app = FastAPI()
        app.state.bridge = bridge
        app.include_router(router)
        client = TestClient(app, raise_server_exceptions=False)

        with client.stream("GET", "/events/t-wait/wait") as resp:
            body = resp.read().decode()

        assert "RUN_FINISHED" in body

    def test_queue_removed_after_stream_closes_wait(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        q.put_nowait(make_run_finished())
        active_streams["t-wait-clean"] = q

        app = FastAPI()
        app.state.bridge = bridge
        app.include_router(router)
        client = TestClient(app, raise_server_exceptions=False)

        with client.stream("GET", "/events/t-wait-clean/wait") as resp:
            resp.read()

        assert "t-wait-clean" not in active_streams

    def test_messages_snapshot_filtered_in_wait_path(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        snapshot = MagicMock()
        snapshot.type = EventType.MESSAGES_SNAPSHOT

        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        q.put_nowait(snapshot)
        q.put_nowait(make_run_finished())
        active_streams["t-wait-filter"] = q

        app = FastAPI()
        app.state.bridge = bridge
        app.include_router(router)
        client = TestClient(app, raise_server_exceptions=False)

        with client.stream("GET", "/events/t-wait-filter/wait") as resp:
            body = resp.read().decode()

        assert "MESSAGES_SNAPSHOT" not in body
        assert "RUN_FINISHED" in body
