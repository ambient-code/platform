"""
LangGraphBridge — PlatformBridge implementation for LangGraph.

Demonstrates that the PlatformBridge abstraction works with a fundamentally
different framework:

- No filesystem access (LangGraph agents are stateless functions)
- No native MCP support (tools are LangChain-style)
- Different tracing (LangSmith instead of Langfuse)
- No CWD concept (workspace context goes in system prompt only)

This bridge shows how a different framework plugs into the same Ambient
platform. The frontend capabilities system automatically hides UI panels
that don't apply.

Usage::

    from ambient_runner import create_ambient_app
    from ambient_runner.bridges.langgraph import LangGraphBridge

    app = create_ambient_app(LangGraphBridge(), title="LangGraph Runner")
"""

import logging
from typing import Any, AsyncIterator, Optional

from ag_ui.core import BaseEvent, RunAgentInput

from ambient_runner.bridge import FrameworkCapabilities, PlatformBridge
from ambient_runner.platform.context import RunnerContext

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
        self._context: RunnerContext | None = None

    # ------------------------------------------------------------------
    # PlatformBridge interface
    # ------------------------------------------------------------------

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

    def set_context(self, context: RunnerContext) -> None:
        self._context = context

    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        """Run the LangGraph adapter and yield AG-UI events.

        Lazily creates the adapter on first run.
        """
        if self._adapter is None:
            self._create_adapter()

        async for event in self._adapter.run(input_data):
            yield event

    async def interrupt(self, thread_id: Optional[str] = None) -> None:
        """Interrupt the current LangGraph execution."""
        if self._adapter is None:
            raise RuntimeError("LangGraphBridge: no adapter to interrupt")

        if hasattr(self._adapter, "interrupt"):
            await self._adapter.interrupt()
        else:
            logger.warning("LangGraphBridge: adapter does not support interrupt")

    @property
    def context(self) -> RunnerContext | None:
        return self._context

    # ------------------------------------------------------------------
    # Private
    # ------------------------------------------------------------------

    def _create_adapter(self) -> None:
        """Build the LangGraph AG-UI adapter from environment config."""
        try:
            from ag_ui_langgraph import LangGraphAgent
        except ImportError:
            raise RuntimeError(
                "LangGraphBridge requires ag_ui_langgraph. "
                "Install it: pip install ag-ui-langgraph"
            )

        env = self._context.environment if self._context else {}
        graph_url = env.get("LANGGRAPH_URL", "")
        graph_id = env.get("LANGGRAPH_GRAPH_ID", "agent")
        api_key = env.get("LANGSMITH_API_KEY", "")

        if not graph_url:
            raise RuntimeError("LANGGRAPH_URL must be set for LangGraph bridge")

        self._adapter = LangGraphAgent(
            url=graph_url,
            graph_id=graph_id,
            api_key=api_key,
        )
        logger.info(
            f"LangGraphBridge: adapter created (url={graph_url}, graph={graph_id})"
        )
