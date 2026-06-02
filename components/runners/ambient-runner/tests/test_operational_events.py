"""Tests for OperationalEventWriter and its payload-building helpers."""

import asyncio
import json
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional

import pytest

from ambient_runner.bridges.claude.operational_events import (
    OperationalEventWriter,
    _build_payload,
    _event_type_str,
)


class FakeEventType(Enum):
    TOOL_CALL_START = "TOOL_CALL_START"
    TOOL_CALL_RESULT = "TOOL_CALL_RESULT"
    TOOL_CALL_ARGS = "TOOL_CALL_ARGS"
    TOOL_CALL_END = "TOOL_CALL_END"
    TEXT_MESSAGE_CONTENT = "TEXT_MESSAGE_CONTENT"
    RUN_STARTED = "RUN_STARTED"
    RUN_FINISHED = "RUN_FINISHED"
    RUN_ERROR = "RUN_ERROR"
    CUSTOM = "CUSTOM"


@dataclass
class FakeEvent:
    type: Optional[FakeEventType] = None
    tool_call_name: str = ""
    tool_call_id: str = ""
    content: str = ""
    message: str = ""
    code: str = ""
    name: str = ""
    value: Optional[str] = None


@dataclass
class FakePushCapture:
    calls: list = field(default_factory=list)

    def push(self, session_id: str, event_type: str, payload: str, **kwargs):
        self.calls.append(
            {"session_id": session_id, "event_type": event_type, "payload": payload}
        )


@dataclass
class FakeGRPCClient:
    session_messages: FakePushCapture = field(default_factory=FakePushCapture)

    def reconnect(self):
        pass


class TestEventTypeStr:
    def test_enum_value(self):
        event = FakeEvent(type=FakeEventType.TOOL_CALL_START)
        assert _event_type_str(event) == "TOOL_CALL_START"

    def test_none_type(self):
        event = FakeEvent(type=None)
        assert _event_type_str(event) is None


class TestBuildPayload:
    def test_tool_call_start(self):
        event = FakeEvent(
            type=FakeEventType.TOOL_CALL_START,
            tool_call_name="Read",
            tool_call_id="tc-123",
        )
        payload = json.loads(_build_payload("TOOL_CALL_START", event))
        assert payload["tool"] == "Read"
        assert payload["tool_call_id"] == "tc-123"

    def test_tool_call_result(self):
        event = FakeEvent(
            type=FakeEventType.TOOL_CALL_RESULT,
            tool_call_id="tc-456",
            content="file contents here",
        )
        payload = json.loads(_build_payload("TOOL_CALL_RESULT", event))
        assert payload["tool_call_id"] == "tc-456"
        assert payload["result"] == "file contents here"

    def test_tool_call_result_truncation(self):
        long_content = "x" * 3000
        event = FakeEvent(
            type=FakeEventType.TOOL_CALL_RESULT,
            tool_call_id="tc-789",
            content=long_content,
        )
        payload = json.loads(_build_payload("TOOL_CALL_RESULT", event))
        assert payload["result"].endswith("... (truncated)")
        assert len(payload["result"]) == 2000 + len("... (truncated)")

    def test_run_error(self):
        event = FakeEvent(
            type=FakeEventType.RUN_ERROR,
            message="tool execution failed",
            code="TOOL_ERROR",
        )
        payload = json.loads(_build_payload("RUN_ERROR", event))
        assert payload["error"] == "tool execution failed"
        assert payload["code"] == "TOOL_ERROR"

    def test_run_started(self):
        event = FakeEvent(type=FakeEventType.RUN_STARTED)
        payload = json.loads(_build_payload("RUN_STARTED", event))
        assert payload["event"] == "run_started"

    def test_run_finished(self):
        event = FakeEvent(type=FakeEventType.RUN_FINISHED)
        payload = json.loads(_build_payload("RUN_FINISHED", event))
        assert payload["event"] == "run_finished"

    def test_custom_event(self):
        event = FakeEvent(
            type=FakeEventType.CUSTOM,
            name="my_custom_event",
            value='{"key": "val"}',
        )
        payload = json.loads(_build_payload("CUSTOM", event))
        assert payload["custom_event"] == "my_custom_event"
        assert payload["value"] == {"key": "val"}

    def test_custom_event_non_json_value(self):
        event = FakeEvent(
            type=FakeEventType.CUSTOM,
            name="plain",
            value="not json",
        )
        payload = json.loads(_build_payload("CUSTOM", event))
        assert payload["value"] == "not json"


class TestOperationalEventWriter:
    @pytest.fixture
    def grpc_client(self):
        return FakeGRPCClient()

    @pytest.fixture
    def writer(self, grpc_client):
        return OperationalEventWriter(
            session_id="sess-001",
            grpc_client=grpc_client,
        )

    def test_tool_call_start_pushes_tool_use(self, writer, grpc_client):
        event = FakeEvent(
            type=FakeEventType.TOOL_CALL_START,
            tool_call_name="Bash",
            tool_call_id="tc-1",
        )
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert len(grpc_client.session_messages.calls) == 1
        call = grpc_client.session_messages.calls[0]
        assert call["session_id"] == "sess-001"
        assert call["event_type"] == "tool_use"

    def test_tool_call_result_pushes_tool_result(self, writer, grpc_client):
        event = FakeEvent(
            type=FakeEventType.TOOL_CALL_RESULT,
            tool_call_id="tc-2",
            content="output",
        )
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert len(grpc_client.session_messages.calls) == 1
        assert grpc_client.session_messages.calls[0]["event_type"] == "tool_result"

    def test_run_error_pushes_error(self, writer, grpc_client):
        event = FakeEvent(
            type=FakeEventType.RUN_ERROR,
            message="crashed",
            code="FATAL",
        )
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert grpc_client.session_messages.calls[0]["event_type"] == "error"

    def test_run_started_pushes_lifecycle(self, writer, grpc_client):
        event = FakeEvent(type=FakeEventType.RUN_STARTED)
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert grpc_client.session_messages.calls[0]["event_type"] == "lifecycle"

    def test_custom_error_pushes_error_type(self, writer, grpc_client):
        event = FakeEvent(
            type=FakeEventType.CUSTOM,
            name="tool_execution_error",
        )
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert grpc_client.session_messages.calls[0]["event_type"] == "error"

    def test_custom_non_error_pushes_system_type(self, writer, grpc_client):
        event = FakeEvent(
            type=FakeEventType.CUSTOM,
            name="status_update",
        )
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert grpc_client.session_messages.calls[0]["event_type"] == "system"

    def test_text_message_content_skipped(self, writer, grpc_client):
        event = FakeEvent(type=FakeEventType.TEXT_MESSAGE_CONTENT)
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert len(grpc_client.session_messages.calls) == 0

    def test_tool_call_args_skipped(self, writer, grpc_client):
        event = FakeEvent(type=FakeEventType.TOOL_CALL_ARGS)
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert len(grpc_client.session_messages.calls) == 0

    def test_none_grpc_client_no_push_no_exception(self):
        writer = OperationalEventWriter(session_id="sess-002", grpc_client=None)
        event = FakeEvent(
            type=FakeEventType.TOOL_CALL_START,
            tool_call_name="Read",
        )
        asyncio.get_event_loop().run_until_complete(writer.consume(event))

    def test_none_event_type_no_push(self, writer, grpc_client):
        event = FakeEvent(type=None)
        asyncio.get_event_loop().run_until_complete(writer.consume(event))
        assert len(grpc_client.session_messages.calls) == 0
