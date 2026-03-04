"""
Claude-specific authentication — Vertex AI and Anthropic API key setup.

Framework-agnostic credential fetching lives in ``ambient_runner.platform.auth``.
This module adds Claude Agent SDK-specific concerns:
- Vertex AI model mapping and credential setup
- SDK authentication environment variable population
"""

import logging
import os
from pathlib import Path

from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Vertex AI model mapping
# ---------------------------------------------------------------------------

VERTEX_MODEL_MAP: dict[str, str] = {
    "claude-opus-4-6": "claude-opus-4-6@default",
    "claude-opus-4-5": "claude-opus-4-5@20251101",
    "claude-opus-4-1": "claude-opus-4-1@20250805",
    "claude-sonnet-4-5": "claude-sonnet-4-5@20250929",
    "claude-haiku-4-5": "claude-haiku-4-5@20251001",
}


def map_to_vertex_model(model: str) -> str:
    """Map Anthropic API model names to Vertex AI model names."""
    return VERTEX_MODEL_MAP.get(model, model)


async def setup_vertex_credentials(context: RunnerContext) -> dict:
    """Set up Google Cloud Vertex AI credentials from service account."""
    service_account_path = context.get_env("GOOGLE_APPLICATION_CREDENTIALS", "").strip()
    project_id = context.get_env("ANTHROPIC_VERTEX_PROJECT_ID", "").strip()
    region = context.get_env("CLOUD_ML_REGION", "").strip()

    if not service_account_path:
        raise RuntimeError("GOOGLE_APPLICATION_CREDENTIALS must be set when CLAUDE_CODE_USE_VERTEX=1")
    if not project_id:
        raise RuntimeError("ANTHROPIC_VERTEX_PROJECT_ID must be set when CLAUDE_CODE_USE_VERTEX=1")
    if not region:
        raise RuntimeError("CLOUD_ML_REGION must be set when CLAUDE_CODE_USE_VERTEX=1")

    if not Path(service_account_path).exists():
        raise RuntimeError(f"Service account key file not found at {service_account_path}")

    logger.info(f"Vertex AI configured: project={project_id}, region={region}")
    return {"credentials_path": service_account_path, "project_id": project_id, "region": region}


async def setup_sdk_authentication(context: RunnerContext) -> tuple[str, bool, str]:
    """Set up SDK auth env vars for the Claude Agent SDK.

    Returns:
        (api_key, use_vertex, configured_model)
    """
    api_key = context.get_env("ANTHROPIC_API_KEY", "")
    use_vertex = context.get_env("CLAUDE_CODE_USE_VERTEX", "").strip() == "1"

    if not api_key and not use_vertex:
        raise RuntimeError("Either ANTHROPIC_API_KEY or CLAUDE_CODE_USE_VERTEX=1 must be set")

    model = context.get_env("LLM_MODEL")

    # Default model differs: Vertex AI uses @date suffixes, Anthropic API does not
    DEFAULT_MODEL = "claude-sonnet-4-5"
    DEFAULT_VERTEX_MODEL = "claude-sonnet-4-5@20250929"

    if api_key and not use_vertex:
        os.environ["ANTHROPIC_API_KEY"] = api_key
        configured_model = model or DEFAULT_MODEL
        logger.info(f"Using Anthropic API key authentication (model={configured_model})")

    elif use_vertex:
        vertex_credentials = await setup_vertex_credentials(context)
        os.environ["ANTHROPIC_API_KEY"] = "vertex-auth-mode"
        os.environ["CLAUDE_CODE_USE_VERTEX"] = "1"
        os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = vertex_credentials.get("credentials_path", "")
        os.environ["ANTHROPIC_VERTEX_PROJECT_ID"] = vertex_credentials.get("project_id", "")
        os.environ["CLOUD_ML_REGION"] = vertex_credentials.get("region", "")
        configured_model = map_to_vertex_model(model) if model else DEFAULT_VERTEX_MODEL
        logger.info(f"Using Vertex AI authentication (model={configured_model})")

    else:
        configured_model = model or DEFAULT_MODEL

    return api_key, use_vertex, configured_model
