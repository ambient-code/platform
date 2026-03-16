"""Gemini CLI adapter for AG-UI protocol.

Translates Gemini CLI JSONL (stream-json) output into AG-UI protocol events,
enabling Gemini-powered agents to work with any AG-UI compatible frontend.
"""

import json
import logging
import uuid
from datetime import datetime
from typing import AsyncIterator, Optional

from ag_ui.core import (
    EventType,
    RunAgentInput,
    BaseEvent,
    AssistantMessage as AguiAssistantMessage,
    RunStartedEvent,
    RunFinishedEvent,
    RunErrorEvent,
    TextMessageStartEvent,
    TextMessageContentEvent,
    TextMessageEndEvent,
    ToolCallStartEvent,
    ToolCallArgsEvent,
    ToolCallEndEvent,
    ToolCallResultEvent,
    MessagesSnapshotEvent,
)

from .types import (
    InitEvent,
    MessageEvent,
    ToolUseEvent,
    ToolResultEvent,
    ErrorEvent,
    ResultEvent,
    parse_event,
)

logger = logging.getLogger(__name__)


def _iso_to_ms(iso_timestamp: str) -> Optional[int]:
    """Convert an ISO 8601 timestamp string to epoch milliseconds."""
    try:
        dt = datetime.fromisoformat(iso_timestamp.replace("Z", "+00:00"))
        return int(dt.timestamp() * 1000)
    except (ValueError, AttributeError):
        return None


def _summarize_event(event: object) -> str:
    """One-line summary of a Gemini CLI event for logging."""
    if isinstance(event, InitEvent):
        return f"session_id={event.session_id} model={event.model}"
    if isinstance(event, MessageEvent):
        preview = (event.content or "")[:80]
        return f"role={event.role} delta={event.delta} content={preview!r}"
    if isinstance(event, ToolUseEvent):
        return f"tool={event.tool_name} id={event.tool_id}"
    if isinstance(event, ToolResultEvent):
        return f"id={event.tool_id} status={event.status}"
    if isinstance(event, ErrorEvent):
        return f"severity={event.severity} msg={event.message[:80]}"
    if isinstance(event, ResultEvent):
        return f"status={event.status} stats={event.stats}"
    return ""


class GeminiCLIAdapter:
    """Adapter that translates Gemini CLI JSONL output to AG-UI events.

    Receives an ``AsyncIterator[str]`` of NDJSON lines from the Gemini CLI
    process (managed externally by the bridge/session layer) and yields
    AG-UI ``BaseEvent`` instances.
    """

    async def run(
        self,
        input_data: RunAgentInput,
        *,
        line_stream: AsyncIterator[str],
    ) -> AsyncIterator[BaseEvent]:
        """Process a Gemini CLI run and yield AG-UI events.

        Args:
            input_data: AG-UI run input.
            line_stream: Async iterator of NDJSON lines from the Gemini CLI.

        Yields:
            AG-UI BaseEvent instances.
        """
        thread_id = input_data.thread_id or str(uuid.uuid4())
        run_id = input_data.run_id or str(uuid.uuid4())

        # Per-run streaming state
        text_message_open = False
        current_message_id: Optional[str] = None
        accumulated_text = ""
        message_timestamp_ms: Optional[int] = None

        # Tool tracking
        current_tool_call_id: Optional[str] = None

        # Metadata captured from init event
        session_id: Optional[str] = None
        model: Optional[str] = None

        # Accumulated messages for MESSAGES_SNAPSHOT
        run_messages: list[AguiAssistantMessage] = []

        try:
            # 1. Emit RUN_STARTED
            yield RunStartedEvent(
                type=EventType.RUN_STARTED,
                thread_id=thread_id,
                run_id=run_id,
            )

            # 2. Process JSONL lines
            async for line in line_stream:
                event = parse_event(line)
                if event is None:
                    logger.debug("Gemini CLI: unparseable line: %s", line[:200])
                    continue

                logger.info(
                    "Gemini CLI event: type=%s %s",
                    type(event).__name__,
                    _summarize_event(event),
                )

                # ── init ──
                if isinstance(event, InitEvent):
                    session_id = event.session_id
                    model = event.model
                    logger.debug(
                        "Gemini CLI init: session_id=%s model=%s",
                        session_id,
                        model,
                    )
                    continue

                # ── message (assistant, delta) ──
                if isinstance(event, MessageEvent):
                    # Skip user messages (already in input)
                    if event.role == "user":
                        continue

                    if event.role == "assistant" and event.delta:
                        # First text chunk: open a text message
                        if not text_message_open:
                            current_message_id = str(uuid.uuid4())
                            message_timestamp_ms = _iso_to_ms(event.timestamp)
                            yield TextMessageStartEvent(
                                type=EventType.TEXT_MESSAGE_START,
                                message_id=current_message_id,
                                role="assistant",
                                timestamp=message_timestamp_ms,
                            )
                            text_message_open = True
                            accumulated_text = ""

                        # Stream text content
                        if event.content:
                            accumulated_text += event.content
                            yield TextMessageContentEvent(
                                type=EventType.TEXT_MESSAGE_CONTENT,
                                message_id=current_message_id,
                                delta=event.content,
                            )
                        continue

                    # Non-delta assistant message (full message)
                    if event.role == "assistant" and not event.delta:
                        if not text_message_open:
                            current_message_id = str(uuid.uuid4())
                            message_timestamp_ms = _iso_to_ms(event.timestamp)
                            yield TextMessageStartEvent(
                                type=EventType.TEXT_MESSAGE_START,
                                message_id=current_message_id,
                                role="assistant",
                                timestamp=message_timestamp_ms,
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

                        # Close the message immediately for non-delta
                        yield TextMessageEndEvent(
                            type=EventType.TEXT_MESSAGE_END,
                            message_id=current_message_id,
                        )
                        text_message_open = False

                        # Accumulate for snapshot
                        if accumulated_text:
                            run_messages.append(
                                AguiAssistantMessage(
                                    id=current_message_id,
                                    role="assistant",
                                    content=accumulated_text,
                                    timestamp=message_timestamp_ms,
                                )
                            )
                        current_message_id = None
                        accumulated_text = ""
                        message_timestamp_ms = None
                        continue

                # ── tool_use ──
                if isinstance(event, ToolUseEvent):
                    # Close any open text message before tool call
                    if text_message_open and current_message_id:
                        yield TextMessageEndEvent(
                            type=EventType.TEXT_MESSAGE_END,
                            message_id=current_message_id,
                        )
                        if accumulated_text:
                            run_messages.append(
                                AguiAssistantMessage(
                                    id=current_message_id,
                                    role="assistant",
                                    content=accumulated_text,
                                    timestamp=message_timestamp_ms,
                                )
                            )
                        text_message_open = False
                        current_message_id = None
                        accumulated_text = ""
                        message_timestamp_ms = None

                    current_tool_call_id = event.tool_id or str(uuid.uuid4())
                    yield ToolCallStartEvent(
                        type=EventType.TOOL_CALL_START,
                        tool_call_id=current_tool_call_id,
                        tool_call_name=event.tool_name,
                    )

                    # Emit arguments
                    args_json = (
                        json.dumps(event.parameters) if event.parameters else "{}"
                    )
                    yield ToolCallArgsEvent(
                        type=EventType.TOOL_CALL_ARGS,
                        tool_call_id=current_tool_call_id,
                        delta=args_json,
                    )
                    continue

                # ── tool_result ──
                if isinstance(event, ToolResultEvent):
                    tid = event.tool_id or current_tool_call_id
                    if tid:
                        yield ToolCallEndEvent(
                            type=EventType.TOOL_CALL_END,
                            tool_call_id=tid,
                        )
                        # Emit the tool result so the frontend can display it
                        if event.status == "error" and event.error:
                            result_content = json.dumps(event.error)
                        else:
                            result_content = event.output or "(completed)"
                        yield ToolCallResultEvent(
                            type=EventType.TOOL_CALL_RESULT,
                            tool_call_id=tid,
                            message_id=f"{tid}-result",
                            role="tool",
                            content=result_content,
                        )
                    current_tool_call_id = None
                    continue

                # ── error ──
                if isinstance(event, ErrorEvent):
                    if event.severity == "error":
                        # Close open text message
                        if text_message_open and current_message_id:
                            yield TextMessageEndEvent(
                                type=EventType.TEXT_MESSAGE_END,
                                message_id=current_message_id,
                            )
                            text_message_open = False

                        yield RunErrorEvent(
                            type=EventType.RUN_ERROR,
                            thread_id=thread_id,
                            run_id=run_id,
                            message=event.message,
                        )
                        return
                    else:
                        # Warning: log only
                        logger.warning("Gemini CLI warning: %s", event.message)
                    continue

                # ── result ──
                if isinstance(event, ResultEvent):
                    # Close open text message
                    if text_message_open and current_message_id:
                        yield TextMessageEndEvent(
                            type=EventType.TEXT_MESSAGE_END,
                            message_id=current_message_id,
                        )
                        if accumulated_text:
                            run_messages.append(
                                AguiAssistantMessage(
                                    id=current_message_id,
                                    role="assistant",
                                    content=accumulated_text,
                                    timestamp=message_timestamp_ms,
                                )
                            )
                        text_message_open = False
                        current_message_id = None
                        accumulated_text = ""
                        message_timestamp_ms = None

                    if event.status == "error":
                        error_msg = "Gemini CLI run failed"
                        if event.error:
                            error_msg = event.error.get("message", error_msg)
                        yield RunErrorEvent(
                            type=EventType.RUN_ERROR,
                            thread_id=thread_id,
                            run_id=run_id,
                            message=error_msg,
                        )
                        return

                    # Success: emit snapshot and finish
                    break

            # 3. Emit MESSAGES_SNAPSHOT with accumulated messages
            all_messages = list(input_data.messages or []) + run_messages
            if all_messages:
                yield MessagesSnapshotEvent(
                    type=EventType.MESSAGES_SNAPSHOT,
                    messages=all_messages,
                )

            # 4. Emit RUN_FINISHED
            yield RunFinishedEvent(
                type=EventType.RUN_FINISHED,
                thread_id=thread_id,
                run_id=run_id,
            )

        except Exception as exc:
            logger.error("Error in Gemini CLI adapter run: %s", exc)
            # Clean up open text message
            if text_message_open and current_message_id:
                try:
                    yield TextMessageEndEvent(
                        type=EventType.TEXT_MESSAGE_END,
                        message_id=current_message_id,
                    )
                except Exception:
                    pass
                text_message_open = False  # Prevent duplicate in finally

            yield RunErrorEvent(
                type=EventType.RUN_ERROR,
                thread_id=thread_id,
                run_id=run_id,
                message=str(exc),
            )
        finally:
            # Safety: close any hanging text message
            if text_message_open and current_message_id:
                try:
                    yield TextMessageEndEvent(
                        type=EventType.TEXT_MESSAGE_END,
                        message_id=current_message_id,
                    )
                except Exception:
                    pass
