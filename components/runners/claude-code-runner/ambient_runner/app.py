"""
Ambient Runner SDK — FastAPI application factory.

Provides three public APIs:

- ``create_ambient_app(bridge)`` — creates a fully wired FastAPI app with
  lifespan, endpoints, and the platform lifecycle (context, auto-prompt,
  shutdown).  This is the recommended way to build a runner.

- ``run_ambient_app(bridge)`` — creates the app AND starts the uvicorn
  server. One-liner entry point for runners.

- ``add_ambient_endpoints(app, bridge)`` — lower-level: registers only the
  endpoint routers on an existing app (caller owns the lifespan).

Usage::

    from ambient_runner import run_ambient_app
    from ambient_runner.bridges.claude import ClaudeBridge

    run_ambient_app(ClaudeBridge(), title="Claude Code AG-UI Server")
"""

import asyncio
import logging
import os
import uuid
from contextlib import asynccontextmanager
from typing import Optional

import aiohttp
from fastapi import FastAPI

from ambient_runner.bridge import PlatformBridge
from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


def _log_auto_exec_failure(task: asyncio.Task) -> None:
    """Callback for the auto-execution task — logs unhandled exceptions."""
    if task.cancelled():
        logger.warning("Auto-execution task was cancelled")
        return
    exc = task.exception()
    if exc is not None:
        logger.error(
            "Auto-execution of INITIAL_PROMPT failed: %s: %s",
            type(exc).__name__,
            exc,
        )


# ------------------------------------------------------------------
# High-level: create_ambient_app
# ------------------------------------------------------------------


def create_ambient_app(
    bridge: PlatformBridge,
    *,
    title: str = "Ambient AG-UI Server",
    version: str = "0.3.0",
    enable_repos: bool = True,
    enable_workflows: bool = True,
    enable_feedback: bool = True,
    enable_mcp_status: bool = True,
    enable_capabilities: bool = True,
    enable_content: bool = True,
) -> FastAPI:
    """Create a fully wired FastAPI application for an AG-UI runner.

    Handles the full platform lifecycle:

    1. **Startup** — creates ``RunnerContext`` from env vars, sets it on the
       bridge, and fires the auto-prompt for non-interactive sessions.
    2. **Request handling** — all Ambient endpoints are registered and
       delegate to the bridge.
    3. **Shutdown** — calls ``bridge.shutdown()`` for graceful cleanup.

    Args:
        bridge: A ``PlatformBridge`` implementation (e.g. ``ClaudeBridge``).
        title: FastAPI application title.
        version: Application version string.
        enable_*: Toggle optional endpoint groups.

    Returns:
        A ready-to-use ``FastAPI`` application.
    """

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        session_id = os.getenv("SESSION_ID", "unknown")
        workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")

        logger.info(f"Initializing AG-UI server for session {session_id}")

        context = RunnerContext(
            session_id=session_id,
            workspace_path=workspace_path,
        )
        bridge.set_context(context)

        # Resume detection
        is_resume = os.getenv("IS_RESUME", "").strip().lower() == "true"
        if is_resume:
            logger.info("IS_RESUME=true — this is a resumed session")

        # Auto-prompt for non-interactive, non-resumed sessions
        is_interactive = os.getenv("INTERACTIVE", "true").strip().lower() == "true"
        initial_prompt = os.getenv("INITIAL_PROMPT", "").strip()

        if initial_prompt:
            if not is_interactive and not is_resume:
                logger.info(
                    f"INITIAL_PROMPT detected ({len(initial_prompt)} chars) "
                    f"— auto-executing for non-interactive session"
                )
                task = asyncio.create_task(
                    _auto_execute_initial_prompt(initial_prompt, session_id)
                )
                task.add_done_callback(_log_auto_exec_failure)
            else:
                mode = "resumed" if is_resume else "interactive"
                logger.info(
                    f"INITIAL_PROMPT detected ({len(initial_prompt)} chars) "
                    f"but not auto-executing ({mode} session)"
                )

        logger.info(f"AG-UI server ready for session {session_id}")

        yield

        await bridge.shutdown()
        logger.info("AG-UI server shut down")

    app = FastAPI(title=title, version=version, lifespan=lifespan)

    add_ambient_endpoints(
        app,
        bridge,
        enable_repos=enable_repos,
        enable_workflows=enable_workflows,
        enable_feedback=enable_feedback,
        enable_mcp_status=enable_mcp_status,
        enable_capabilities=enable_capabilities,
        enable_content=enable_content,
    )

    return app


# ------------------------------------------------------------------
# Low-level: add_ambient_endpoints
# ------------------------------------------------------------------


def add_ambient_endpoints(
    app: FastAPI,
    bridge: PlatformBridge,
    *,
    enable_repos: bool = True,
    enable_workflows: bool = True,
    enable_feedback: bool = True,
    enable_mcp_status: bool = True,
    enable_capabilities: bool = True,
    enable_content: bool = True,
) -> None:
    """Register Ambient platform endpoints on an existing FastAPI app.

    Use this when you need to own the lifespan yourself.  For most cases,
    prefer ``create_ambient_app()`` instead.

    Args:
        app: The FastAPI application.
        bridge: A ``PlatformBridge`` implementation for the chosen framework.
        enable_*: Toggle optional endpoint groups.
    """
    # Store bridge on app state so endpoints can access it
    app.state.bridge = bridge

    # Core endpoints (always registered)
    from ambient_runner.endpoints.health import router as health_router
    from ambient_runner.endpoints.interrupt import router as interrupt_router
    from ambient_runner.endpoints.run import router as run_router

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

    if enable_content:
        from ambient_runner.endpoints.content import router as content_router

        app.include_router(content_router)

    caps = bridge.capabilities()
    logger.info(
        f"Ambient endpoints registered: framework={caps.framework}, "
        f"features={caps.agent_features}"
    )


# ------------------------------------------------------------------
# Platform: auto-execute initial prompt
# ------------------------------------------------------------------


async def _auto_execute_initial_prompt(prompt: str, session_id: str) -> None:
    """Auto-execute INITIAL_PROMPT for non-interactive sessions.

    Waits briefly then POSTs the prompt to the backend's AG-UI run
    endpoint, which routes it back through the runner's ``POST /``.
    """
    delay_seconds = float(os.getenv("INITIAL_PROMPT_DELAY_SECONDS", "1"))
    logger.info(f"Waiting {delay_seconds}s before auto-executing INITIAL_PROMPT...")
    await asyncio.sleep(delay_seconds)

    backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
    project_name = (
        os.getenv("PROJECT_NAME", "").strip()
        or os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
    )

    if not backend_url or not project_name:
        logger.error(
            "Cannot auto-execute INITIAL_PROMPT: "
            "BACKEND_API_URL or PROJECT_NAME not set"
        )
        return

    url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"

    payload = {
        "threadId": session_id,
        "runId": str(uuid.uuid4()),
        "messages": [
            {
                "id": str(uuid.uuid4()),
                "role": "user",
                "content": prompt,
                "metadata": {
                    "hidden": True,
                    "autoSent": True,
                    "source": "runner_initial_prompt",
                },
            }
        ],
    }

    bot_token = os.getenv("BOT_TOKEN", "").strip()
    headers = {"Content-Type": "application/json"}
    if bot_token:
        headers["Authorization"] = f"Bearer {bot_token}"

    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(
                url,
                json=payload,
                headers=headers,
                timeout=aiohttp.ClientTimeout(total=30),
            ) as resp:
                if resp.status == 200:
                    logger.info(
                        f"INITIAL_PROMPT auto-execution started: {await resp.json()}"
                    )
                else:
                    logger.warning(
                        f"INITIAL_PROMPT failed with status {resp.status}: "
                        f"{(await resp.text())[:200]}"
                    )
    except Exception as e:
        logger.warning(
            f"INITIAL_PROMPT auto-execution error (backend will retry): {e}"
        )


# ------------------------------------------------------------------
# One-liner: run_ambient_app
# ------------------------------------------------------------------


def run_ambient_app(
    app_or_bridge: FastAPI | PlatformBridge,
    *,
    title: str = "Ambient AG-UI Server",
    version: str = "0.3.0",
    host: str | None = None,
    port: int | None = None,
    log_level: str = "info",
    **kwargs,
) -> None:
    """Start the uvicorn server for an Ambient runner.

    Accepts either a pre-built ``FastAPI`` app (from ``create_ambient_app``)
    or a ``PlatformBridge`` (creates the app for you).

    Reads ``AGUI_HOST`` and ``AGUI_PORT`` from environment if not provided.

    Args:
        app_or_bridge: A ``FastAPI`` app or a ``PlatformBridge`` implementation.
        title: FastAPI application title (only used if bridge is passed).
        version: Application version string (only used if bridge is passed).
        host: Bind address (default: ``AGUI_HOST`` env or ``0.0.0.0``).
        port: Bind port (default: ``AGUI_PORT`` env or ``8000``).
        log_level: Uvicorn log level.
        **kwargs: Passed through to ``create_ambient_app()`` if bridge is passed.
    """
    import uvicorn

    if isinstance(app_or_bridge, FastAPI):
        app = app_or_bridge
    else:
        app = create_ambient_app(app_or_bridge, title=title, version=version, **kwargs)

    resolved_host = host or os.getenv("AGUI_HOST", "0.0.0.0")
    resolved_port = port or int(os.getenv("AGUI_PORT", "8000"))

    logger.info(f"Starting on {resolved_host}:{resolved_port}")
    uvicorn.run(app, host=resolved_host, port=resolved_port, log_level=log_level)
