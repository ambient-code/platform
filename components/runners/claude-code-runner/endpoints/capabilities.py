"""GET /capabilities — reports what this runner/framework supports.

The capabilities response tells the frontend which UI panels to show
(repos, workflows, MCP diagnostics, feedback, etc.) and what AG-UI
features the underlying framework adapter provides.
"""

import logging
import os

from fastapi import APIRouter

from endpoints import state

logger = logging.getLogger(__name__)

router = APIRouter()

# Framework-specific capabilities for Claude Agent SDK
_CLAUDE_AGENT_FEATURES = [
    "agentic_chat",
    "backend_tool_rendering",
    "shared_state",
    "human_in_the_loop",
    "thinking",
]

# Platform features exposed by our endpoints
_PLATFORM_FEATURES = [
    "repos",
    "workflows",
    "feedback",
    "mcp_diagnostics",
]


@router.get("/capabilities")
async def get_capabilities():
    """Return the full capabilities manifest for this runner session."""
    has_langfuse = state._obs is not None and state._obs.langfuse_client is not None

    return {
        "framework": "claude-agent-sdk",
        "agent_features": _CLAUDE_AGENT_FEATURES,
        "platform_features": _PLATFORM_FEATURES,
        "file_system": True,
        "mcp": True,
        "tracing": "langfuse" if has_langfuse else None,
        "session_persistence": True,
        "model": state._configured_model or None,
        "session_id": state.context.session_id if state.context else None,
    }
