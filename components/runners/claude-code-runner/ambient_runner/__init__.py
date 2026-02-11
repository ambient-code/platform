"""
Ambient Runner SDK — reusable platform package for AG-UI agent runners.

Public API::

    from ambient_runner import add_ambient_endpoints, PlatformBridge
    from ambient_runner.bridge import PlatformContext, FrameworkCapabilities
"""

from ambient_runner.app import add_ambient_endpoints
from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
    PlatformContext,
)

__all__ = [
    "add_ambient_endpoints",
    "PlatformBridge",
    "PlatformContext",
    "FrameworkCapabilities",
]
