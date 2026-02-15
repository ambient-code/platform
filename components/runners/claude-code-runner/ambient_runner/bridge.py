"""
PlatformBridge — abstract base class for framework-specific bridges.

Each framework (Claude Agent SDK, LangGraph, etc.) provides a bridge
implementation that handles the full lifecycle of an AG-UI runner:
context setup, adapter creation, request handling, and shutdown.

The bridge is the single integration point between the Ambient platform
and any AG-UI-compatible framework adapter.

Minimal implementation example::

    class MyBridge(PlatformBridge):
        def capabilities(self) -> FrameworkCapabilities:
            return FrameworkCapabilities(framework="my-framework")

        async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
            yield RunStartedEvent(...)
            yield TextMessageStartEvent(...)
            ...

        async def interrupt(self, thread_id=None) -> None:
            pass
"""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any, AsyncIterator, Optional

from ag_ui.core import BaseEvent, RunAgentInput

from ambient_runner.platform.context import RunnerContext


@dataclass
class FrameworkCapabilities:
    """Declares what a framework adapter supports.

    Used by the ``/capabilities`` endpoint and the frontend to determine
    which UI panels and features to show.
    """

    framework: str
    agent_features: list[str] = field(default_factory=list)
    file_system: bool = False
    mcp: bool = False
    tracing: Optional[str] = None
    session_persistence: bool = False


class PlatformBridge(ABC):
    """Abstract bridge between the Ambient platform and a framework adapter.

    **Required** (must implement):

    - ``capabilities()`` — declares what the framework supports
    - ``run()`` — handles a single AG-UI run request, yielding events
    - ``interrupt()`` — interrupts the current execution

    **Lifecycle** (override as needed):

    - ``set_context()`` — receives the ``RunnerContext`` at startup
    - ``shutdown()`` — called on server shutdown for cleanup
    - ``mark_dirty()`` — called when repos/workflows change at runtime
    - ``get_mcp_status()`` — returns MCP server diagnostics
    - ``get_error_context()`` — returns extra error info for failed runs
    """

    # ------------------------------------------------------------------
    # Required (abstract)
    # ------------------------------------------------------------------

    @abstractmethod
    def capabilities(self) -> FrameworkCapabilities:
        """Return the capabilities of this framework."""
        ...

    @abstractmethod
    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        """Run the adapter and yield AG-UI events.

        The bridge handles all internal lifecycle: lazy platform setup,
        adapter creation, session management, and tracing middleware.

        Args:
            input_data: The AG-UI run input.

        Yields:
            AG-UI ``BaseEvent`` instances.
        """
        ...

    @abstractmethod
    async def interrupt(self, thread_id: Optional[str] = None) -> None:
        """Interrupt the current run.

        Args:
            thread_id: Optional thread to interrupt. If ``None``, interrupts
                the default/most recent thread.
        """
        ...

    # ------------------------------------------------------------------
    # Lifecycle (override in subclasses as needed)
    # ------------------------------------------------------------------

    def set_context(self, context: RunnerContext) -> None:
        """Store the runner context for later use.

        Called by the platform lifespan before any requests are served.
        Override to capture the context for use in ``run()``.
        """
        pass

    async def shutdown(self) -> None:
        """Graceful shutdown — release resources, persist state.

        Called when the FastAPI app is shutting down.
        """
        pass

    def mark_dirty(self) -> None:
        """Signal that the adapter should be rebuilt on the next ``run()``.

        Called by the repos and workflow endpoints when the workspace
        changes at runtime.
        """
        pass

    async def get_mcp_status(self) -> dict:
        """Return MCP server connection diagnostics.

        Default: empty result. Override for frameworks that support MCP.
        """
        return {"servers": [], "totalCount": 0}

    def get_error_context(self) -> str:
        """Return extra context for error reporting (e.g. stderr output).

        Called by the run endpoint when the event stream raises an
        exception. The returned string is appended to the error message
        in the ``RunErrorEvent``.

        Default: empty string (no extra context).
        """
        return ""

    # ------------------------------------------------------------------
    # Properties (override to expose state to endpoints)
    # ------------------------------------------------------------------

    @property
    def context(self) -> Optional[RunnerContext]:
        """The current ``RunnerContext``, or ``None`` before ``set_context()``."""
        return None

    @property
    def configured_model(self) -> str:
        """The resolved model name (e.g. ``'claude-sonnet-4-5'``)."""
        return ""

    @property
    def obs(self) -> Any:
        """The observability manager, or ``None`` if not configured."""
        return None
