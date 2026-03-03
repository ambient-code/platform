"""Smoke tests for the Gemini CLI adapter — event parsing and translation."""

import json

import pytest

from ag_ui_gemini_cli.types import (
    ErrorEvent,
    InitEvent,
    MessageEvent,
    ResultEvent,
    ToolResultEvent,
    ToolUseEvent,
    parse_event,
)


class TestParseEvent:
    """Verify NDJSON lines are parsed into the correct dataclass."""

    def test_init_event(self):
        line = json.dumps(
            {
                "type": "init",
                "timestamp": "2025-01-01T00:00:00Z",
                "session_id": "sess-123",
                "model": "gemini-2.5-flash",
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, InitEvent)
        assert evt.session_id == "sess-123"
        assert evt.model == "gemini-2.5-flash"

    def test_message_event_assistant_delta(self):
        line = json.dumps(
            {
                "type": "message",
                "timestamp": "2025-01-01T00:00:01Z",
                "role": "assistant",
                "content": "Hello!",
                "delta": True,
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, MessageEvent)
        assert evt.role == "assistant"
        assert evt.content == "Hello!"
        assert evt.delta is True

    def test_message_event_user(self):
        line = json.dumps(
            {
                "type": "message",
                "timestamp": "2025-01-01T00:00:00Z",
                "role": "user",
                "content": "What is 2+2?",
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, MessageEvent)
        assert evt.role == "user"
        assert evt.delta is False

    def test_tool_use_event(self):
        line = json.dumps(
            {
                "type": "tool_use",
                "timestamp": "2025-01-01T00:00:02Z",
                "tool_name": "read_file",
                "tool_id": "read_1",
                "parameters": {"file_path": "main.py"},
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, ToolUseEvent)
        assert evt.tool_name == "read_file"
        assert evt.tool_id == "read_1"
        assert evt.parameters == {"file_path": "main.py"}

    def test_tool_result_success(self):
        line = json.dumps(
            {
                "type": "tool_result",
                "timestamp": "2025-01-01T00:00:03Z",
                "tool_id": "read_1",
                "status": "success",
                "output": "file contents",
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, ToolResultEvent)
        assert evt.status == "success"
        assert evt.output == "file contents"

    def test_tool_result_error(self):
        line = json.dumps(
            {
                "type": "tool_result",
                "timestamp": "2025-01-01T00:00:03Z",
                "tool_id": "read_1",
                "status": "error",
                "error": {"type": "file_not_found", "message": "Not found"},
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, ToolResultEvent)
        assert evt.status == "error"
        assert evt.error["type"] == "file_not_found"

    def test_error_event(self):
        line = json.dumps(
            {
                "type": "error",
                "timestamp": "2025-01-01T00:00:04Z",
                "severity": "warning",
                "message": "Loop detected",
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, ErrorEvent)
        assert evt.severity == "warning"

    def test_result_event_success(self):
        line = json.dumps(
            {
                "type": "result",
                "timestamp": "2025-01-01T00:00:05Z",
                "status": "success",
                "stats": {
                    "total_tokens": 100,
                    "input_tokens": 50,
                    "output_tokens": 50,
                    "duration_ms": 1200,
                    "tool_calls": 2,
                },
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, ResultEvent)
        assert evt.status == "success"
        assert evt.stats["total_tokens"] == 100

    def test_result_event_error(self):
        line = json.dumps(
            {
                "type": "result",
                "timestamp": "2025-01-01T00:00:05Z",
                "status": "error",
                "error": {
                    "type": "FatalAuthenticationError",
                    "message": "Auth failed",
                },
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, ResultEvent)
        assert evt.status == "error"
        assert evt.error["type"] == "FatalAuthenticationError"

    def test_invalid_json_returns_none(self):
        evt = parse_event("not valid json")
        assert evt is None

    def test_unknown_type_returns_none(self):
        line = json.dumps({"type": "unknown_type", "timestamp": "2025-01-01T00:00:00Z"})
        evt = parse_event(line)
        assert evt is None


class TestGeminiCLIAdapter:
    """Verify the adapter translates events to AG-UI correctly."""

    @pytest.mark.asyncio
    async def test_simple_text_response(self):
        """init + assistant message + result → RUN_STARTED + TEXT events + RUN_FINISHED."""
        from ag_ui_gemini_cli.adapter import GeminiCLIAdapter
        from ag_ui.core import RunAgentInput

        lines = [
            json.dumps({"type": "init", "timestamp": "T", "session_id": "s1", "model": "gemini-2.5-flash"}),
            json.dumps({"type": "message", "timestamp": "T", "role": "user", "content": "hi"}),
            json.dumps({"type": "message", "timestamp": "T", "role": "assistant", "content": "Hello!", "delta": True}),
            json.dumps({"type": "result", "timestamp": "T", "status": "success", "stats": {"total_tokens": 10, "input_tokens": 5, "output_tokens": 5, "duration_ms": 100, "tool_calls": 0}}),
        ]

        async def line_stream():
            for line in lines:
                yield line

        input_data = RunAgentInput(thread_id="t1", run_id="r1", state={}, messages=[], tools=[], context=[], forwardedProps={})
        adapter = GeminiCLIAdapter()
        events = []
        async for event in adapter.run(input_data, line_stream=line_stream()):
            events.append(event)

        types = [e.type for e in events]
        assert "RUN_STARTED" in types
        assert "TEXT_MESSAGE_START" in types
        assert "TEXT_MESSAGE_CONTENT" in types
        assert "TEXT_MESSAGE_END" in types
        assert "RUN_FINISHED" in types

    @pytest.mark.asyncio
    async def test_tool_call_flow(self):
        """tool_use + tool_result → TOOL_CALL events."""
        from ag_ui_gemini_cli.adapter import GeminiCLIAdapter
        from ag_ui.core import RunAgentInput

        lines = [
            json.dumps({"type": "init", "timestamp": "T", "session_id": "s1", "model": "m"}),
            json.dumps({"type": "tool_use", "timestamp": "T", "tool_name": "read_file", "tool_id": "t1", "parameters": {"path": "a.py"}}),
            json.dumps({"type": "tool_result", "timestamp": "T", "tool_id": "t1", "status": "success", "output": "data"}),
            json.dumps({"type": "message", "timestamp": "T", "role": "assistant", "content": "Done", "delta": True}),
            json.dumps({"type": "result", "timestamp": "T", "status": "success"}),
        ]

        async def line_stream():
            for line in lines:
                yield line

        input_data = RunAgentInput(thread_id="t1", run_id="r1", state={}, messages=[], tools=[], context=[], forwardedProps={})
        adapter = GeminiCLIAdapter()
        events = []
        async for event in adapter.run(input_data, line_stream=line_stream()):
            events.append(event)

        types = [e.type for e in events]
        assert "TOOL_CALL_START" in types
        assert "TOOL_CALL_ARGS" in types
        assert "TOOL_CALL_END" in types
