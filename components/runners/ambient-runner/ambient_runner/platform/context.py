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
        """Merge environment variables (explicit overrides win)."""
        self.environment = {**os.environ, **self.environment}

    def get_env(self, key: str, default: Optional[str] = None) -> Optional[str]:
        """Get an environment variable value."""
        return self.environment.get(key, default)

    def set_metadata(self, key: str, value: Any) -> None:
        """Set a metadata value."""
        self.metadata[key] = value

    def get_metadata(self, key: str, default: Any = None) -> Any:
        """Get a metadata value."""
        return self.metadata.get(key, default)
