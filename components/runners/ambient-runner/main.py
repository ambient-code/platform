"""Ambient Runner — polymorphic AG-UI server."""

from __future__ import annotations

import importlib
import logging
import os
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ambient_runner.bridge import PlatformBridge

os.umask(0o022)

logger = logging.getLogger(__name__)

RUNNER_TYPE = os.getenv("RUNNER_TYPE", "claude-agent-sdk").strip().lower()

BRIDGE_REGISTRY: dict[str, tuple[str, str]] = {
    "claude-agent-sdk": ("ambient_runner.bridges.claude", "ClaudeBridge"),
    "gemini-cli": ("ambient_runner.bridges.gemini_cli", "GeminiCLIBridge"),
    "langgraph": ("ambient_runner.bridges.langgraph", "LangGraphBridge"),
}


def _load_bridge() -> "PlatformBridge":
    if RUNNER_TYPE not in BRIDGE_REGISTRY:
        raise ValueError(
            f"Unknown RUNNER_TYPE={RUNNER_TYPE!r}. Available: {sorted(BRIDGE_REGISTRY)}"
        )
    module_path, class_name = BRIDGE_REGISTRY[RUNNER_TYPE]
    module = importlib.import_module(module_path)
    bridge_cls = getattr(module, class_name)
    logger.info("Loading bridge: %s from %s", class_name, module_path)
    return bridge_cls()


from ambient_runner import create_ambient_app, run_ambient_app  # noqa: E402

app = create_ambient_app(_load_bridge(), title="Ambient Runner AG-UI Server")

if __name__ == "__main__":
    run_ambient_app(app)
