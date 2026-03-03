"""
AG-UI middleware for the Ambient Runner SDK.

Middleware wraps the adapter's event stream to add platform concerns
(tracing, developer events) without modifying the adapter itself.
"""

from ambient_runner.middleware.developer_events import emit_developer_message
from ambient_runner.middleware.tracing import tracing_middleware

__all__ = ["tracing_middleware", "emit_developer_message"]
