"""
ClaudeBridge — PlatformBridge implementation for the Claude Agent SDK.

Maps Ambient platform concepts to ``ClaudeAgentAdapter`` options and
handles the adapter lifecycle (create, run, interrupt).
"""

import logging
from typing import Any, AsyncIterator

from ag_ui.core import BaseEvent, RunAgentInput
from ag_ui_claude_sdk import ClaudeAgentAdapter

from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
    PlatformContext,
)

logger = logging.getLogger(__name__)


class ClaudeBridge(PlatformBridge):
    """Bridge between the Ambient platform and the Claude Agent SDK.

    Creates and manages a ``ClaudeAgentAdapter`` instance. The adapter
    is created once and reused across runs unless ``needs_rebuild()``
    returns True.
    """

    def __init__(self) -> None:
        self._adapter: ClaudeAgentAdapter | None = None
        self._last_ctx: PlatformContext | None = None

    def capabilities(self) -> FrameworkCapabilities:
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
            tracing="langfuse",
            session_persistence=True,
        )

    def create_adapter(self, ctx: PlatformContext) -> ClaudeAgentAdapter:
        """Build a ``ClaudeAgentAdapter`` from platform context."""
        options = self._build_options(ctx)

        adapter = ClaudeAgentAdapter(
            name="claude_code_runner",
            description="Ambient Code Platform Claude session",
            options=options,
        )

        self._adapter = adapter
        self._last_ctx = ctx
        logger.info(f"ClaudeBridge: adapter created (model={ctx.model})")
        return adapter

    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        """Run the Claude adapter and yield AG-UI events."""
        if self._adapter is None:
            raise RuntimeError("ClaudeBridge: adapter not created — call create_adapter() first")

        async for event in self._adapter.run(input_data):
            yield event

    async def interrupt(self) -> None:
        """Interrupt the current Claude SDK execution."""
        if self._adapter is None:
            raise RuntimeError("ClaudeBridge: no adapter to interrupt")
        await self._adapter.interrupt()

    def needs_rebuild(self, ctx: PlatformContext) -> bool:
        """Rebuild if CWD, MCP servers, or model changed."""
        if self._last_ctx is None:
            return True
        return (
            ctx.cwd_path != self._last_ctx.cwd_path
            or ctx.model != self._last_ctx.model
            or ctx.mcp_servers != self._last_ctx.mcp_servers
        )

    # ------------------------------------------------------------------
    # Private
    # ------------------------------------------------------------------

    @staticmethod
    def _build_options(ctx: PlatformContext) -> dict[str, Any]:
        """Build the options dict for ``ClaudeAgentAdapter``."""

        def _stderr_handler(line: str) -> None:
            logger.warning(f"[SDK stderr] {line.rstrip()}")

        options: dict[str, Any] = {
            "cwd": ctx.cwd_path,
            "permission_mode": "acceptEdits",
            "allowed_tools": ctx.allowed_tools,
            "mcp_servers": ctx.mcp_servers,
            "setting_sources": ["project"],
            "system_prompt": ctx.system_prompt,
            "include_partial_messages": True,
            "stderr": _stderr_handler,
        }

        if ctx.add_dirs:
            options["add_dirs"] = ctx.add_dirs

        if ctx.model:
            options["model"] = ctx.model

        # Optional max_tokens / temperature from environment
        max_tokens = ctx.environment.get("LLM_MAX_TOKENS") or ctx.environment.get("MAX_TOKENS")
        if max_tokens:
            try:
                options["max_tokens"] = int(max_tokens)
            except (ValueError, TypeError):
                pass

        temperature = ctx.environment.get("LLM_TEMPERATURE") or ctx.environment.get("TEMPERATURE")
        if temperature:
            try:
                options["temperature"] = float(temperature)
            except (ValueError, TypeError):
                pass

        if not ctx.first_run or ctx.is_resume:
            options["continue_conversation"] = True
            logger.info("ClaudeBridge: enabled continue_conversation")

        return options
