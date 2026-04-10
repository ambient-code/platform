"""Generic hook -> CustomEvent bridge for Claude Agent SDK hooks.

Converts any Claude SDK hook callback into an AG-UI ``CustomEvent`` and
pushes it onto an ``asyncio.Queue`` for the adapter's SSE stream to drain.

Usage::

    from ag_ui_claude_sdk.hooks import build_hooks_dict

    queue = asyncio.Queue()
    hooks = build_hooks_dict(queue)
    # pass as ClaudeAgentOptions(hooks=hooks)
"""

import asyncio
import logging
import os
from typing import Any

from ag_ui.core import CustomEvent, EventType

logger = logging.getLogger(__name__)

# Default hook event names to register (only what the UI consumes).
_DEFAULT_HOOKS = frozenset({
    "SubagentStart",
    "SubagentStop",
    "Notification",
    "Stop",
})

# Keys stripped from payloads (internal paths the frontend should not see).
_SANITIZE_KEYS = frozenset({"transcript_path", "cwd"})


async def _forward_hook_as_custom_event(
    hook_input: dict[str, Any],
    tool_use_id: str,
    queue: asyncio.Queue,
) -> dict:
    """Generic async callback that converts a hook input into a CustomEvent.

    Args:
        hook_input: The raw hook payload dict from the SDK.
        tool_use_id: The tool-use ID associated with this hook invocation.
        queue: The asyncio queue shared with the adapter for SSE draining.

    Returns:
        Empty dict (no hook result to return to the SDK).
    """
    event_name = hook_input.get("hook_event_name", "unknown")
    payload = {k: v for k, v in hook_input.items() if k not in _SANITIZE_KEYS}

    logger.debug("[Hook] %s fired (agent_id=%s)", event_name, hook_input.get("agent_id", "n/a"))

    await queue.put(
        CustomEvent(
            type=EventType.CUSTOM,
            name=f"hook:{event_name}",
            value=payload,
        )
    )
    return {}


def _parse_hook_names() -> set[str]:
    """Return the set of hook event names to register.

    Starts with ``_DEFAULT_HOOKS`` and extends with any names listed in the
    ``AGUI_HOOKS`` environment variable (comma-separated).
    """
    names = set(_DEFAULT_HOOKS)
    extra = os.getenv("AGUI_HOOKS", "").strip()
    if extra:
        for name in extra.split(","):
            name = name.strip()
            if name:
                names.add(name)
    return names


def build_hooks_dict(queue: asyncio.Queue) -> dict[str, list[Any]]:
    """Build a ``hooks`` dict for ``ClaudeAgentOptions``.

    The SDK expects ``hooks`` to be ``dict[HookEvent, list[HookMatcher]]``.
    Each entry targets one hook event name and routes it to the generic
    ``_forward_hook_as_custom_event`` callback, which pushes a ``CustomEvent``
    onto *queue*.

    Args:
        queue: The asyncio queue the adapter drains between SDK messages.

    Returns:
        Dict mapping hook event names to ``[HookMatcher(...)]`` lists,
        ready to pass as ``ClaudeAgentOptions(hooks=...)``.
    """
    from claude_agent_sdk import HookMatcher

    hook_names = _parse_hook_names()
    hooks: dict[str, list[Any]] = {}

    for name in sorted(hook_names):

        async def _callback(
            hook_input: dict[str, Any],
            tool_use_id: str | None = None,
            _context: Any = None,
            _q: asyncio.Queue = queue,
        ) -> dict:
            return await _forward_hook_as_custom_event(
                hook_input, tool_use_id or "", _q
            )

        hooks[name] = [HookMatcher(hooks=[_callback])]

    logger.info("Registered %d hook events: %s", len(hooks), sorted(hook_names))
    return hooks
