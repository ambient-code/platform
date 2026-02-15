"""
Claude Agent SDK bridge for the Ambient Runner SDK.

Usage::

    from ambient_runner.bridges.claude import ClaudeBridge

    app = create_ambient_app(ClaudeBridge(), title="Claude Runner")
"""

from ambient_runner.bridges.claude.bridge import ClaudeBridge

__all__ = ["ClaudeBridge"]
