"""
LangGraphBridge — PlatformBridge implementation for LangGraph.

Validates that the PlatformBridge abstraction works with a fundamentally
different framework:

- No filesystem access (LangGraph agents are stateless functions)
- No native MCP support (tools are LangChain-style)
- Different tracing (LangSmith instead of Langfuse)
- No CWD concept (workspace context goes in system prompt only)

This bridge demonstrates that a different framework can plug into the
same Ambient platform with different capabilities. The frontend
capabilities system automatically hides UI panels that don't apply.
"""

import logging
from typing import Any, AsyncIterator

from ag_ui.core import BaseEvent, RunAgentInput

from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
    PlatformContext,
)

logger = logging.getLogger(__name__)


class LangGraphBridge(PlatformBridge):
    """Bridge between the Ambient platform and LangGraph.

    Requires ``ag_ui_langgraph`` to be installed. The adapter translates
    LangGraph's graph execution into AG-UI events.

    This bridge differs from ClaudeBridge in several ways:
    - ``file_system=False`` — LangGraph agents don't have filesystem access
    - ``mcp=False`` — no native MCP; tools are defined in the graph
    - ``tracing="langsmith"`` — uses LangSmith instead of Langfuse
    - No CWD, add_dirs, or allowed_tools
    """

    def __init__(self) -> None:
        self._adapter: Any = None
        self._last_ctx: PlatformContext | None = None

    def capabilities(self) -> FrameworkCapabilities:
        return FrameworkCapabilities(
            framework="langgraph",
            agent_features=[
                "agentic_chat",
                "shared_state",
                "human_in_the_loop",
            ],
            file_system=False,
            mcp=False,
            tracing="langsmith",
            session_persistence=False,
        )

    def create_adapter(self, ctx: PlatformContext) -> Any:
        """Build a LangGraph AG-UI adapter from platform context.

        The adapter wraps a LangGraph ``CompiledGraph`` and translates
        its execution into AG-UI events.
        """
        try:
            from ag_ui_langgraph import LangGraphAgent
        except ImportError:
            raise RuntimeError(
                "LangGraphBridge requires ag_ui_langgraph. "
                "Install it: pip install ag-ui-langgraph"
            )

        # LangGraph config from platform context
        graph_url = ctx.environment.get("LANGGRAPH_URL", "")
        graph_id = ctx.environment.get("LANGGRAPH_GRAPH_ID", "agent")
        api_key = ctx.environment.get("LANGSMITH_API_KEY", "")

        if not graph_url:
            raise RuntimeError("LANGGRAPH_URL must be set for LangGraph bridge")

        self._adapter = LangGraphAgent(
            url=graph_url,
            graph_id=graph_id,
            api_key=api_key,
        )

        self._last_ctx = ctx
        logger.info(f"LangGraphBridge: adapter created (url={graph_url}, graph={graph_id})")
        return self._adapter

    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        """Run the LangGraph adapter and yield AG-UI events."""
        if self._adapter is None:
            raise RuntimeError("LangGraphBridge: adapter not created")

        async for event in self._adapter.run(input_data):
            yield event

    async def interrupt(self) -> None:
        """Interrupt the current LangGraph execution."""
        if self._adapter is None:
            raise RuntimeError("LangGraphBridge: no adapter to interrupt")

        if hasattr(self._adapter, "interrupt"):
            await self._adapter.interrupt()
        else:
            logger.warning("LangGraphBridge: adapter does not support interrupt")

    def needs_rebuild(self, ctx: PlatformContext) -> bool:
        """Rebuild if the graph URL or graph ID changed."""
        if self._last_ctx is None:
            return True
        return (
            ctx.environment.get("LANGGRAPH_URL") != self._last_ctx.environment.get("LANGGRAPH_URL")
            or ctx.environment.get("LANGGRAPH_GRAPH_ID") != self._last_ctx.environment.get("LANGGRAPH_GRAPH_ID")
        )
