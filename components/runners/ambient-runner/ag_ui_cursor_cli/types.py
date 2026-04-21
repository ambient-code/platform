"""Dataclasses for Cursor CLI stream-json event types."""

import json
import logging
from dataclasses import dataclass, field

logger = logging.getLogger(__name__)


@dataclass
class InitEvent:
    type: str  # "system"
    subtype: str  # "init"
    session_id: str = ""
    model: str = ""
    cwd: str = ""
    permission_mode: str = ""


@dataclass
class MessageEvent:
    type: str  # "assistant" or "user"
    content: str = ""


@dataclass
class ToolCallStartEvent:
    type: str  # "tool_call"
    subtype: str  # "started"
    tool_id: str = ""
    tool_name: str = ""
    arguments: str = ""


@dataclass
class ToolCallCompletedEvent:
    type: str  # "tool_call"
    subtype: str  # "completed"
    tool_id: str = ""
    output: str = ""
    error: str = ""


@dataclass
class ResultEvent:
    type: str  # "result"
    subtype: str  # "success" or "error"
    result: str = ""
    session_id: str = ""
    duration_ms: int = 0
    is_error: bool = False
    usage: dict = field(default_factory=dict)


def _extract_content_text(message: dict) -> str:
    """Extract text from message.content which is [{type, text}, ...]."""
    content = message.get("content", [])
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "".join(
            block.get("text", "")
            for block in content
            if isinstance(block, dict) and block.get("type") == "text"
        )
    return ""


def _extract_tool_info(tool_call: dict) -> tuple[str, str]:
    """Extract (tool_name, args_json) from tool-type-keyed tool_call dict.

    Real format: {"readToolCall": {"args": {...}}}
    """
    for key, value in tool_call.items():
        if isinstance(value, dict) and "args" in value:
            return key, json.dumps(value["args"])
    for key in tool_call:
        return key, "{}"
    return "unknown", "{}"


def _extract_tool_result(tool_call: dict) -> tuple[str, str]:
    """Extract (output, error) from completed tool_call.

    Real format: {"readToolCall": {"args": ..., "result": {"success": {"content": "..."}}}}
    """
    for key, value in tool_call.items():
        if not isinstance(value, dict):
            continue
        result = value.get("result", {})
        if not isinstance(result, dict):
            continue
        success = result.get("success", {})
        if isinstance(success, dict) and "content" in success:
            return str(success["content"]), ""
        error = result.get("error", {})
        if error:
            return "", json.dumps(error) if isinstance(error, dict) else str(error)
        return json.dumps(result), ""
    return "(completed)", ""


def parse_event(
    line: str,
) -> (
    InitEvent
    | MessageEvent
    | ToolCallStartEvent
    | ToolCallCompletedEvent
    | ResultEvent
    | None
):
    """Parse a JSON line from cursor-agent --output-format stream-json."""
    try:
        data = json.loads(line)
    except json.JSONDecodeError:
        logger.warning("Failed to parse JSONL line: %s", line[:120])
        return None

    event_type = data.get("type")
    subtype = data.get("subtype", "")

    if event_type == "system" and subtype == "init":
        return InitEvent(
            type=event_type,
            subtype=subtype,
            session_id=data.get("session_id", ""),
            model=data.get("model", ""),
            cwd=data.get("cwd", ""),
            permission_mode=data.get("permissionMode", ""),
        )

    if event_type in ("assistant", "user"):
        msg = data.get("message", {})
        content = _extract_content_text(msg) if isinstance(msg, dict) else ""
        return MessageEvent(
            type=event_type,
            content=content,
        )

    if event_type == "tool_call" and subtype == "started":
        tc = data.get("tool_call", {})
        tool_name, arguments = _extract_tool_info(tc)
        return ToolCallStartEvent(
            type=event_type,
            subtype=subtype,
            tool_id=data.get("call_id", ""),
            tool_name=tool_name,
            arguments=arguments,
        )

    if event_type == "tool_call" and subtype == "completed":
        tc = data.get("tool_call", {})
        output, error = _extract_tool_result(tc)
        return ToolCallCompletedEvent(
            type=event_type,
            subtype=subtype,
            tool_id=data.get("call_id", ""),
            output=output,
            error=error,
        )

    if event_type == "result":
        return ResultEvent(
            type=event_type,
            subtype=subtype,
            result=data.get("result", ""),
            session_id=data.get("session_id", ""),
            duration_ms=data.get("duration_ms", 0),
            is_error=data.get("is_error", False),
            usage=data.get("usage", {}),
        )

    logger.debug("Unknown Cursor CLI event: type=%s subtype=%s", event_type, subtype)
    return None
