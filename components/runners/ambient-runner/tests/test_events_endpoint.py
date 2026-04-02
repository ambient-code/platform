"""Unit tests for GET /events/{thread_id} and GET /events/{thread_id}/wait.

Coverage targets:
- Queue registration before streaming begins
- Identity-safe cleanup (only removes if queue is the same object)
- Duplicate registration warning (second connect logs warning, replaces queue)
- 503 when bridge has no _active_streams attribute
- MESSAGES_SNAPSHOT filtered from output
- Stream closes on RUN_FINISHED / RUN_ERROR
- Text events emitted
- /wait: 404 on timeout, 503 when no attr, streams when queue registered,
  MESSAGES_SNAPSHOT filtered in wait path
- Real async producer: background task puts events into the actual registered
  queue while the endpoint is streaming, verifying end-to-end delivery
"""

import asyncio
from unittest.mock import MagicMock

import httpx
import pytest
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


def _make_bridge(active_streams=None):
    bridge = MagicMock()
    bridge._active_streams = active_streams if active_streams is not None else {}
    return bridge


def _make_app(bridge):
    app = FastAPI()
    app.state.bridge = bridge
    app.include_router(router)
    return app


def _make_client(bridge):
    return TestClient(_make_app(bridge), raise_server_exceptions=False)


# ---------------------------------------------------------------------------
# GET /events/{thread_id} — 503 guard (sync, instant)
# ---------------------------------------------------------------------------


class TestEventsEndpointGuards:
    def test_returns_503_when_bridge_has_no_active_streams(self):
        bridge = MagicMock(spec=[])
        client = _make_client(bridge)
        resp = client.get("/events/t-1")
        assert resp.status_code == 503

    def test_wait_returns_503_when_bridge_has_no_active_streams(self):
        bridge = MagicMock(spec=[])
        client = _make_client(bridge)
        resp = client.get("/events/t-1/wait")
        assert resp.status_code == 503

    def test_wait_returns_404_when_no_active_stream(self, monkeypatch):
        monkeypatch.setenv("EVENTS_TAP_TIMEOUT_SEC", "0.05")
        bridge = _make_bridge(active_streams={})
        client = _make_client(bridge)
        resp = client.get("/events/missing-thread/wait")
        assert resp.status_code == 404


# ---------------------------------------------------------------------------
# GET /events/{thread_id} — async producer tests (real queue delivery)
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestEventsEndpointAsyncDelivery:
    """Use httpx.AsyncClient with ASGI transport to test real async queue delivery.

    The background producer task polls active_streams until the endpoint has
    registered its own queue, then feeds events into it — exactly mimicking
    how GRPCSessionListener would deliver events in production.
    """

    async def _stream_events(
        self, app, path: str, active_streams: dict, events_to_put: list
    ) -> str:
        """Open SSE stream and concurrently feed events into the endpoint's registered queue."""
        collected = []

        async def producer():
            deadline = asyncio.get_event_loop().time() + 3.0
            thread_id = path.split("/events/")[-1].split("/")[0]
            while asyncio.get_event_loop().time() < deadline:
                q = active_streams.get(thread_id)
                if q is not None:
                    for ev in events_to_put:
                        await q.put(ev)
                    return
                await asyncio.sleep(0.005)

        async with httpx.AsyncClient(
            transport=httpx.ASGITransport(app=app),
            base_url="http://test",
        ) as client:
            producer_task = asyncio.create_task(producer())
            async with client.stream("GET", path) as resp:
                assert resp.status_code == 200
                async for chunk in resp.aiter_bytes():
                    collected.append(chunk.decode())
            await producer_task

        return "".join(collected)

    async def test_run_finished_closes_stream(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)
        app = _make_app(bridge)

        body = await self._stream_events(
            app,
            "/events/t-async-fin",
            active_streams,
            [make_text_start(), make_run_finished()],
        )
        assert "RUN_FINISHED" in body

    async def test_run_error_closes_stream(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)
        app = _make_app(bridge)

        from ag_ui.core import RunErrorEvent

        run_error = RunErrorEvent(message="test error", code="TEST")

        body = await self._stream_events(
            app, "/events/t-async-err", active_streams, [run_error]
        )
        assert "RUN_ERROR" in body

    async def test_messages_snapshot_filtered(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)
        app = _make_app(bridge)

        snapshot = MagicMock()
        snapshot.type = EventType.MESSAGES_SNAPSHOT

        body = await self._stream_events(
            app, "/events/t-async-snap", active_streams, [snapshot, make_run_finished()]
        )
        assert "MESSAGES_SNAPSHOT" not in body
        assert "RUN_FINISHED" in body

    async def test_text_events_delivered(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)
        app = _make_app(bridge)

        body = await self._stream_events(
            app,
            "/events/t-async-text",
            active_streams,
            [make_text_start(), make_text_content(), make_run_finished()],
        )
        assert "TEXT_MESSAGE_START" in body
        assert "TEXT_MESSAGE_CONTENT" in body

    async def test_queue_removed_from_active_streams_after_stream_closes(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)
        app = _make_app(bridge)

        await self._stream_events(
            app, "/events/t-async-cleanup", active_streams, [make_run_finished()]
        )
        assert "t-async-cleanup" not in active_streams

    async def test_identity_safe_cleanup_preserves_newer_queue(self):
        """After stream closes, the endpoint must not remove a queue it didn't create."""
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)
        app = _make_app(bridge)

        newer_queue = asyncio.Queue(maxsize=100)

        async def producer():
            deadline = asyncio.get_event_loop().time() + 3.0
            while asyncio.get_event_loop().time() < deadline:
                q = active_streams.get("t-id-safe")
                if q is not None:
                    active_streams["t-id-safe"] = newer_queue
                    await q.put(make_run_finished())
                    return
                await asyncio.sleep(0.005)

        async with httpx.AsyncClient(
            transport=httpx.ASGITransport(app=app),
            base_url="http://test",
        ) as client:
            producer_task = asyncio.create_task(producer())
            async with client.stream("GET", "/events/t-id-safe") as resp:
                async for _ in resp.aiter_bytes():
                    pass
            await producer_task

        assert active_streams.get("t-id-safe") is newer_queue


# ---------------------------------------------------------------------------
# GET /events/{thread_id}/wait — async variants
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestEventsWaitEndpointAsync:
    async def test_streams_when_queue_pre_registered(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        await q.put(make_text_start())
        await q.put(make_run_finished())
        active_streams["t-wait-async"] = q

        app = _make_app(bridge)

        collected = []
        async with httpx.AsyncClient(
            transport=httpx.ASGITransport(app=app),
            base_url="http://test",
        ) as client:
            async with client.stream("GET", "/events/t-wait-async/wait") as resp:
                assert resp.status_code == 200
                async for chunk in resp.aiter_bytes():
                    collected.append(chunk.decode())

        body = "".join(collected)
        assert "RUN_FINISHED" in body

    async def test_wait_messages_snapshot_filtered(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        snapshot = MagicMock()
        snapshot.type = EventType.MESSAGES_SNAPSHOT

        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        await q.put(snapshot)
        await q.put(make_run_finished())
        active_streams["t-wait-filter"] = q

        app = _make_app(bridge)

        collected = []
        async with httpx.AsyncClient(
            transport=httpx.ASGITransport(app=app),
            base_url="http://test",
        ) as client:
            async with client.stream("GET", "/events/t-wait-filter/wait") as resp:
                async for chunk in resp.aiter_bytes():
                    collected.append(chunk.decode())

        body = "".join(collected)
        assert "MESSAGES_SNAPSHOT" not in body
        assert "RUN_FINISHED" in body

    async def test_wait_queue_removed_after_stream(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        q: asyncio.Queue = asyncio.Queue(maxsize=100)
        await q.put(make_run_finished())
        active_streams["t-wait-cleanup"] = q

        app = _make_app(bridge)

        async with httpx.AsyncClient(
            transport=httpx.ASGITransport(app=app),
            base_url="http://test",
        ) as client:
            async with client.stream("GET", "/events/t-wait-cleanup/wait") as resp:
                async for _ in resp.aiter_bytes():
                    pass

        assert "t-wait-cleanup" not in active_streams

    async def test_wait_identity_safe_cleanup(self):
        active_streams = {}
        bridge = _make_bridge(active_streams=active_streams)

        old_queue: asyncio.Queue = asyncio.Queue(maxsize=100)
        await old_queue.put(make_run_finished())
        active_streams["t-wait-id"] = old_queue

        newer_queue = asyncio.Queue(maxsize=100)

        app = _make_app(bridge)

        async with httpx.AsyncClient(
            transport=httpx.ASGITransport(app=app),
            base_url="http://test",
        ) as client:
            async with client.stream("GET", "/events/t-wait-id/wait") as resp:
                active_streams["t-wait-id"] = newer_queue
                async for _ in resp.aiter_bytes():
                    pass

        assert active_streams.get("t-wait-id") is newer_queue
