"""
OperationalEventWriter — pushes granular operational events to the session messages API.

Sits alongside GRPCMessageWriter in the event fan-out loop. Maps AG-UI event types
to operational event_type strings that the Logs tab UI filters for:
  tool_use, tool_result, error, lifecycle, system

Skips TEXT_MESSAGE_* (belongs in Chat tab) and STATE_SNAPSHOT (too noisy).
Pushes are fire-and-forget with no retry beyond the gRPC client's built-in retry.
"""

import asyncio
import json
import logging
from typing import TYPE_CHECKING, Optional

from ag_ui.core import BaseEvent

if TYPE_CHECKING:
    from ambient_runner._grpc_client import AmbientGRPCClient

logger = logging.getLogger(__name__)

# ag_ui.core.EventType.TOOL_CALL_RESULT is emitted by handlers.py:199
# (ToolCallResultEvent after ToolCallEndEvent). See test_operational_events.py.
_AGUI_TO_EVENT_TYPE = {
    "TOOL_CALL_START": "tool_use",
    "TOOL_CALL_RESULT": "tool_result",
    "RUN_STARTED": "lifecycle",
    "RUN_FINISHED": "lifecycle",
    "RUN_ERROR": "error",
}

_SKIP_TYPES = frozenset({
    "TOOL_CALL_ARGS",
    "TOOL_CALL_END",
    "TEXT_MESSAGE_START",
    "TEXT_MESSAGE_CONTENT",
    "TEXT_MESSAGE_END",
    "STATE_SNAPSHOT",
    "MESSAGES_SNAPSHOT",
})


def _event_type_str(event: BaseEvent) -> Optional[str]:
    raw = getattr(event, "type", None)
    if raw is None:
        return None
    return raw.value if hasattr(raw, "value") else str(raw)


def _build_payload(event_type_str: str, event: BaseEvent) -> str:
    if event_type_str == "TOOL_CALL_START":
        name = getattr(event, "tool_call_name", None) or "unknown"
        tool_id = getattr(event, "tool_call_id", None) or ""
        return json.dumps({"tool": name, "tool_call_id": tool_id})

    if event_type_str == "TOOL_CALL_RESULT":
        tool_id = getattr(event, "tool_call_id", None) or ""
        content = getattr(event, "content", None) or ""
        if len(content) > 2000:
            content = content[:2000] + "... (truncated)"
        return json.dumps({"tool_call_id": tool_id, "result": content})

    if event_type_str == "RUN_ERROR":
        message = getattr(event, "message", None) or ""
        code = getattr(event, "code", None) or ""
        return json.dumps({"error": message, "code": code})

    if event_type_str == "RUN_STARTED":
        return json.dumps({"event": "run_started"})

    if event_type_str == "RUN_FINISHED":
        return json.dumps({"event": "run_finished"})

    if event_type_str == "CUSTOM":
        name = getattr(event, "name", None) or ""
        value = getattr(event, "value", None)
        payload = {"custom_event": name}
        if value is not None:
            try:
                payload["value"] = json.loads(value) if isinstance(value, str) else value
            except (json.JSONDecodeError, TypeError):
                payload["value"] = str(value)
        return json.dumps(payload)

    return json.dumps({"raw_type": event_type_str})


class OperationalEventWriter:
    """Pushes operational events to the session messages API as they happen."""

    def __init__(
        self,
        session_id: str,
        grpc_client: Optional["AmbientGRPCClient"],
    ) -> None:
        self._session_id = session_id
        self._grpc_client = grpc_client

    async def consume(self, event: BaseEvent) -> None:
        event_type_str = _event_type_str(event)
        if event_type_str is None:
            return

        if event_type_str in _SKIP_TYPES:
            return

        mapped_type = _AGUI_TO_EVENT_TYPE.get(event_type_str)

        if event_type_str == "CUSTOM":
            name = getattr(event, "name", None) or ""
            if "error" in name.lower():
                mapped_type = "error"
            else:
                mapped_type = "system"

        if mapped_type is None:
            return

        if self._grpc_client is None:
            return

        payload = _build_payload(event_type_str, event)
        client = self._grpc_client
        session_id = self._session_id

        def _do_push() -> None:
            client.session_messages.push(
                session_id,
                event_type=mapped_type,
                payload=payload,
            )

        try:
            await asyncio.get_running_loop().run_in_executor(None, _do_push)
        except Exception as exc:
            logger.warning(
                "[OPS EVENTS] Push failed: session=%s type=%s error=%s",
                self._session_id,
                mapped_type,
                exc,
            )
