"""
AG-UI gRPC Push Middleware — forwards events to ambient-api-server via gRPC.

Wraps an AG-UI event stream and pushes each event as a ``SessionMessage``
to the ``PushSessionMessage`` RPC on the ambient-api-server.  The push is
fire-and-forget: failures are logged but never propagate to the caller.

Usage::

    from ambient_runner.middleware import grpc_push_middleware

    async for event in grpc_push_middleware(
        bridge.run(input_data),
        session_id=session_id,
    ):
        yield encoder.encode(event)

When ``AMBIENT_GRPC_URL`` is unset the middleware is a transparent no-op
with zero overhead.
"""

from __future__ import annotations

import json
import logging
import os
from typing import AsyncIterator, Optional

from ag_ui.core import BaseEvent

logger = logging.getLogger(__name__)

_ENV_GRPC_URL = "AMBIENT_GRPC_URL"
_ENV_SESSION_ID = "SESSION_ID"


def _event_to_payload(event: BaseEvent) -> str:
    """Serialise an AG-UI event to a JSON string for the gRPC payload."""
    try:
        if hasattr(event, "model_dump"):
            return json.dumps(event.model_dump())
        if hasattr(event, "dict"):
            return json.dumps(event.dict())
        return json.dumps({"type": str(getattr(event, "type", "unknown"))})
    except Exception:
        return json.dumps({"type": str(getattr(event, "type", "unknown"))})


def _event_type_str(event: BaseEvent) -> str:
    raw = getattr(event, "type", None)
    if raw is None:
        return "unknown"
    return str(raw.value) if hasattr(raw, "value") else str(raw)


async def grpc_push_middleware(
    event_stream: AsyncIterator[BaseEvent],
    *,
    session_id: Optional[str] = None,
) -> AsyncIterator[BaseEvent]:
    """Wrap an AG-UI event stream with gRPC push to ambient-api-server.

    Args:
        event_stream: The upstream event stream.
        session_id: Session ID to push messages under.  Falls back to the
            ``SESSION_ID`` environment variable.

    Yields:
        The original events unchanged.
    """
    grpc_url = os.environ.get(_ENV_GRPC_URL, "").strip()
    if not grpc_url:
        async for event in event_stream:
            yield event
        return

    sid = session_id or os.environ.get(_ENV_SESSION_ID, "").strip()
    if not sid:
        logger.warning(
            "grpc_push_middleware: AMBIENT_GRPC_URL set but SESSION_ID missing — push disabled"
        )
        async for event in event_stream:
            yield event
        return

    grpc_client: Optional[object] = None
    try:
        from ambient_platform._grpc_client import AmbientGRPCClient

        grpc_client = AmbientGRPCClient.from_env()
        logger.info("grpc_push_middleware: connected to %s (session=%s)", grpc_url, sid)
    except Exception as exc:
        logger.warning(
            "grpc_push_middleware: failed to create gRPC client (%s) — push disabled",
            exc,
        )
        async for event in event_stream:
            yield event
        return

    try:
        async for event in event_stream:
            yield event
            _push_event(grpc_client, sid, event)
    finally:
        try:
            grpc_client.close()
        except Exception:
            pass


def _push_event(grpc_client: object, session_id: str, event: BaseEvent) -> None:
    """Fire-and-forget push of a single AG-UI event via gRPC."""
    try:
        event_type = _event_type_str(event)
        payload = _event_to_payload(event)
        grpc_client.session_messages.push(
            session_id=session_id,
            event_type=event_type,
            payload=payload,
        )
    except Exception as exc:
        logger.debug(
            "grpc_push_middleware: push failed (event=%s): %s",
            _event_type_str(event),
            exc,
        )
