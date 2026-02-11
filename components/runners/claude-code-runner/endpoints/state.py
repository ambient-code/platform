"""
Shared mutable state for all endpoint routers.

Centralises the globals that were previously scattered across main.py.
Endpoints import from here rather than reaching into main.
"""

import asyncio
from typing import Any, Dict, Optional

from context import RunnerContext

# --- Mutable server state ---

context: Optional[RunnerContext] = None
adapter = None                       # Current ClaudeAgentAdapter instance (persistent)
_obs = None                          # ObservabilityManager (or None)
_platform_ready = False              # One-time platform setup done?
_platform_info: Dict[str, Any] = {}  # cwd_path, add_dirs from setup_platform
_configured_model: str = ""          # Resolved model name
_first_run = True                    # Controls conversation continuation
_adapter_dirty = True                # True = adapter needs (re)building
_workflow_change_lock = asyncio.Lock()
