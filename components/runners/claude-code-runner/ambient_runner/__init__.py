"""
Ambient Runner SDK — reusable platform package for AG-UI agent runners.

Quick start::

    from ambient_runner import create_ambient_app
    from ambient_runner.bridges.claude import ClaudeBridge

    app = create_ambient_app(ClaudeBridge(), title="My Runner")

To build your own bridge, subclass ``PlatformBridge``::

    from ambient_runner import PlatformBridge, FrameworkCapabilities

    class MyBridge(PlatformBridge):
        def capabilities(self):
            return FrameworkCapabilities(framework="my-framework")
        async def run(self, input_data): ...
        async def interrupt(self, thread_id=None): ...
"""

from ambient_runner.app import add_ambient_endpoints, create_ambient_app, run_ambient_app
from ambient_runner.bridge import FrameworkCapabilities, PlatformBridge

__all__ = [
    "create_ambient_app",
    "run_ambient_app",
    "add_ambient_endpoints",
    "PlatformBridge",
    "FrameworkCapabilities",
]
