"""GET /capabilities — reports framework and platform capabilities."""

import logging

from fastapi import APIRouter, Request

logger = logging.getLogger(__name__)

router = APIRouter()

# Map of route paths to platform feature names
_ROUTE_TO_FEATURE = {
    "/repos/add": "repos",
    "/repos/remove": "repos",
    "/repos/status": "repos",
    "/workflow": "workflows",
    "/feedback": "feedback",
    "/mcp/status": "mcp_diagnostics",
}


def _detect_platform_features(app) -> list[str]:
    """Detect platform features from registered routes."""
    features = set()
    for route in app.routes:
        path = getattr(route, "path", "")
        if path in _ROUTE_TO_FEATURE:
            features.add(_ROUTE_TO_FEATURE[path])
    return sorted(features)


@router.get("/capabilities")
async def get_capabilities(request: Request):
    """Return the capabilities manifest from the bridge."""
    bridge = request.app.state.bridge
    caps = bridge.capabilities()
    context = bridge.context

    platform_features = _detect_platform_features(request.app)

    return {
        "framework": caps.framework,
        "agent_features": caps.agent_features,
        "platform_features": platform_features,
        "file_system": caps.file_system,
        "mcp": caps.mcp,
        "tracing": caps.tracing,
        "session_persistence": caps.session_persistence,
        "model": bridge.configured_model or None,
        "session_id": context.session_id if context else None,
    }
