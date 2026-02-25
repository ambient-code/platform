"""
ClaudeBridge — full-lifecycle PlatformBridge for the Claude Agent SDK.

Owns the entire Claude session lifecycle:
- Platform setup (auth, workspace, MCP, observability)
- Adapter creation and caching
- Session worker management (persistent SDK clients)
- Tracing middleware integration
- Interrupt and graceful shutdown
"""

import asyncio
import logging
import os
from typing import Any, AsyncIterator, Optional

from ag_ui.core import BaseEvent, RunAgentInput
from ag_ui_claude_sdk import ClaudeAgentAdapter

from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
)
from ambient_runner.bridges.claude.session import SessionManager
from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)

# Maximum stderr lines kept in ring buffer for error reporting
_MAX_STDERR_LINES = 50


class ClaudeBridge(PlatformBridge):
    """Bridge between the Ambient platform and the Claude Agent SDK.

    Handles lazy platform initialisation on first ``run()`` call, builds
    and caches the ``ClaudeAgentAdapter``, manages persistent
    ``SessionWorker`` instances, and wraps the event stream with
    Langfuse tracing.
    """

    def __init__(self) -> None:
        self._adapter: ClaudeAgentAdapter | None = None
        self._session_manager: SessionManager | None = None
        self._obs: Any = None
        self._context: RunnerContext | None = None

        # Platform state (populated by _setup_platform)
        self._ready: bool = False
        self._first_run: bool = True
        self._configured_model: str = ""
        self._cwd_path: str = ""
        self._add_dirs: list[str] = []
        self._mcp_servers: dict = {}
        self._allowed_tools: list[str] = []
        self._system_prompt: dict = {}
        self._stderr_lines: list[str] = []

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
            framework="claude-agent-sdk",
            agent_features=[
                "agentic_chat",
                "backend_tool_rendering",
                "shared_state",
                "human_in_the_loop",
                "thinking",
            ],
            file_system=True,
            mcp=True,
            tracing="langfuse" if has_tracing else None,
            session_persistence=True,
        )

    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        """Full run lifecycle: lazy setup → adapter → session worker → tracing."""
        # 1. Lazy platform setup
        await self._ensure_ready()

        # 2. Ensure adapter exists
        self._ensure_adapter()

        # 3. Extract user message for worker and observability
        from ag_ui_claude_sdk.utils import process_messages

        user_msg, _ = process_messages(input_data)

        # 4. Get or create session worker for this thread
        thread_id = input_data.thread_id or self._context.session_id
        api_key = os.getenv("ANTHROPIC_API_KEY", "")
        sdk_options = self._adapter.build_options(input_data, thread_id=thread_id)
        worker = await self._session_manager.get_or_create(
            thread_id, sdk_options, api_key
        )

        # 5. Run adapter with message stream, wrapped in tracing
        session_label = self._session_manager.get_session_id(thread_id) or thread_id
        async with self._session_manager.get_lock(thread_id):
            message_stream = worker.query(user_msg, session_id=session_label)

            from ambient_runner.middleware import tracing_middleware

            wrapped_stream = tracing_middleware(
                self._adapter.run(input_data, message_stream=message_stream),
                obs=self._obs,
                model=self._configured_model,
                prompt=user_msg,
            )

            async for event in wrapped_stream:
                yield event

        self._first_run = False

    async def interrupt(self, thread_id: Optional[str] = None) -> None:
        """Interrupt the running session for a given thread."""
        if not self._session_manager:
            raise RuntimeError("No active session manager")

        tid = thread_id or (self._context.session_id if self._context else None)
        if not tid:
            raise RuntimeError("No thread_id available")

        worker = self._session_manager.get_existing(tid)
        if not worker:
            raise RuntimeError(f"No active session for thread {tid}")

        logger.info(f"Interrupt request for thread={tid}")
        await worker.interrupt()

    # ------------------------------------------------------------------
    # Lifecycle methods
    # ------------------------------------------------------------------

    def set_context(self, context: RunnerContext) -> None:
        """Store the runner context (called from lifespan)."""
        self._context = context

    async def shutdown(self) -> None:
        """Graceful shutdown: persist sessions, finalise tracing."""
        if self._session_manager:
            await self._session_manager.shutdown()
        if self._obs:
            await self._obs.finalize()
        logger.info("ClaudeBridge: shutdown complete")

    def mark_dirty(self) -> None:
        """Signal adapter rebuild on next run (repo/workflow change).

        Destroys existing session workers so the new MCP server
        configuration (e.g. updated correction tool targets) is applied
        to the CLI process on the next run.  Conversation state is
        preserved via the CLI's ``--resume`` mechanism.
        """
        self._ready = False
        self._first_run = True
        self._adapter = None
        if self._session_manager:
            manager = self._session_manager
            self._session_manager = None
            try:
                loop = asyncio.get_running_loop()
                future = asyncio.ensure_future(manager.shutdown())
                future.add_done_callback(
                    lambda f: logger.warning(
                        "mark_dirty: session_manager shutdown error: %s", f.exception()
                    )
                    if f.exception()
                    else None
                )
            except RuntimeError:
                # No running loop — safe to block
                try:
                    asyncio.run(manager.shutdown())
                except Exception as e:
                    logger.warning("mark_dirty: session_manager shutdown error: %s", e)
        logger.info("ClaudeBridge: marked dirty — will reinitialise on next run")

    def get_error_context(self) -> str:
        """Return recent Claude CLI stderr lines for error reporting."""
        if self._stderr_lines:
            recent = self._stderr_lines[-10:]
            return "Claude CLI stderr:\n" + "\n".join(recent)
        return ""

    async def get_mcp_status(self) -> dict:
        """Get MCP server status via an ephemeral SDK client."""
        if not self._context:
            return {"servers": [], "totalCount": 0, "message": "Context not initialized"}

        try:
            from claude_agent_sdk import ClaudeAgentOptions, ClaudeSDKClient

            from ambient_runner.platform.config import load_mcp_config

            workspace_path = self._context.workspace_path or "/workspace"
            active_workflow_url = os.getenv("ACTIVE_WORKFLOW_GIT_URL", "").strip()
            cwd_path = workspace_path

            if active_workflow_url:
                workflow_name = active_workflow_url.split("/")[-1].removesuffix(".git")
                workflow_path = os.path.join(workspace_path, "workflows", workflow_name)
                if os.path.exists(workflow_path):
                    cwd_path = workflow_path

            mcp_servers = load_mcp_config(self._context, cwd_path) or {}

            options = ClaudeAgentOptions(
                cwd=cwd_path,
                permission_mode="acceptEdits",
                mcp_servers=mcp_servers,
            )

            client = ClaudeSDKClient(options=options)
            try:
                logger.info("MCP Status: Connecting ephemeral SDK client...")
                await client.connect()

                sdk_status = await client.get_mcp_status()

                raw_servers = []
                if isinstance(sdk_status, dict):
                    raw_servers = sdk_status.get("mcpServers", [])
                elif isinstance(sdk_status, list):
                    raw_servers = sdk_status

                servers_list = []
                for srv in raw_servers:
                    if not isinstance(srv, dict):
                        continue
                    server_info = srv.get("serverInfo") or {}
                    raw_tools = srv.get("tools") or []
                    tools = [
                        {
                            "name": t.get("name", ""),
                            "annotations": {
                                k: v
                                for k, v in (t.get("annotations") or {}).items()
                            },
                        }
                        for t in raw_tools
                        if isinstance(t, dict)
                    ]
                    servers_list.append(
                        {
                            "name": srv.get("name", ""),
                            "displayName": server_info.get(
                                "name", srv.get("name", "")
                            ),
                            "status": srv.get("status", "unknown"),
                            "version": server_info.get("version", ""),
                            "tools": tools,
                        }
                    )

                return {"servers": servers_list, "totalCount": len(servers_list)}
            finally:
                logger.info("MCP Status: Disconnecting ephemeral SDK client...")
                await client.disconnect()

        except Exception as e:
            logger.error(f"Failed to get MCP status: {e}", exc_info=True)
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

    @property
    def session_manager(self) -> SessionManager | None:
        return self._session_manager

    # ------------------------------------------------------------------
    # Private: platform setup (lazy, called on first run)
    # ------------------------------------------------------------------

    async def _ensure_ready(self) -> None:
        """Run one-time platform setup if not already done."""
        if self._ready:
            return
        if not self._context:
            raise RuntimeError("Context not set — call set_context() first")
        await self._setup_platform()
        self._ready = True
        logger.info(
            f"Platform ready — model: {self._configured_model}, "
            f"cwd: {self._cwd_path}"
        )

    async def _setup_platform(self) -> None:
        """Full platform setup: auth, workspace, MCP, observability."""
        # Session manager
        if self._session_manager is None:
            self._session_manager = SessionManager()

        # Claude-specific auth
        from ambient_runner.bridges.claude.auth import setup_sdk_authentication
        from ambient_runner.platform.auth import populate_runtime_credentials
        from ambient_runner.platform.workspace import resolve_workspace_paths, validate_prerequisites

        await validate_prerequisites(self._context)
        _api_key, _use_vertex, configured_model = await setup_sdk_authentication(
            self._context
        )
        await populate_runtime_credentials(self._context)

        # Workspace paths
        cwd_path, add_dirs = resolve_workspace_paths(self._context)

        # Observability (before MCP so rubric tool can access it)
        await self._setup_observability(configured_model)

        # MCP servers
        from ambient_runner.bridges.claude.mcp import (
            build_allowed_tools,
            build_mcp_servers,
            log_auth_status,
        )

        mcp_servers = build_mcp_servers(self._context, cwd_path, self._obs)
        log_auth_status(mcp_servers)
        allowed_tools = build_allowed_tools(mcp_servers)

        # System prompt
        from ambient_runner.bridges.claude.prompts import build_sdk_system_prompt

        system_prompt = build_sdk_system_prompt(
            self._context.workspace_path, cwd_path
        )

        # Store results
        self._configured_model = configured_model
        self._cwd_path = cwd_path
        self._add_dirs = add_dirs
        self._mcp_servers = mcp_servers
        self._allowed_tools = allowed_tools
        self._system_prompt = system_prompt

    async def _setup_observability(self, configured_model: str) -> None:
        """Initialise Langfuse observability (best-effort)."""
        try:
            from ambient_runner.observability import ObservabilityManager

            from ambient_runner.platform.auth import sanitize_user_context

            raw_user_id = os.getenv("USER_ID", "").strip()
            raw_user_name = os.getenv("USER_NAME", "").strip()
            user_id, user_name = sanitize_user_context(raw_user_id, raw_user_name)

            obs = ObservabilityManager(
                session_id=self._context.session_id,
                user_id=user_id,
                user_name=user_name,
            )
            await obs.initialize(
                prompt="(pending)",
                namespace=self._context.get_env(
                    "AGENTIC_SESSION_NAMESPACE", "unknown"
                ),
                model=configured_model,
            )
            self._obs = obs
        except Exception as e:
            logger.warning(f"Failed to initialize observability: {e}")

    # ------------------------------------------------------------------
    # Private: adapter lifecycle
    # ------------------------------------------------------------------

    def _ensure_adapter(self) -> None:
        """Build or reuse the ClaudeAgentAdapter."""
        if self._adapter is not None:
            return

        self._stderr_lines.clear()

        def _stderr_handler(line: str) -> None:
            stripped = line.rstrip()
            logger.warning(f"[SDK stderr] {stripped}")
            self._stderr_lines.append(stripped)
            if len(self._stderr_lines) > _MAX_STDERR_LINES:
                self._stderr_lines.pop(0)

        options: dict[str, Any] = {
            "cwd": self._cwd_path,
            "permission_mode": "acceptEdits",
            "allowed_tools": self._allowed_tools,
            "mcp_servers": self._mcp_servers,
            "setting_sources": ["project"],
            "system_prompt": self._system_prompt,
            "include_partial_messages": True,
            "stderr": _stderr_handler,
        }

        if self._add_dirs:
            options["add_dirs"] = self._add_dirs
        if self._configured_model:
            options["model"] = self._configured_model

        adapter = ClaudeAgentAdapter(
            name="claude_code_runner",
            description="Ambient Code Platform Claude session",
            options=options,
        )
        # Attach stderr buffer so error handler can read it
        adapter._stderr_lines = self._stderr_lines  # type: ignore[attr-defined]
        self._adapter = adapter
        logger.info("Adapter built (persistent, will be reused across runs)")

