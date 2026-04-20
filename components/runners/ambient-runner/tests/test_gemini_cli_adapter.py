"""Smoke tests for the Gemini CLI adapter — event parsing and translation."""

import json

import pytest

from ag_ui_gemini_cli.types import (
    ErrorEvent,
    InitEvent,
    MessageEvent,
    ResultEvent,
    ThinkingEvent,
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

    def test_thinking_event(self):
        line = json.dumps(
            {
                "type": "thinking",
                "timestamp": "2025-01-01T00:00:01Z",
                "content": "Let me reason about this...",
                "delta": True,
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, ThinkingEvent)
        assert evt.content == "Let me reason about this..."
        assert evt.delta is True

    def test_thinking_event_non_delta(self):
        line = json.dumps(
            {
                "type": "thinking",
                "timestamp": "2025-01-01T00:00:01Z",
                "content": "Full thought.",
            }
        )
        evt = parse_event(line)
        assert isinstance(evt, ThinkingEvent)
        assert evt.content == "Full thought."
        assert evt.delta is False

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
            json.dumps(
                {
                    "type": "init",
                    "timestamp": "T",
                    "session_id": "s1",
                    "model": "gemini-2.5-flash",
                }
            ),
            json.dumps(
                {"type": "message", "timestamp": "T", "role": "user", "content": "hi"}
            ),
            json.dumps(
                {
                    "type": "message",
                    "timestamp": "T",
                    "role": "assistant",
                    "content": "Hello!",
                    "delta": True,
                }
            ),
            json.dumps(
                {
                    "type": "result",
                    "timestamp": "T",
                    "status": "success",
                    "stats": {
                        "total_tokens": 10,
                        "input_tokens": 5,
                        "output_tokens": 5,
                        "duration_ms": 100,
                        "tool_calls": 0,
                    },
                }
            ),
        ]

        async def line_stream():
            for line in lines:
                yield line

        input_data = RunAgentInput(
            thread_id="t1",
            run_id="r1",
            state={},
            messages=[],
            tools=[],
            context=[],
            forwardedProps={},
        )
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
            json.dumps(
                {"type": "init", "timestamp": "T", "session_id": "s1", "model": "m"}
            ),
            json.dumps(
                {
                    "type": "tool_use",
                    "timestamp": "T",
                    "tool_name": "read_file",
                    "tool_id": "t1",
                    "parameters": {"path": "a.py"},
                }
            ),
            json.dumps(
                {
                    "type": "tool_result",
                    "timestamp": "T",
                    "tool_id": "t1",
                    "status": "success",
                    "output": "data",
                }
            ),
            json.dumps(
                {
                    "type": "message",
                    "timestamp": "T",
                    "role": "assistant",
                    "content": "Done",
                    "delta": True,
                }
            ),
            json.dumps({"type": "result", "timestamp": "T", "status": "success"}),
        ]

        async def line_stream():
            for line in lines:
                yield line

        input_data = RunAgentInput(
            thread_id="t1",
            run_id="r1",
            state={},
            messages=[],
            tools=[],
            context=[],
            forwardedProps={},
        )
        adapter = GeminiCLIAdapter()
        events = []
        async for event in adapter.run(input_data, line_stream=line_stream()):
            events.append(event)

        types = [e.type for e in events]
        assert "TOOL_CALL_START" in types
        assert "TOOL_CALL_ARGS" in types
        assert "TOOL_CALL_END" in types

    @pytest.mark.asyncio
    async def test_thinking_then_text_response(self):
        """thinking + assistant message → REASONING events + TEXT events."""
        from ag_ui_gemini_cli.adapter import GeminiCLIAdapter
        from ag_ui.core import RunAgentInput

        lines = [
            json.dumps(
                {
                    "type": "init",
                    "timestamp": "T",
                    "session_id": "s1",
                    "model": "gemini-2.5-pro",
                }
            ),
            json.dumps(
                {
                    "type": "thinking",
                    "timestamp": "T",
                    "content": "Let me think about this...",
                    "delta": True,
                }
            ),
            json.dumps(
                {
                    "type": "thinking",
                    "timestamp": "T",
                    "content": " I should consider X.",
                    "delta": True,
                }
            ),
            json.dumps(
                {
                    "type": "message",
                    "timestamp": "T",
                    "role": "assistant",
                    "content": "Here is my answer.",
                    "delta": True,
                }
            ),
            json.dumps(
                {
                    "type": "result",
                    "timestamp": "T",
                    "status": "success",
                    "stats": {"total_tokens": 20},
                }
            ),
        ]

        async def line_stream():
            for line in lines:
                yield line

        input_data = RunAgentInput(
            thread_id="t1",
            run_id="r1",
            state={},
            messages=[],
            tools=[],
            context=[],
            forwardedProps={},
        )
        adapter = GeminiCLIAdapter()
        events = []
        async for event in adapter.run(input_data, line_stream=line_stream()):
            events.append(event)

        types = [e.type if isinstance(e.type, str) else e.type for e in events]
        assert "RUN_STARTED" in types
        assert "REASONING_START" in types
        assert "REASONING_MESSAGE_START" in types
        assert "REASONING_MESSAGE_CONTENT" in types
        assert "REASONING_MESSAGE_END" in types
        assert "REASONING_END" in types
        assert "TEXT_MESSAGE_START" in types
        assert "TEXT_MESSAGE_CONTENT" in types
        assert "RUN_FINISHED" in types

        # Reasoning events should come before text events
        reasoning_start_idx = types.index("REASONING_START")
        reasoning_end_idx = types.index("REASONING_END")
        text_start_idx = types.index("TEXT_MESSAGE_START")
        assert reasoning_start_idx < reasoning_end_idx < text_start_idx

        # Should have two REASONING_MESSAGE_CONTENT events (two delta chunks)
        reasoning_content_events = [
            e for e in events if getattr(e, "type", None) == "REASONING_MESSAGE_CONTENT"
        ]
        assert len(reasoning_content_events) == 2
        assert reasoning_content_events[0].delta == "Let me think about this..."
        assert reasoning_content_events[1].delta == " I should consider X."

    @pytest.mark.asyncio
    async def test_non_delta_thinking(self):
        """Non-delta thinking event opens and closes reasoning block immediately."""
        from ag_ui_gemini_cli.adapter import GeminiCLIAdapter
        from ag_ui.core import RunAgentInput

        lines = [
            json.dumps(
                {
                    "type": "init",
                    "timestamp": "T",
                    "session_id": "s1",
                    "model": "gemini-2.5-pro",
                }
            ),
            json.dumps(
                {
                    "type": "thinking",
                    "timestamp": "T",
                    "content": "Full reasoning block.",
                }
            ),
            json.dumps(
                {
                    "type": "message",
                    "timestamp": "T",
                    "role": "assistant",
                    "content": "Answer.",
                    "delta": True,
                }
            ),
            json.dumps({"type": "result", "timestamp": "T", "status": "success"}),
        ]

        async def line_stream():
            for line in lines:
                yield line

        input_data = RunAgentInput(
            thread_id="t1",
            run_id="r1",
            state={},
            messages=[],
            tools=[],
            context=[],
            forwardedProps={},
        )
        adapter = GeminiCLIAdapter()
        events = []
        async for event in adapter.run(input_data, line_stream=line_stream()):
            events.append(event)

        types = [e.type if isinstance(e.type, str) else e.type for e in events]
        # Reasoning should be fully closed before text starts
        assert "REASONING_START" in types
        assert "REASONING_MESSAGE_START" in types
        assert "REASONING_MESSAGE_CONTENT" in types
        assert "REASONING_MESSAGE_END" in types
        assert "REASONING_END" in types
        assert "TEXT_MESSAGE_START" in types

        # Non-delta: reasoning block closed immediately (not by the message handler)
        reasoning_end_idx = types.index("REASONING_END")
        text_start_idx = types.index("TEXT_MESSAGE_START")
        assert reasoning_end_idx < text_start_idx

    @pytest.mark.asyncio
    async def test_thinking_before_tool_call(self):
        """Reasoning block is closed before tool call events are emitted."""
        from ag_ui_gemini_cli.adapter import GeminiCLIAdapter
        from ag_ui.core import RunAgentInput

        lines = [
            json.dumps(
                {
                    "type": "init",
                    "timestamp": "T",
                    "session_id": "s1",
                    "model": "gemini-2.5-pro",
                }
            ),
            json.dumps(
                {
                    "type": "thinking",
                    "timestamp": "T",
                    "content": "I need to read a file.",
                    "delta": True,
                }
            ),
            json.dumps(
                {
                    "type": "tool_use",
                    "timestamp": "T",
                    "tool_name": "read_file",
                    "tool_id": "t1",
                    "parameters": {"path": "a.py"},
                }
            ),
            json.dumps(
                {
                    "type": "tool_result",
                    "timestamp": "T",
                    "tool_id": "t1",
                    "status": "success",
                    "output": "data",
                }
            ),
            json.dumps({"type": "result", "timestamp": "T", "status": "success"}),
        ]

        async def line_stream():
            for line in lines:
                yield line

        input_data = RunAgentInput(
            thread_id="t1",
            run_id="r1",
            state={},
            messages=[],
            tools=[],
            context=[],
            forwardedProps={},
        )
        adapter = GeminiCLIAdapter()
        events = []
        async for event in adapter.run(input_data, line_stream=line_stream()):
            events.append(event)

        types = [e.type if isinstance(e.type, str) else e.type for e in events]
        assert "REASONING_END" in types
        assert "TOOL_CALL_START" in types

        # Reasoning must be closed before tool call starts
        reasoning_end_idx = types.index("REASONING_END")
        tool_start_idx = types.index("TOOL_CALL_START")
        assert reasoning_end_idx < tool_start_idx
