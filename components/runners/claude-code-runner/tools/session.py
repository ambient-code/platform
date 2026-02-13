"""
Session control MCP tool — allows Claude to request a session restart.
"""

import logging

from prompts import RESTART_TOOL_DESCRIPTION

logger = logging.getLogger(__name__)


def create_restart_session_tool(adapter_ref, sdk_tool_decorator):
    """Create the restart_session MCP tool.

    Args:
        adapter_ref: Reference to the ClaudeCodeAdapter instance
            (used to set _restart_requested flag).
        sdk_tool_decorator: The ``tool`` decorator from ``claude_agent_sdk``.

    Returns:
        Decorated async tool function.
    """

    @sdk_tool_decorator(
        "restart_session",
        RESTART_TOOL_DESCRIPTION,
        {},
    )
    async def restart_session_tool(args: dict) -> dict:
        """Tool that allows Claude to request a session restart."""
        adapter_ref._restart_requested = True
        logger.info("🔄 Session restart requested by Claude via MCP tool")
        return {
            "content": [
                {
                    "type": "text",
                    "text": (
                        "Session restart has been requested. The current run "
                        "will complete and a fresh session will be established. "
                        "Your conversation context will be preserved on disk."
                    ),
                }
            ]
        }

    return restart_session_tool
