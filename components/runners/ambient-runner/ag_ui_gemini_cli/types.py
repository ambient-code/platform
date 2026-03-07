"""Dataclasses for the 6 Gemini CLI JSONL event types."""

import json
import logging
from dataclasses import dataclass, field
from typing import Optional, Union

logger = logging.getLogger(__name__)

# Type tag used to dispatch parsed events.
_EVENT_TYPES = frozenset(
    {"init", "message", "tool_use", "tool_result", "error", "result"}
)


@dataclass
class InitEvent:
    type: str  # "init"
    timestamp: str
    session_id: str
    model: str


@dataclass
class MessageEvent:
    type: str  # "message"
    timestamp: str
    role: str  # "user" | "assistant"
    content: str
    delta: bool = False


@dataclass
class ToolUseEvent:
    type: str  # "tool_use"
    timestamp: str
    tool_name: str
    tool_id: str
    parameters: dict = field(default_factory=dict)


@dataclass
class ToolResultEvent:
    type: str  # "tool_result"
    timestamp: str
    tool_id: str
    status: str  # "success" | "error"
    output: Optional[str] = None
    error: Optional[dict] = None


@dataclass
class ErrorEvent:
    type: str  # "error"
    timestamp: str
    severity: str  # "warning" | "error"
    message: str


@dataclass
class ResultEvent:
    type: str  # "result"
    timestamp: str
    status: str  # "success" | "error"
    error: Optional[dict] = None
    stats: Optional[dict] = None


_TYPE_MAP = {
    "init": InitEvent,
    "message": MessageEvent,
    "tool_use": ToolUseEvent,
    "tool_result": ToolResultEvent,
    "error": ErrorEvent,
    "result": ResultEvent,
}


def parse_event(
    line: str,
) -> Optional[
    Union[
        InitEvent, MessageEvent, ToolUseEvent, ToolResultEvent, ErrorEvent, ResultEvent
    ]
]:
    """Parse a JSON line into the appropriate event dataclass.

    Returns ``None`` when the line cannot be parsed or has an unknown type.
    """
    try:
        data = json.loads(line)
    except json.JSONDecodeError:
        logger.warning("Failed to parse JSONL line: %s", line[:120])
        return None

    event_type = data.get("type")
    if event_type not in _TYPE_MAP:
        logger.debug("Unknown Gemini CLI event type: %s", event_type)
        return None

    cls = _TYPE_MAP[event_type]
    # Build kwargs matching the dataclass fields
    import dataclasses

    field_names = {f.name for f in dataclasses.fields(cls)}
    kwargs = {k: v for k, v in data.items() if k in field_names}
    try:
        return cls(**kwargs)
    except TypeError as exc:
        logger.warning("Failed to construct %s: %s", cls.__name__, exc)
        return None
