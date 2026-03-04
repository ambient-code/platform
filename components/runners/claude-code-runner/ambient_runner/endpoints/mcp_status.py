"""GET /mcp/status — MCP server connection diagnostics.

Delegates to ``bridge.get_mcp_status()`` which creates an ephemeral SDK
client (framework-specific) to query live MCP server status.
"""

import logging

from fastapi import APIRouter, Request

logger = logging.getLogger(__name__)

router = APIRouter()


@router.get("/mcp/status")
async def get_mcp_status(request: Request):
    """Returns MCP server connection status via the bridge."""
    bridge = request.app.state.bridge
    return await bridge.get_mcp_status()
