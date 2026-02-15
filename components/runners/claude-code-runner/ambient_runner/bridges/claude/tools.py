"""
Claude-specific MCP tool definitions.

Tools are created dynamically per-run and registered as in-process
MCP servers alongside the Claude Agent SDK.

- ``restart_session`` — allows Claude to request a session restart
- ``evaluate_rubric`` — logs a rubric evaluation score to Langfuse
"""

import json as _json
import logging
import os
from pathlib import Path
from typing import Any

from ambient_runner.platform.prompts import RESTART_TOOL_DESCRIPTION

logger = logging.getLogger(__name__)


# ------------------------------------------------------------------
# Session restart tool
# ------------------------------------------------------------------


def create_restart_session_tool(adapter_ref, sdk_tool_decorator):
    """Create the restart_session MCP tool.

    Args:
        adapter_ref: Reference to the ClaudeCodeAdapter instance
            (used to set _restart_requested flag).
        sdk_tool_decorator: The ``tool`` decorator from ``claude_agent_sdk``.

    Returns:
        Decorated async tool function.
    """

    @sdk_tool_decorator(
        "restart_session",
        RESTART_TOOL_DESCRIPTION,
        {},
    )
    async def restart_session_tool(args: dict) -> dict:
        """Tool that allows Claude to request a session restart."""
        adapter_ref._restart_requested = True
        logger.info("Session restart requested by Claude via MCP tool")
        return {
            "content": [
                {
                    "type": "text",
                    "text": (
                        "Session restart has been requested. The current run "
                        "will complete and a fresh session will be established. "
                        "Your conversation context will be preserved on disk."
                    ),
                }
            ]
        }

    return restart_session_tool


# ------------------------------------------------------------------
# Rubric evaluation tool
# ------------------------------------------------------------------


def load_rubric_content(cwd_path: str) -> tuple:
    """Load rubric content from the workflow's .ambient/ folder.

    Looks for ``.ambient/rubric.md`` — a single markdown file containing
    the evaluation criteria.

    Returns:
        Tuple of ``(rubric_content, rubric_config)`` where rubric_content
        is the markdown string and rubric_config is the ``rubric`` key
        from ambient.json.  Returns ``(None, {})`` if no rubric found.
    """
    ambient_dir = Path(cwd_path) / ".ambient"
    rubric_content = None

    single_rubric = ambient_dir / "rubric.md"
    if single_rubric.exists() and single_rubric.is_file():
        try:
            rubric_content = single_rubric.read_text(encoding="utf-8")
            logger.info(f"Loaded rubric from {single_rubric}")
        except Exception as e:
            logger.error(f"Failed to read rubric.md: {e}")

    rubric_config: dict = {}
    try:
        config_path = ambient_dir / "ambient.json"
        if config_path.exists():
            with open(config_path, "r") as f:
                config = _json.load(f)
                rubric_config = config.get("rubric", {})
    except Exception as e:
        logger.error(f"Failed to load rubric config from ambient.json: {e}")

    return rubric_content, rubric_config


def create_rubric_mcp_tool(
    rubric_content: str,
    rubric_config: dict,
    obs: Any,
    session_id: str,
    sdk_tool_decorator,
):
    """Create a dynamic MCP tool for rubric-based evaluation.

    The tool accepts a score, comment, and optional metadata, then makes
    a single ``langfuse.create_score()`` call. The ``rubric.schema`` from
    ambient.json is passed through as the ``metadata`` field's JSON Schema
    in the tool's input_schema.

    Args:
        rubric_content: Markdown rubric instructions (for reference only).
        rubric_config: Config dict with ``activationPrompt`` and ``schema``.
        obs: ObservabilityManager instance for trace ID.
        session_id: Current session ID.
        sdk_tool_decorator: The ``tool`` decorator from ``claude_agent_sdk``.

    Returns:
        Decorated async tool function.
    """
    user_schema = rubric_config.get("schema", {})

    properties: dict = {
        "score": {"type": "number", "description": "Overall evaluation score."},
        "comment": {"type": "string", "description": "Evaluation reasoning and commentary."},
    }
    if user_schema:
        properties["metadata"] = user_schema

    required = ["score", "comment"]
    if user_schema:
        required.append("metadata")

    input_schema: dict = {
        "type": "object",
        "properties": properties,
        "required": required,
    }

    tool_description = (
        "Log a rubric evaluation score to Langfuse. "
        "Read .ambient/rubric.md FIRST, evaluate the output "
        "against the criteria, then call this tool with your "
        "score, comment, and metadata."
    )

    _obs = obs
    _session_id = session_id

    @sdk_tool_decorator(
        "evaluate_rubric",
        tool_description,
        input_schema,
    )
    async def evaluate_rubric_tool(args: dict) -> dict:
        """Log a single rubric evaluation score to Langfuse."""
        score = args.get("score")
        comment = args.get("comment", "")
        metadata = args.get("metadata")

        success, error = _log_to_langfuse(
            score=score,
            comment=comment,
            metadata=metadata,
            obs=_obs,
            session_id=_session_id,
        )

        if success:
            return {
                "content": [
                    {"type": "text", "text": f"Score {score} logged to Langfuse."}
                ]
            }
        else:
            return {
                "content": [
                    {"type": "text", "text": f"Failed to log score: {error}"}
                ],
                "isError": True,
            }

    return evaluate_rubric_tool


def _log_to_langfuse(
    score: float | None,
    comment: str,
    metadata: Any,
    obs: Any,
    session_id: str,
) -> tuple[bool, str | None]:
    """Make a single langfuse.create_score() call."""
    try:
        langfuse_client = getattr(obs, "langfuse_client", None) if obs else None

        if not langfuse_client:
            langfuse_enabled = os.getenv(
                "LANGFUSE_ENABLED", ""
            ).strip().lower() in ("1", "true", "yes")
            if not langfuse_enabled:
                return False, "Langfuse not enabled."

            from langfuse import Langfuse

            public_key = os.getenv("LANGFUSE_PUBLIC_KEY", "").strip()
            secret_key = os.getenv("LANGFUSE_SECRET_KEY", "").strip()
            host = os.getenv("LANGFUSE_HOST", "").strip()

            if not (public_key and secret_key and host):
                return False, "Langfuse credentials missing."

            langfuse_client = Langfuse(
                public_key=public_key,
                secret_key=secret_key,
                host=host,
            )

        trace_id = obs.get_current_trace_id() if obs else None

        if score is None:
            return False, "Score value is required (got None)."

        kwargs: dict = {
            "name": "rubric-evaluation",
            "value": score,
            "data_type": "NUMERIC",
            "comment": comment[:500] if comment else None,
            "metadata": metadata,
        }
        if trace_id:
            kwargs["trace_id"] = trace_id

        langfuse_client.create_score(**kwargs)
        langfuse_client.flush()

        logger.info(
            f"Rubric score logged to Langfuse: "
            f"value={score}, trace_id={trace_id}"
        )
        return True, None

    except ImportError:
        return False, "Langfuse package not installed."
    except Exception as e:
        msg = str(e)
        logger.error(f"Failed to log rubric score to Langfuse: {msg}")
        return False, msg
