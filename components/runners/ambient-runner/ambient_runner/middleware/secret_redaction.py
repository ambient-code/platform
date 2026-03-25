"""
AG-UI Secret Redaction Middleware — scrub secrets from outbound events.

Wraps an adapter's event stream to detect and redact secrets before they
reach the frontend. Combines two approaches:

1. **Pattern-based**: regex patterns for known token formats (GitHub PATs,
   Anthropic keys, Langfuse keys, Google API keys, credential URLs, etc.)
   via the existing ``redact_secrets()`` utility.

2. **Value-based**: collects actual secret values from environment variables
   at middleware init time and replaces exact occurrences. This catches
   secrets that don't match any known pattern format.

Usage::

    from ambient_runner.middleware import secret_redaction_middleware

    async for event in secret_redaction_middleware(bridge.run(input)):
        yield encoder.encode(event)
"""

import logging
import os
from typing import AsyncIterator

from ag_ui.core import (
    BaseEvent,
    CustomEvent,
    RunErrorEvent,
    TextMessageChunkEvent,
    TextMessageContentEvent,
    ToolCallArgsEvent,
    ToolCallChunkEvent,
    ToolCallResultEvent,
)

from ambient_runner.platform.utils import redact_secrets

logger = logging.getLogger(__name__)

# Environment variables that may contain secret values.
# Order matters: longer matches should come first to avoid partial replacements,
# so we sort by value length descending at collection time.
_SECRET_ENV_VARS = (
    "ANTHROPIC_API_KEY",
    "BOT_TOKEN",
    "GITHUB_TOKEN",
    "GITLAB_TOKEN",
    "JIRA_API_TOKEN",
    "GEMINI_API_KEY",
    "GOOGLE_API_KEY",
    "GOOGLE_OAUTH_CLIENT_SECRET",
    "LANGFUSE_SECRET_KEY",
    "LANGFUSE_PUBLIC_KEY",
    "LANGSMITH_API_KEY",
)


def _collect_secret_values() -> list[tuple[str, str]]:
    """Collect current secret values from environment, sorted longest-first."""
    pairs = []
    for var in _SECRET_ENV_VARS:
        val = (os.getenv(var) or "").strip()
        if len(val) >= 8:  # skip empty/trivially short values
            pairs.append((var, val))
    # Sort longest value first so longer tokens are replaced before shorter
    # substrings (e.g. a full PAT before a prefix that happens to match).
    pairs.sort(key=lambda p: len(p[1]), reverse=True)
    return pairs


def _redact_text(text: str, secret_values: list[tuple[str, str]]) -> str:
    """Apply both value-based and pattern-based redaction to a string."""
    for var_name, secret_val in secret_values:
        if secret_val in text:
            text = text.replace(secret_val, f"[REDACTED_{var_name}]")

    text = redact_secrets(text)

    return text


def _redact_event(event: BaseEvent, secret_values: list[tuple[str, str]]) -> BaseEvent:
    """Return a copy of the event with secrets redacted from text fields.

    Only processes event types that carry user-visible text. All other events
    pass through unchanged (zero cost).
    """
    if isinstance(event, (TextMessageContentEvent, TextMessageChunkEvent)):
        redacted = _redact_text(event.delta, secret_values)
        if redacted != event.delta:
            return event.model_copy(update={"delta": redacted})

    elif isinstance(event, (ToolCallArgsEvent, ToolCallChunkEvent)):
        redacted = _redact_text(event.delta, secret_values)
        if redacted != event.delta:
            return event.model_copy(update={"delta": redacted})

    elif isinstance(event, ToolCallResultEvent):
        if isinstance(event.content, str):
            redacted = _redact_text(event.content, secret_values)
            if redacted != event.content:
                return event.model_copy(update={"content": redacted})

    elif isinstance(event, RunErrorEvent):
        redacted = _redact_text(event.message, secret_values)
        if redacted != event.message:
            return event.model_copy(update={"message": redacted})

    elif isinstance(event, CustomEvent):
        if isinstance(event.value, str):
            redacted = _redact_text(event.value, secret_values)
            if redacted != event.value:
                return event.model_copy(update={"value": redacted})
        elif isinstance(event.value, dict):
            redacted_dict = _redact_dict(event.value, secret_values)
            if redacted_dict != event.value:
                return event.model_copy(update={"value": redacted_dict})

    return event


def _redact_dict(d: dict, secret_values: list[tuple[str, str]]) -> dict:
    """Recursively redact string values in a dict. Returns original if unchanged."""
    changed = False
    result = {}
    for k, v in d.items():
        if isinstance(v, str):
            redacted = _redact_text(v, secret_values)
            if redacted != v:
                changed = True
            result[k] = redacted
        elif isinstance(v, dict):
            redacted_v = _redact_dict(v, secret_values)
            if redacted_v is not v:
                changed = True
            result[k] = redacted_v
        else:
            result[k] = v
    return result if changed else d


async def secret_redaction_middleware(
    event_stream: AsyncIterator[BaseEvent],
) -> AsyncIterator[BaseEvent]:
    """Wrap an AG-UI event stream with secret redaction.

    Collects secret values from the current environment at invocation time
    and scrubs them from all text-bearing events before yielding.

    Args:
        event_stream: The upstream event stream.

    Yields:
        Events with secrets redacted from text fields.
    """
    secret_values = _collect_secret_values()

    async for event in event_stream:
        yield _redact_event(event, secret_values)
