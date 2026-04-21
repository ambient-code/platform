# components/runners/ambient-runner/ambient_runner/bridges/cursor_cli/auth.py
"""Cursor CLI authentication — API key setup."""

import logging

from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


async def setup_cursor_cli_auth(context: RunnerContext) -> tuple[str, str]:
    """Set up Cursor CLI authentication from environment.

    Cursor CLI authenticates via CURSOR_API_KEY env var.
    No Vertex AI support — Cursor routes through its own subscription.

    Returns:
        (model, api_key)
    """
    from ag_ui_cursor_cli.config import DEFAULT_MODEL

    model = context.get_env("LLM_MODEL", DEFAULT_MODEL).strip()

    # Strip the "cursor:" prefix if present — the registry uses
    # namespaced model IDs (e.g., "cursor:claude-sonnet-4-6") but the
    # CLI --model flag expects the raw name ("claude-sonnet-4-6").
    if model.startswith("cursor:"):
        model = model[len("cursor:"):]

    api_key = context.get_env("CURSOR_API_KEY", "").strip()

    if api_key:
        logger.info("Cursor CLI: using API key (model=%s)", model)
    else:
        logger.warning("Cursor CLI: no CURSOR_API_KEY set — CLI may fail")

    return model, api_key
