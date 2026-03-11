"""
Runner context — session information and environment for an AG-UI runner.

The ``RunnerContext`` is created once at startup and passed to the bridge
via ``set_context()``.  It provides typed access to environment variables
and a metadata store for cross-cutting state.
"""

import os
from dataclasses import dataclass, field
from typing import Any, Dict, Optional


@dataclass
class RunnerContext:
    """Context provided to runner adapters.

    Args:
        session_id: Unique identifier for this runner session.
        workspace_path: Absolute path to the workspace root directory.
        environment: Extra environment overrides (merged with ``os.environ``).
        metadata: Arbitrary key-value store for cross-cutting state.
    """

    session_id: str
    workspace_path: str
    environment: Dict[str, str] = field(default_factory=dict)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        """Store explicit overrides for precedence in get_env(); keep environment populated for backward compatibility."""
        self._overrides = dict(self.environment)
        self.environment = {**os.environ, **self.environment}

    def get_env(self, key: str, default: Optional[str] = None) -> Optional[str]:
        """Get an environment variable, with explicit overrides winning. Reads live from os.environ for non-overridden keys."""
        overrides = getattr(self, "_overrides", None)
        if overrides is None:
            return self.environment.get(key, default)
        if key in overrides:
            return overrides[key]
        return os.environ.get(key, default)

    def set_metadata(self, key: str, value: Any) -> None:
        """Set a metadata value."""
        self.metadata[key] = value

    def get_metadata(self, key: str, default: Any = None) -> Any:
        """Get a metadata value."""
        return self.metadata.get(key, default)
