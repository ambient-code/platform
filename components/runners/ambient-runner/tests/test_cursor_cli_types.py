"""Tests for Cursor CLI stream-json event parsing against real binary output."""

from ag_ui_cursor_cli.types import (
    InitEvent,
    MessageEvent,
    ToolCallStartEvent,
    ToolCallCompletedEvent,
    ResultEvent,
    parse_event,
)

REAL_INIT = '{"type":"system","subtype":"init","apiKeySource":"login","cwd":"/tmp","session_id":"ae85-cc7a","model":"Composer 2 Fast","permissionMode":"default"}'
REAL_ASSISTANT = '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]},"session_id":"ae85"}'
REAL_USER = '{"type":"user","message":{"role":"user","content":[{"type":"text","text":"read test.txt"}]},"session_id":"ae85"}'
REAL_TOOL_START = '{"type":"tool_call","subtype":"started","call_id":"tool_584d","tool_call":{"readToolCall":{"args":{"path":"/tmp/test.txt"}}},"model_call_id":"7d78","session_id":"ae85","timestamp_ms":1776794504467}'
REAL_TOOL_DONE = '{"type":"tool_call","subtype":"completed","call_id":"tool_584d","tool_call":{"readToolCall":{"args":{"path":"/tmp/test.txt"},"result":{"success":{"content":"placeholder\\n"}}}},"model_call_id":"7d78","session_id":"ae85","timestamp_ms":1776794504551}'
REAL_RESULT = '{"type":"result","subtype":"success","duration_ms":6155,"duration_api_ms":6155,"is_error":false,"result":"hello world","session_id":"ae85","request_id":"db94","usage":{"inputTokens":19483,"outputTokens":105}}'


def test_parse_init_event():
    ev = parse_event(REAL_INIT)
    assert isinstance(ev, InitEvent)
    assert ev.session_id == "ae85-cc7a"
    assert ev.model == "Composer 2 Fast"
    assert ev.cwd == "/tmp"
    assert ev.permission_mode == "default"


def test_parse_assistant_message():
    ev = parse_event(REAL_ASSISTANT)
    assert isinstance(ev, MessageEvent)
    assert ev.type == "assistant"
    assert ev.content == "hello"


def test_parse_user_message():
    ev = parse_event(REAL_USER)
    assert isinstance(ev, MessageEvent)
    assert ev.type == "user"
    assert ev.content == "read test.txt"


def test_parse_tool_call_start():
    ev = parse_event(REAL_TOOL_START)
    assert isinstance(ev, ToolCallStartEvent)
    assert ev.tool_id == "tool_584d"
    assert ev.tool_name == "readToolCall"
    assert '"path"' in ev.arguments


def test_parse_tool_call_completed():
    ev = parse_event(REAL_TOOL_DONE)
    assert isinstance(ev, ToolCallCompletedEvent)
    assert ev.tool_id == "tool_584d"
    assert "placeholder" in ev.output
    assert ev.error == ""


def test_parse_result_event():
    ev = parse_event(REAL_RESULT)
    assert isinstance(ev, ResultEvent)
    assert ev.result == "hello world"
    assert ev.duration_ms == 6155
    assert ev.is_error is False
    assert ev.usage["inputTokens"] == 19483


def test_parse_garbage_returns_none():
    assert parse_event("not json") is None
    assert parse_event('{"type":"unknown_thing"}') is None
