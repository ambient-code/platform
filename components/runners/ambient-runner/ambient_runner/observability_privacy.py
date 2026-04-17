"""Shared privacy masking for observability backends (Langfuse, MLflow)."""

from __future__ import annotations

import os
from typing import Any, Callable


def resolve_message_mask_fn() -> Callable[[Any], Any] | None:
    """Return a mask callable when LANGFUSE_MASK_MESSAGES requests redaction.

    Uses the same env var as Langfuse for a single privacy knob across backends.
    """
    mask_messages_env = os.getenv("LANGFUSE_MASK_MESSAGES", "true").strip().lower()
    enable_masking = mask_messages_env not in ("false", "0", "no")
    return privacy_mask_message_data if enable_masking else None


def privacy_mask_message_data(data: Any, **kwargs: Any) -> Any:
    """Mask sensitive user inputs and outputs while preserving usage metrics."""
    if isinstance(data, str):
        if len(data) > 50:
            return "[REDACTED FOR PRIVACY]"
        return data
    if isinstance(data, dict):
        masked: dict[str, Any] = {}
        for key, value in data.items():
            if key in (
                "usage",
                "usage_details",
                "metadata",
                "model",
                "turn",
                "input_tokens",
                "output_tokens",
                "cache_read_input_tokens",
                "cache_creation_input_tokens",
                "total_tokens",
                "cost_usd",
                "duration_ms",
                "duration_api_ms",
                "num_turns",
                "session_id",
                "tool_id",
                "tool_name",
                "is_error",
                "level",
            ):
                masked[key] = value
            elif key in ("content", "text", "input", "output", "prompt", "completion"):
                if isinstance(value, str) and len(value) > 50:
                    masked[key] = "[REDACTED FOR PRIVACY]"
                else:
                    masked[key] = privacy_mask_message_data(value)
            else:
                masked[key] = privacy_mask_message_data(value)
        return masked
    if isinstance(data, list):
        return [privacy_mask_message_data(item) for item in data]
    return data
