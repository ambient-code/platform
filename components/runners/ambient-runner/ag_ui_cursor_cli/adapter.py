# components/runners/ambient-runner/ag_ui_cursor_cli/adapter.py
"""Cursor CLI adapter for AG-UI protocol.

Translates Cursor CLI stream-json output into AG-UI protocol events.
"""

import logging
import uuid
from typing import AsyncIterator

from ag_ui.core import (
    AssistantMessage as AguiAssistantMessage,
    BaseEvent,
    EventType,
    MessagesSnapshotEvent,
    RunAgentInput,
    RunErrorEvent,
    RunFinishedEvent,
    RunStartedEvent,
    TextMessageContentEvent,
    TextMessageEndEvent,
    TextMessageStartEvent,
    ToolCallArgsEvent,
    ToolCallEndEvent,
    ToolCallResultEvent,
    ToolCallStartEvent as AguiToolCallStartEvent,
)

from .types import (
    InitEvent,
    MessageEvent,
    ResultEvent,
    ToolCallCompletedEvent,
    ToolCallStartEvent,
    parse_event,
)

logger = logging.getLogger(__name__)


class CursorCLIAdapter:
    """Adapter that translates Cursor CLI stream-json to AG-UI events.

    Receives an AsyncIterator[str] of NDJSON lines from the Cursor CLI
    process and yields AG-UI BaseEvent instances.
    """

    async def _flush_text_message(
        self,
        message_id: str,
        accumulated_text: str,
        run_messages: list[AguiAssistantMessage],
    ) -> AsyncIterator[BaseEvent]:
        """Yield TextMessageEndEvent and record the completed message."""
        yield TextMessageEndEvent(
            type=EventType.TEXT_MESSAGE_END,
            message_id=message_id,
        )
        if accumulated_text:
            run_messages.append(
                AguiAssistantMessage(
                    id=message_id,
                    role="assistant",
                    content=accumulated_text,
                )
            )

    async def run(
        self,
        input_data: RunAgentInput,
        *,
        line_stream: AsyncIterator[str],
    ) -> AsyncIterator[BaseEvent]:
        thread_id = input_data.thread_id or str(uuid.uuid4())
        run_id = input_data.run_id or str(uuid.uuid4())

        text_message_open = False
        current_message_id: str | None = None
        accumulated_text = ""
        current_tool_call_id: str | None = None
        run_messages: list[AguiAssistantMessage] = []

        try:
            yield RunStartedEvent(
                type=EventType.RUN_STARTED,
                thread_id=thread_id,
                run_id=run_id,
            )

            async for line in line_stream:
                event = parse_event(line)
                if event is None:
                    logger.debug("Cursor CLI: unparseable line: %s", line[:200])
                    continue

                # ── init ──
                if isinstance(event, InitEvent):
                    logger.debug(
                        "Cursor CLI init: session_id=%s model=%s",
                        event.session_id,
                        event.model,
                    )
                    continue

                # ── assistant message ──
                if isinstance(event, MessageEvent):
                    if event.type == "user":
                        # user events are yielded by Cursor for echo purposes; skip
                        continue

                    if not text_message_open:
                        current_message_id = str(uuid.uuid4())
                        yield TextMessageStartEvent(
                            type=EventType.TEXT_MESSAGE_START,
                            message_id=current_message_id,
                            role="assistant",
                        )
                        text_message_open = True
                        accumulated_text = ""

                    if event.content:
                        accumulated_text += event.content
                        yield TextMessageContentEvent(
                            type=EventType.TEXT_MESSAGE_CONTENT,
                            message_id=current_message_id,
                            delta=event.content,
                        )
                    continue

                # ── tool_call started ──
                if isinstance(event, ToolCallStartEvent):
                    if text_message_open and current_message_id:
                        async for ev in self._flush_text_message(
                            current_message_id, accumulated_text, run_messages
                        ):
                            yield ev
                        text_message_open = False
                        current_message_id = None
                        accumulated_text = ""

                    current_tool_call_id = event.tool_id or str(uuid.uuid4())
                    yield AguiToolCallStartEvent(
                        type=EventType.TOOL_CALL_START,
                        tool_call_id=current_tool_call_id,
                        tool_call_name=event.tool_name,
                    )
                    yield ToolCallArgsEvent(
                        type=EventType.TOOL_CALL_ARGS,
                        tool_call_id=current_tool_call_id,
                        delta=event.arguments or "{}",
                    )
                    continue

                # ── tool_call completed ──
                if isinstance(event, ToolCallCompletedEvent):
                    tid = event.tool_id or current_tool_call_id
                    if tid:
                        yield ToolCallEndEvent(
                            type=EventType.TOOL_CALL_END,
                            tool_call_id=tid,
                        )
                        result_content = (
                            event.error
                            if event.error
                            else (event.output or "(completed)")
                        )
                        yield ToolCallResultEvent(
                            type=EventType.TOOL_CALL_RESULT,
                            tool_call_id=tid,
                            message_id=f"{tid}-result",
                            role="tool",
                            content=result_content,
                        )
                    current_tool_call_id = None
                    continue

                # ── result ──
                if isinstance(event, ResultEvent):
                    if text_message_open and current_message_id:
                        async for ev in self._flush_text_message(
                            current_message_id, accumulated_text, run_messages
                        ):
                            yield ev
                        text_message_open = False
                        current_message_id = None
                        accumulated_text = ""

                    if event.is_error or event.subtype == "error":
                        yield RunErrorEvent(
                            type=EventType.RUN_ERROR,
                            thread_id=thread_id,
                            run_id=run_id,
                            message=event.result or "Cursor CLI run failed",
                        )
                        return

                    break

            all_messages = list(input_data.messages or []) + run_messages
            if all_messages:
                yield MessagesSnapshotEvent(
                    type=EventType.MESSAGES_SNAPSHOT,
                    messages=all_messages,
                )

            yield RunFinishedEvent(
                type=EventType.RUN_FINISHED,
                thread_id=thread_id,
                run_id=run_id,
            )

        except Exception as exc:
            logger.error("Error in Cursor CLI adapter run: %s", exc)
            if text_message_open and current_message_id:
                try:
                    yield TextMessageEndEvent(
                        type=EventType.TEXT_MESSAGE_END,
                        message_id=current_message_id,
                    )
                except Exception:
                    pass
                text_message_open = False

            yield RunErrorEvent(
                type=EventType.RUN_ERROR,
                thread_id=thread_id,
                run_id=run_id,
                message=str(exc),
            )
        finally:
            # Handles asyncio.CancelledError (BaseException, not caught by `except
            # Exception`) — ensures any open text message is always closed on
            # generator cleanup.
            if text_message_open and current_message_id:
                try:
                    yield TextMessageEndEvent(
                        type=EventType.TEXT_MESSAGE_END,
                        message_id=current_message_id,
                    )
                except Exception:
                    pass
