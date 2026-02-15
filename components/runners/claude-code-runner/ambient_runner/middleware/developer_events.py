"""
AG-UI Developer Events — platform setup lifecycle events.

Emits ``TextMessage`` events with ``role="developer"`` to report platform
setup progress (auth, workspace, MCP servers) through the AG-UI stream.

The frontend can show or hide developer messages via a debug toggle.

Usage::

    from ambient_runner.middleware import emit_developer_message

    async for event in emit_developer_message("Auth connected via API key"):
        yield encoder.encode(event)
"""

import logging
import uuid
from typing import AsyncIterator

from ag_ui.core import (
    BaseEvent,
    EventType,
    TextMessageContentEvent,
    TextMessageEndEvent,
    TextMessageStartEvent,
)

logger = logging.getLogger(__name__)


async def emit_developer_message(text: str) -> AsyncIterator[BaseEvent]:
    """Emit a single developer message as three AG-UI events.

    Args:
        text: The developer message content.

    Yields:
        TextMessageStartEvent, TextMessageContentEvent, TextMessageEndEvent
    """
    msg_id = str(uuid.uuid4())

    yield TextMessageStartEvent(
        type=EventType.TEXT_MESSAGE_START,
        message_id=msg_id,
        role="developer",
    )
    yield TextMessageContentEvent(
        type=EventType.TEXT_MESSAGE_CONTENT,
        message_id=msg_id,
        delta=text,
    )
    yield TextMessageEndEvent(
        type=EventType.TEXT_MESSAGE_END,
        message_id=msg_id,
    )

    logger.debug(f"Developer event emitted: {text}")
