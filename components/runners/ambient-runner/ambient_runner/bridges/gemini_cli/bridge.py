"""GeminiCLIBridge -- full-lifecycle PlatformBridge for the Gemini CLI.

Owns the Gemini CLI session lifecycle:
- Platform setup (auth, workspace, observability)
- Adapter creation
- Session worker management (one invocation per turn)
- Tracing middleware integration
- Interrupt and graceful shutdown
"""

import asyncio
import logging
import os
import time
from typing import Any, AsyncIterator, Optional

from ag_ui.core import BaseEvent, RunAgentInput
from ag_ui_gemini_cli import GeminiCLIAdapter
from ag_ui_gemini_cli.utils import extract_user_message

from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
    _async_safe_manager_shutdown,
    setup_bridge_observability,
)
from ambient_runner.bridges.gemini_cli.session import (
    SHUTDOWN_TIMEOUT_SEC,
    GeminiSessionManager,
)
from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


class GeminiCLIBridge(PlatformBridge):
    """Bridge between the Ambient platform and the Gemini CLI.

    Handles lazy platform initialisation on first ``run()`` call, manages
    ``GeminiSessionWorker`` instances (one CLI invocation per turn), and
    wraps the event stream with Langfuse tracing.
    """

    def __init__(self) -> None:
        super().__init__()
        self._session_manager: GeminiSessionManager | None = None
        self._adapter: GeminiCLIAdapter | None = None
        self._obs: Any = None

        # Platform state (populated by _setup_platform)
        self._configured_model: str = ""
        self._api_key: str = ""
        self._use_vertex: bool = False
        self._cwd_path: str = ""
        self._include_directories: list[str] = []
        self._mcp_settings_path: str | None = None
        self._mcp_status_cache: dict | None = None

    # ------------------------------------------------------------------
    # PlatformBridge interface
    # ------------------------------------------------------------------

    def capabilities(self) -> FrameworkCapabilities:
        has_tracing = (
            self._obs is not None
            and hasattr(self._obs, "langfuse_client")
            and self._obs.langfuse_client is not None
        )
        return FrameworkCapabilities(
            framework="gemini-cli",
            agent_features=["agentic_chat", "backend_tool_rendering"],
            file_system=True,
            mcp=True,
            tracing="langfuse" if has_tracing else None,
        )

    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        """Full run lifecycle: lazy setup -> session worker -> tracing."""
        # 1. Lazy platform setup
        await self._ensure_ready()
        await self._refresh_credentials_if_stale()

        # 2. Extract user message
        user_msg = extract_user_message(input_data)

        # 3. Get session worker for this thread
        thread_id = input_data.thread_id or self._context.session_id
        worker = self._session_manager.get_or_create_worker(
            thread_id,
            model=self._configured_model,
            api_key=self._api_key,
            use_vertex=self._use_vertex,
            cwd=self._cwd_path,
            include_directories=self._include_directories,
        )

        # 4. Get last session_id for --resume
        session_id = self._session_manager.get_session_id(thread_id)

        # 5. Get line stream from worker, wrap with session_id capture
        async def _line_stream_with_capture():
            import json as _json

            async for line in worker.query(user_msg, session_id=session_id):
                # Capture session_id from init events for future --resume
                # Use lightweight JSON check instead of full parse_event() to
                # avoid double-parsing every line (adapter parses it again).
                if '"type":"init"' in line or '"type": "init"' in line:
                    try:
                        raw = _json.loads(line)
                        sid = raw.get("session_id")
                        if sid:
                            self._session_manager.set_session_id(thread_id, sid)
                    except (ValueError, KeyError):
                        pass
                yield line

        # 6. Create adapter and run
        if self._adapter is None:
            self._adapter = GeminiCLIAdapter()

        async with self._session_manager.get_lock(thread_id):
            from ambient_runner.middleware import tracing_middleware

            wrapped_stream = tracing_middleware(
                self._adapter.run(input_data, line_stream=_line_stream_with_capture()),
                obs=self._obs,
                model=self._configured_model,
                prompt=user_msg,
            )

            async for event in wrapped_stream:
                yield event

    async def interrupt(self, thread_id: Optional[str] = None) -> None:
        """Interrupt the running session for a given thread."""
        if not self._session_manager:
            raise RuntimeError("No active session manager")

        tid = thread_id or (self._context.session_id if self._context else None)
        if not tid:
            raise RuntimeError("No thread_id available")

        logger.info("Interrupt request for thread=%s", tid)
        await self._session_manager.interrupt(tid)

    # ------------------------------------------------------------------
    # Lifecycle methods
    # ------------------------------------------------------------------

    async def shutdown(self) -> None:
        """Graceful shutdown: stop workers, finalise tracing."""
        if self._session_manager:
            try:
                await asyncio.wait_for(
                    self._session_manager.shutdown(),
                    timeout=SHUTDOWN_TIMEOUT_SEC * 3,
                )
            except asyncio.TimeoutError:
                logger.warning(
                    "GeminiCLIBridge: manager shutdown timed out after %ds",
                    SHUTDOWN_TIMEOUT_SEC * 3,
                )
        if self._obs:
            await self._obs.finalize()
        logger.info("GeminiCLIBridge: shutdown complete")

    def mark_dirty(self) -> None:
        """Signal reinitialisation on next run."""
        self._ready = False
        self._adapter = None
        self._mcp_status_cache = None
        if self._session_manager:
            manager = self._session_manager
            self._session_manager = None
            _async_safe_manager_shutdown(manager)
        logger.info("GeminiCLIBridge: marked dirty -- will reinitialise on next run")

    def get_error_context(self) -> str:
        """Return recent Gemini CLI stderr lines for error reporting."""
        if not self._session_manager:
            return ""
        all_lines = self._session_manager.get_all_stderr(max_per_worker=10)
        if all_lines:
            return "Gemini CLI stderr:\n" + "\n".join(all_lines[-20:])
        return ""

    async def get_mcp_status(self) -> dict:
        """Get MCP server status from the written .gemini/settings.json.

        Unlike the Claude bridge (which uses an ephemeral SDK client to probe
        MCP servers), the Gemini CLI requires full auth just to run
        ``gemini mcp list``, so we read the settings file directly instead.
        The servers listed here are what the CLI will discover on its next run.
        """
        if self._mcp_status_cache is not None:
            return self._mcp_status_cache

        import json

        empty: dict = {"servers": [], "totalCount": 0}
        if not self._mcp_settings_path:
            # Don't cache if platform isn't ready yet — path will be set later
            return empty

        try:
            from pathlib import Path

            settings_path = Path(self._mcp_settings_path)
            if not settings_path.exists():
                return empty

            with open(settings_path) as f:
                settings = json.load(f)

            mcp_servers = settings.get("mcpServers", {})
            servers_list = []
            for name, config in mcp_servers.items():
                transport = "stdio"
                if config.get("httpUrl"):
                    transport = "http"
                elif config.get("url"):
                    transport = "sse"
                servers_list.append(
                    {
                        "name": name,
                        "displayName": name,
                        "status": "configured",
                        "transport": transport,
                        "tools": [],
                    }
                )

            result = {"servers": servers_list, "totalCount": len(servers_list)}
            self._mcp_status_cache = result
            return result
        except Exception as e:
            logger.error("Failed to get MCP status: %s", e, exc_info=True)
            return {"servers": [], "totalCount": 0, "error": str(e)}

    # ------------------------------------------------------------------
    # Properties
    # ------------------------------------------------------------------

    @property
    def context(self) -> RunnerContext | None:
        return self._context

    @property
    def configured_model(self) -> str:
        return self._configured_model

    @property
    def obs(self) -> Any:
        return self._obs

    # ------------------------------------------------------------------
    # Private: platform setup (lazy, called on first run)
    # ------------------------------------------------------------------

    async def _setup_platform(self) -> None:
        """Full platform setup: auth, workspace, observability."""
        # Session manager with state dir for session_id persistence across restarts
        if self._session_manager is None:
            state_dir = os.path.join(
                os.getenv("WORKSPACE_PATH", "/workspace"),
                os.getenv("RUNNER_STATE_DIR", ".gemini"),
            )
            self._session_manager = GeminiSessionManager(state_dir=state_dir)

        # Gemini-specific auth
        from ambient_runner.bridges.gemini_cli.auth import setup_gemini_cli_auth
        from ambient_runner.platform.auth import populate_runtime_credentials
        from ambient_runner.platform.workspace import resolve_workspace_paths

        model, api_key, use_vertex = await setup_gemini_cli_auth(self._context)

        # Populate credentials
        await populate_runtime_credentials(self._context)
        self._last_creds_refresh = time.monotonic()

        # Workspace paths
        cwd_path, add_dirs = resolve_workspace_paths(self._context)

        # Observability
        self._obs = await setup_bridge_observability(self._context, model)

        # MCP servers — write .gemini/settings.json so the CLI discovers them
        from ambient_runner.bridges.gemini_cli.mcp import setup_gemini_mcp
        from ambient_runner.bridges.gemini_cli.system_prompt import write_gemini_system_prompt

        mcp_settings_path = setup_gemini_mcp(self._context, cwd_path)

        # System prompt — write .gemini/system.md and set GEMINI_SYSTEM_MD=true.
        # Uses ${AgentSkills} / ${AvailableTools} substitution to preserve
        # Gemini's built-in instructions, then appends platform context.
        write_gemini_system_prompt(cwd_path)

        # Build include directories: platform-provided dirs (repos, workflows,
        # file-uploads) plus well-known workspace subdirs, excluding cwd itself.
        workspace = os.getenv("WORKSPACE_PATH", "/workspace")
        include_dirs = list(add_dirs) if add_dirs else []
        for subdir in ["repos", "artifacts", "file-uploads"]:
            d = os.path.join(workspace, subdir)
            if os.path.isdir(d) and d != cwd_path and d not in include_dirs:
                include_dirs.append(d)

        # Store results
        self._configured_model = model
        self._api_key = api_key
        self._use_vertex = use_vertex
        self._cwd_path = cwd_path
        self._include_directories = include_dirs
        self._mcp_settings_path = mcp_settings_path
