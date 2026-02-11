"""GET /capabilities — reports framework and platform capabilities."""

import logging

from fastapi import APIRouter, Request

logger = logging.getLogger(__name__)

router = APIRouter()


@router.get("/capabilities")
async def get_capabilities(request: Request):
    """Return the capabilities manifest from the bridge."""
    bridge = request.app.state.bridge
    caps = bridge.capabilities()

    return {
        "framework": caps.framework,
        "agent_features": caps.agent_features,
        "platform_features": [],  # Filled dynamically from registered routers
        "file_system": caps.file_system,
        "mcp": caps.mcp,
        "tracing": caps.tracing,
        "session_persistence": caps.session_persistence,
    }
