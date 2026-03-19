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
from pathlib import Path
from urllib.parse import urlparse

import aiohttp
from fastapi import FastAPI

from ambient_runner.bridge import PlatformBridge
from ambient_runner.bridges.claude.bridge import ClaudeBridge
from ambient_runner.platform.config import load_ambient_config
from ambient_runner.platform.context import RunnerContext
from ambient_runner.platform.utils import parse_owner_repo

# Configure root logger so all ambient_runner.* and ag_ui_* loggers
# have a handler and respect the LOG_LEVEL env var.
logging.basicConfig(
    level=getattr(logging, os.getenv("LOG_LEVEL", "INFO").upper(), logging.INFO),
    format="%(levelname)s:%(name)s:%(message)s",
)

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
       bridge, and fires the auto-prompt if INITIAL_PROMPT is set.
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

        # Eager gRPC listener setup (ClaudeBridge only).
        # Must complete before INITIAL_PROMPT is dispatched so the listener
        # is subscribed before PushSessionMessage fires.
        #
        # OPERATOR COMPATIBILITY: The existing Operator never injects AMBIENT_GRPC_URL
        # into Job pods, so grpc_url is always empty on the Operator path. This entire
        # block — and all gRPC code in run.py — is a strict no-op for operator-created
        # sessions. No existing Operator/Runner behavior is changed by this PR.
        grpc_url = os.getenv("AMBIENT_GRPC_URL", "").strip()
        if grpc_url and isinstance(bridge, ClaudeBridge):
            await bridge._setup_platform()
            await bridge._grpc_listener.ready.wait()
            logger.info(
                "gRPC listener ready for session %s — proceeding to INITIAL_PROMPT",
                session_id,
            )

        # Auto-execute prompts when present (skipped only for resumes,
        # where the conversation is continued rather than re-started).
        if not is_resume:
            # Fetch workflow startupPrompt independently
            workflow_startup_prompt = _get_workflow_startup_prompt()
            user_initial_prompt = os.getenv("INITIAL_PROMPT", "").strip()

            # Combine prompts: workflow first, then user initial prompt
            combined_prompt = ""
            if workflow_startup_prompt and user_initial_prompt:
                logger.info(
                    f"Both workflow startupPrompt ({len(workflow_startup_prompt)} chars) "
                    f"and user INITIAL_PROMPT ({len(user_initial_prompt)} chars) detected"
                )
                combined_prompt = f"{workflow_startup_prompt}\n\n{user_initial_prompt}"
            elif workflow_startup_prompt:
                logger.info(
                    f"Workflow startupPrompt ({len(workflow_startup_prompt)} chars) detected"
                )
                combined_prompt = workflow_startup_prompt
            elif user_initial_prompt:
                logger.info(
                    f"User INITIAL_PROMPT ({len(user_initial_prompt)} chars) detected"
                )
                combined_prompt = user_initial_prompt

            # Auto-execute if we have any prompt
            if combined_prompt:
                logger.info(
                    f"Auto-executing combined prompt ({len(combined_prompt)} chars)"
                )
                task = asyncio.create_task(
                    _auto_execute_initial_prompt(combined_prompt, session_id, grpc_url)
                )
                task.add_done_callback(_log_auto_exec_failure)
        else:
            # Log but don't execute on resume (avoid filesystem I/O, just check env vars)
            has_workflow = bool(os.getenv("ACTIVE_WORKFLOW_GIT_URL", "").strip())
            has_user_prompt = bool(os.getenv("INITIAL_PROMPT", "").strip())
            if has_workflow or has_user_prompt:
                logger.info("Prompts detected but not auto-executing (resumed session)")

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
    from ambient_runner.endpoints.events import router as events_router
    from ambient_runner.endpoints.health import router as health_router
    from ambient_runner.endpoints.interrupt import router as interrupt_router
    from ambient_runner.endpoints.run import router as run_router

    app.include_router(run_router)
    app.include_router(interrupt_router)
    app.include_router(health_router)
    app.include_router(events_router)

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
# Platform: resolve workflow startup prompt
# ------------------------------------------------------------------


def _get_workflow_startup_prompt() -> str:
    """Load startupPrompt from the active workflow's ambient.json.

    Returns the startupPrompt string, or empty string if no workflow
    is active or the config has no startupPrompt.
    """
    active_url = os.getenv("ACTIVE_WORKFLOW_GIT_URL", "").strip()
    if not active_url:
        return ""

    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")

    try:
        _owner, repo, _ = parse_owner_repo(active_url)
        derived_name = repo or ""
        if not derived_name:
            p = urlparse(active_url)
            parts = [pt for pt in (p.path or "").split("/") if pt]
            if parts:
                derived_name = parts[-1]
        derived_name = (derived_name or "").removesuffix(".git").strip()
    except Exception:
        derived_name = ""

    if not derived_name:
        return ""

    workflow_dir = str(Path(workspace_path) / "workflows" / derived_name)
    if not Path(workflow_dir).exists():
        return ""

    config = load_ambient_config(workflow_dir)
    startup = (config.get("startupPrompt") or "").strip()
    if startup:
        logger.info(f"Found startupPrompt in {derived_name}/ambient.json")
    return startup


# ------------------------------------------------------------------
# Platform: auto-execute initial prompt
# ------------------------------------------------------------------


_AUTO_PROMPT_MAX_RETRIES = 8
_AUTO_PROMPT_INITIAL_DELAY = 2.0
_AUTO_PROMPT_MAX_DELAY = 30.0


async def _auto_execute_initial_prompt(
    prompt: str, session_id: str, grpc_url: str = ""
) -> None:
    """Auto-execute INITIAL_PROMPT on session startup.

    When AMBIENT_GRPC_URL is set, pushes the initial prompt as a DB Message
    via PushSessionMessage so the GRPCSessionListener picks it up and triggers
    the run directly. The prompt is then observable to API consumers and
    visible in the frontend session history.

    When AMBIENT_GRPC_URL is not set, falls back to the original HTTP POST
    path with exponential-backoff retry (for DNS propagation races).
    """
    delay_seconds = float(os.getenv("INITIAL_PROMPT_DELAY_SECONDS", "2"))
    logger.info(f"Waiting {delay_seconds}s before auto-executing INITIAL_PROMPT...")
    await asyncio.sleep(delay_seconds)

    if grpc_url:
        await _push_initial_prompt_via_grpc(prompt, session_id)
    else:
        await _push_initial_prompt_via_http(prompt, session_id)


async def _push_initial_prompt_via_grpc(prompt: str, session_id: str) -> None:
    """Push INITIAL_PROMPT as a PushSessionMessage so it is durable in DB."""
    try:
        from ambient_runner._grpc_client import AmbientGRPCClient

        client = AmbientGRPCClient.from_env()
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
        import json as _json

        result = client.session_messages.push(
            session_id,
            event_type="user",
            payload=_json.dumps(payload),
        )
        if result is not None:
            logger.info(
                "INITIAL_PROMPT pushed via gRPC: session=%s seq=%d",
                session_id,
                result.seq,
            )
        else:
            logger.warning(
                "INITIAL_PROMPT gRPC push returned None (push may have failed): session=%s",
                session_id,
            )
        client.close()
    except Exception as exc:
        logger.error(
            "INITIAL_PROMPT gRPC push failed: session=%s error=%s",
            session_id,
            exc,
            exc_info=True,
        )


async def _push_initial_prompt_via_http(prompt: str, session_id: str) -> None:
    """HTTP POST fallback: push INITIAL_PROMPT to local AG-UI run endpoint."""
    agui_port = os.getenv("AGUI_PORT", "8001")
    url = f"http://localhost:{agui_port}/"

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

    backoff = _AUTO_PROMPT_INITIAL_DELAY
    for attempt in range(1, _AUTO_PROMPT_MAX_RETRIES + 1):
        try:
            async with aiohttp.ClientSession() as session:
                async with session.post(
                    url,
                    json=payload,
                    headers=headers,
                    timeout=aiohttp.ClientTimeout(total=30),
                ) as resp:
                    body = await resp.text()
                    if resp.status == 200:
                        logger.info("INITIAL_PROMPT auto-execution started (HTTP)")
                        return

                    logger.error(
                        f"INITIAL_PROMPT failed with status {resp.status}: {body[:200]}"
                    )
                    return
        except Exception as e:
            logger.warning(
                f"INITIAL_PROMPT attempt {attempt}/{_AUTO_PROMPT_MAX_RETRIES} "
                f"error: {e}, retrying in {backoff:.0f}s"
            )

        await asyncio.sleep(backoff)
        backoff = min(backoff * 2, _AUTO_PROMPT_MAX_DELAY)
        payload["runId"] = str(uuid.uuid4())

    logger.error(
        f"INITIAL_PROMPT auto-execution failed after {_AUTO_PROMPT_MAX_RETRIES} attempts"
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
