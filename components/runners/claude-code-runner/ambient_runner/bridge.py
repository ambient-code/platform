"""
PlatformBridge — abstract base class for framework-specific bridges.

Each framework (Claude Agent SDK, LangGraph, Cursor SDK, etc.) provides a
bridge implementation that translates platform concepts into framework
config and returns a ready-to-use AG-UI adapter.

The bridge is the single integration point between the Ambient platform
and any AG-UI-compatible framework adapter.
"""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any, AsyncIterator, Optional

from ag_ui.core import BaseEvent, RunAgentInput


@dataclass
class PlatformContext:
    """Platform context passed to the bridge.

    Contains all resolved platform state: auth credentials, workspace paths,
    MCP servers, system prompts, etc. The bridge maps these into framework-
    specific configuration.
    """

    session_id: str
    workspace_path: str
    cwd_path: str = ""
    add_dirs: list[str] = field(default_factory=list)
    model: str = ""
    mcp_servers: dict[str, Any] = field(default_factory=dict)
    allowed_tools: list[str] = field(default_factory=list)
    system_prompt: dict[str, Any] = field(default_factory=dict)
    first_run: bool = True
    is_resume: bool = False
    environment: dict[str, str] = field(default_factory=dict)
    extra: dict[str, Any] = field(default_factory=dict)


@dataclass
class FrameworkCapabilities:
    """Declares what a framework adapter supports.

    Used by the capabilities endpoint and the frontend to determine which
    UI panels and features to show.
    """

    framework: str
    agent_features: list[str] = field(default_factory=list)
    file_system: bool = False
    mcp: bool = False
    tracing: Optional[str] = None
    session_persistence: bool = False


class PlatformBridge(ABC):
    """Abstract bridge between the Ambient platform and a framework adapter.

    Subclasses must implement:
    - ``capabilities()`` — declares what the framework supports
    - ``create_adapter()`` — creates an AG-UI adapter from platform context
    - ``run()`` — runs the adapter and yields AG-UI events
    - ``interrupt()`` — interrupts the current run
    """

    @abstractmethod
    def capabilities(self) -> FrameworkCapabilities:
        """Return the capabilities of this framework."""
        ...

    @abstractmethod
    def create_adapter(self, ctx: PlatformContext) -> Any:
        """Create the framework's AG-UI adapter from platform context.

        Args:
            ctx: Resolved platform context with all config.

        Returns:
            An AG-UI adapter instance (framework-specific type).
        """
        ...

    @abstractmethod
    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        """Run the adapter and yield AG-UI events.

        Args:
            input_data: The AG-UI run input.

        Yields:
            AG-UI ``BaseEvent`` instances.
        """
        ...

    @abstractmethod
    async def interrupt(self) -> None:
        """Interrupt the current run."""
        ...

    def needs_rebuild(self, ctx: PlatformContext) -> bool:
        """Return True if the adapter should be rebuilt for a new context.

        Default: always returns False (adapter is reused).
        Override for frameworks that need rebuilding on config changes.
        """
        return False
