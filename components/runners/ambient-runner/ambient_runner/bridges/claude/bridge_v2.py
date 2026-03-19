"""
ClaudeBridge (v2) — simplified bridge using upstream ag-ui-claude-sdk.

Key changes from v1:
- Uses upstream ClaudeAgentAdapter with built-in worker management
- Removes custom SessionManager (upstream handles session workers)
- Simplified run() method - adapter manages message streaming
- MockClient support via ANTHROPIC_API_KEY environment variable
"""

import logging
import os
import time
from typing import Any, AsyncIterator, Optional

from ag_ui.core import BaseEvent, RunAgentInput
from ag_ui_claude_sdk import ClaudeAgentAdapter

from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
    setup_bridge_observability,
)
from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


class ClaudeBridgeV2(PlatformBridge):
    """Simplified bridge using upstream ag-ui-claude-sdk package.

    Platform responsibilities:
    - Auth setup (API key, Vertex)
    - Workspace paths and MCP configuration
    - System prompts
    - Observability integration

    Adapter responsibilities (upstream):
    - Session worker lifecycle
    - Message streaming
    - AG-UI event translation
    """

    def __init__(self) -> None:
        super().__init__()
        self._adapter: ClaudeAgentAdapter | None = None
        self._obs: Any = None

        # Platform state (populated by _setup_platform)
        self._first_run: bool = True
        self._configured_model: str = ""
        self._cwd_path: str = ""
        self._add_dirs: list[str] = []
        self._mcp_servers: dict = {}
        self._allowed_tools: list[str] = []
        self._system_prompt: dict = {}

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
        """Simplified run: platform setup → adapter.run() → tracing wrapper."""
        # 1. Lazy platform setup
        await self._ensure_ready()
        await self._refresh_credentials_if_stale()

        # 2. Ensure adapter exists
        self._ensure_adapter()

        # 3. Run adapter (handles worker management internally)
        from ambient_runner.middleware import tracing_middleware

        # Extract user message for observability
        from ag_ui_claude_sdk.utils import process_messages
        user_msg, _ = process_messages(input_data)

        wrapped_stream = tracing_middleware(
            self._adapter.run(input_data),
            obs=self._obs,
            model=self._configured_model,
            prompt=user_msg,
        )

        async for event in wrapped_stream:
            yield event

        self._first_run = False

    async def interrupt(self, thread_id: Optional[str] = None) -> None:
        """Interrupt the running session."""
        if not self._adapter:
            raise RuntimeError("No active adapter")

        # Upstream adapter's interrupt needs thread_id
        tid = thread_id or (self._context.session_id if self._context else None)
        if not tid:
            raise RuntimeError("No thread_id available")

        logger.info(f"Interrupt request for thread={tid}")
        await self._adapter.interrupt(thread_id=tid)

        # Record interrupt in observability
        if self._obs:
            self._obs.record_interrupt()

    # ------------------------------------------------------------------
    # Lifecycle methods
    # ------------------------------------------------------------------

    async def shutdown(self) -> None:
        """Graceful shutdown."""
        if self._adapter:
            await self._adapter.shutdown()
        if self._obs:
            await self._obs.finalize()
        logger.info("ClaudeBridgeV2: shutdown complete")

    def mark_dirty(self) -> None:
        """Signal adapter rebuild on next run (repo/workflow change)."""
        self._ready = False
        self._first_run = True
        self._adapter = None
        logger.info("ClaudeBridgeV2: marked dirty — will reinitialize on next run")

    def get_error_context(self) -> str:
        """Return error context (stderr from adapter if available)."""
        if self._adapter and hasattr(self._adapter, "_stderr_lines"):
            recent = self._adapter._stderr_lines[-10:]
            if recent:
                return "Claude CLI stderr:\n" + "\n".join(recent)
        return ""

    async def get_mcp_status(self) -> dict:
        """Get MCP server status via ephemeral SDK client."""
        # TODO: Port from bridge.py or use adapter's built-in method if available
        return {
            "servers": [],
            "totalCount": 0,
            "message": "Not yet implemented in v2 bridge",
        }

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
        """Full platform setup: auth, workspace, MCP, observability."""
        from ambient_runner.bridges.claude.auth import setup_sdk_authentication
        from ambient_runner.platform.auth import (
            populate_mcp_server_credentials,
            populate_runtime_credentials,
        )
        from ambient_runner.platform.workspace import (
            resolve_workspace_paths,
            validate_prerequisites,
        )

        await validate_prerequisites(self._context)
        _api_key, _use_vertex, configured_model = await setup_sdk_authentication(
            self._context
        )

        # Populate credentials before building system prompt
        await populate_runtime_credentials(self._context)
        await populate_mcp_server_credentials(self._context)
        self._last_creds_refresh = time.monotonic()

        # Workspace paths
        cwd_path, add_dirs = resolve_workspace_paths(self._context)
        if add_dirs:
            os.environ["CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD"] = "1"

        # Observability (before MCP so tools can access it)
        self._obs = await setup_bridge_observability(self._context, configured_model)

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

        system_prompt = build_sdk_system_prompt(self._context.workspace_path, cwd_path)

        # Store results
        self._configured_model = configured_model
        self._cwd_path = cwd_path
        self._add_dirs = add_dirs
        self._mcp_servers = mcp_servers
        self._allowed_tools = allowed_tools
        self._system_prompt = system_prompt

    # ------------------------------------------------------------------
    # Private: adapter lifecycle
    # ------------------------------------------------------------------

    def _ensure_adapter(self) -> None:
        """Build or reuse the ClaudeAgentAdapter (upstream version)."""
        if self._adapter is not None:
            return

        stderr_lines = []

        def _stderr_handler(line: str) -> None:
            stripped = line.rstrip()
            logger.warning(f"[SDK stderr] {stripped}")
            stderr_lines.append(stripped)
            if len(stderr_lines) > 50:
                stderr_lines.pop(0)

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

        # Upstream adapter with built-in worker management
        adapter = ClaudeAgentAdapter(
            name="claude_code_runner",
            description="Ambient Code Platform Claude session",
            options=options,
        )

        # Attach stderr buffer for error reporting
        adapter._stderr_lines = stderr_lines  # type: ignore[attr-defined]

        self._adapter = adapter
        logger.info("ClaudeAgentAdapter (upstream) initialized")
