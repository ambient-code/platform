"""Gemini CLI authentication — API key and Vertex AI setup."""

import logging

from ambient_runner.platform.auth import validate_vertex_credentials_file
from ambient_runner.platform.context import RunnerContext
from ambient_runner.platform.utils import is_vertex_enabled

logger = logging.getLogger(__name__)


async def setup_gemini_cli_auth(context: RunnerContext) -> tuple[str, str, bool]:
    """Set up Gemini CLI authentication from environment.

    Two modes:
    - **API key** (default): Uses GEMINI_API_KEY or GOOGLE_API_KEY
    - **Vertex AI**: When USE_VERTEX=1 (or legacy GEMINI_USE_VERTEX=1), uses
      Google Cloud credentials (GOOGLE_CLOUD_PROJECT, GOOGLE_CLOUD_LOCATION,
      GOOGLE_APPLICATION_CREDENTIALS).

    Returns:
        (model, api_key, use_vertex)
    """
    from ag_ui_gemini_cli.config import DEFAULT_MODEL

    model = context.get_env("LLM_MODEL", DEFAULT_MODEL).strip()
    use_vertex = is_vertex_enabled(legacy_var="GEMINI_USE_VERTEX", context=context)

    if use_vertex:
        validate_vertex_credentials_file(context)

        # Resolve project/location — may be set directly or via platform aliases
        # (ANTHROPIC_VERTEX_PROJECT_ID / CLOUD_ML_REGION). The subprocess mapping
        # in session.py handles the latter; here we just log what we found.
        project = (
            context.get_env("GOOGLE_CLOUD_PROJECT", "").strip()
            or context.get_env("ANTHROPIC_VERTEX_PROJECT_ID", "").strip()
        )
        location = (
            context.get_env("GOOGLE_CLOUD_LOCATION", "").strip()
            or context.get_env("CLOUD_ML_REGION", "").strip()
        )

        logger.info(
            "Gemini CLI: Vertex AI mode (project=%s, location=%s, model=%s)",
            project or "unset",
            location or "default",
            model,
        )
        return model, "", True

    api_key = (
        context.get_env("GEMINI_API_KEY", "").strip()
        or context.get_env("GOOGLE_API_KEY", "").strip()
    )

    if api_key:
        logger.info("Gemini CLI: using API key (model=%s)", model)
    else:
        logger.info("Gemini CLI: no API key, relying on gcloud auth (model=%s)", model)

    return model, api_key, False
