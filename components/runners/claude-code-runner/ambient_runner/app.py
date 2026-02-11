"""
``add_ambient_endpoints(app, bridge)`` — wire Ambient platform endpoints
into a FastAPI application.

This is the public API of the ambient_runner package. Framework authors
call this once to get all platform features (run, interrupt, health,
capabilities, feedback, repos, workflows, MCP diagnostics).

Usage::

    from ambient_runner import add_ambient_endpoints
    from ambient_runner.bridges.claude import ClaudeBridge

    app = FastAPI()
    add_ambient_endpoints(app, bridge=ClaudeBridge())
"""

import logging
from typing import Optional

from fastapi import FastAPI

from ambient_runner.bridge import PlatformBridge

logger = logging.getLogger(__name__)


def add_ambient_endpoints(
    app: FastAPI,
    bridge: PlatformBridge,
    *,
    enable_repos: bool = True,
    enable_workflows: bool = True,
    enable_feedback: bool = True,
    enable_mcp_status: bool = True,
    enable_capabilities: bool = True,
) -> None:
    """Register Ambient platform endpoints on a FastAPI app.

    Args:
        app: The FastAPI application.
        bridge: A ``PlatformBridge`` implementation for the chosen framework.
        enable_repos: Include /repos/* endpoints.
        enable_workflows: Include /workflow endpoint.
        enable_feedback: Include /feedback endpoint.
        enable_mcp_status: Include /mcp/status endpoint.
        enable_capabilities: Include /capabilities endpoint.
    """
    # Store bridge on app state so endpoints can access it
    app.state.bridge = bridge

    # Core endpoints (always registered)
    from ambient_runner.endpoints.run import router as run_router
    from ambient_runner.endpoints.interrupt import router as interrupt_router
    from ambient_runner.endpoints.health import router as health_router

    app.include_router(run_router)
    app.include_router(interrupt_router)
    app.include_router(health_router)

    # Optional platform endpoints
    if enable_capabilities:
        from ambient_runner.endpoints.capabilities import router as cap_router
        app.include_router(cap_router)

    if enable_feedback:
        from ambient_runner.endpoints.feedback import router as fb_router
        app.include_router(fb_router)

    if enable_repos:
        from ambient_runner.endpoints.repos import router as repos_router
        app.include_router(repos_router)

    if enable_workflows:
        from ambient_runner.endpoints.workflow import router as wf_router
        app.include_router(wf_router)

    if enable_mcp_status:
        from ambient_runner.endpoints.mcp_status import router as mcp_router
        app.include_router(mcp_router)

    caps = bridge.capabilities()
    logger.info(
        f"Ambient endpoints registered: framework={caps.framework}, "
        f"features={caps.agent_features}"
    )
