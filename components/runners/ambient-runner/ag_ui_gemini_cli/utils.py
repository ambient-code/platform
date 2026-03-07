"""Utility functions for Gemini CLI adapter."""

import logging
from typing import Any

from ag_ui.core import RunAgentInput

logger = logging.getLogger(__name__)


def extract_user_message(input_data: RunAgentInput) -> str:
    """Extract the last user message text from AG-UI input.

    Handles both string content and content-block formats.
    """
    messages = input_data.messages or []
    if not messages:
        return ""

    last_msg = messages[-1]

    # Extract content
    if hasattr(last_msg, "content"):
        content: Any = last_msg.content
    elif isinstance(last_msg, dict):
        content = last_msg.get("content", "")
    else:
        content = ""

    # String content
    if isinstance(content, str):
        return content

    # Content blocks
    if isinstance(content, list):
        for block in content:
            if hasattr(block, "text"):
                return block.text
            if isinstance(block, dict) and "text" in block:
                return block["text"]

    return ""
