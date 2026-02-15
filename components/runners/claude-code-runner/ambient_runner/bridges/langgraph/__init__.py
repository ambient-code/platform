"""
LangGraph bridge for the Ambient Runner SDK.

Usage::

    from ambient_runner.bridges.langgraph import LangGraphBridge

    app = create_ambient_app(LangGraphBridge(), title="LangGraph Runner")
"""

from ambient_runner.bridges.langgraph.bridge import LangGraphBridge

__all__ = ["LangGraphBridge"]
