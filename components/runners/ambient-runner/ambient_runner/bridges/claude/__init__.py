"""
Claude Agent SDK bridge for the Ambient Runner SDK.

Usage::

    from ambient_runner.bridges.claude import ClaudeBridge

    app = create_ambient_app(ClaudeBridge(), title="Claude Runner")
"""

# Apply mock patch early if needed (before any imports of claude_agent_sdk)
from ambient_runner.bridges.claude import mock_patch  # noqa: F401

from ambient_runner.bridges.claude.bridge import ClaudeBridge
from ambient_runner.bridges.claude.bridge_v2 import ClaudeBridgeV2

__all__ = ["ClaudeBridge", "ClaudeBridgeV2"]
