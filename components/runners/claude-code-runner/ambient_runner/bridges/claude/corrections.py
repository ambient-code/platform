"""
Corrections feedback MCP tool for capturing human corrections.

When a user corrects the agent's work during a session, this tool
logs the correction to Langfuse as a structured score with category,
severity, scope, and contextual metadata. A downstream feedback loop
(GitHub Action) periodically queries these scores and creates
improvement sessions to update workflow instructions and repo context.
"""

import logging
import os
from typing import Any

logger = logging.getLogger(__name__)


# ------------------------------------------------------------------
# Constants
# ------------------------------------------------------------------

CORRECTION_CATEGORIES = [
    "wrong_approach",
    "missing_context",
    "incorrect_output",
    "style_violation",
    "security_issue",
    "incomplete_work",
    "overcomplicated",
    "wrong_scope",
    "misunderstood_requirements",
]

CORRECTION_SCOPES = [
    "workflow_instructions",
    "repo_context",
    "code_patterns",
    "documentation",
]

CORRECTION_TOOL_DESCRIPTION = (
    "Log a correction when the user points out an error, corrects your work, "
    "or provides feedback about mistakes. Call this BEFORE fixing the issue.\n\n"
    "Trigger when the user: says you did something wrong or made a mistake, "
    "asks you to redo or fix previous work, points out bugs/errors/security issues, "
    "says you modified the wrong files or missed relevant ones, indicates you "
    "over-engineered or under-delivered, clarifies requirements you misunderstood, "
    "or provides context you should have known.\n\n"
    "Pick the category that best describes what went wrong, rate severity "
    "(1=minor, 2=significant, 3=critical), and choose the scope that needs "
    "updating to prevent recurrence. Be honest and specific in the description."
)

CORRECTION_INPUT_SCHEMA: dict = {
    "type": "object",
    "properties": {
        "category": {
            "type": "string",
            "enum": CORRECTION_CATEGORIES,
            "description": (
                "The type of correction: "
                "wrong_approach (fundamentally wrong strategy), "
                "missing_context (lacked necessary knowledge), "
                "incorrect_output (bugs or errors in output), "
                "style_violation (didn't follow code style/patterns), "
                "security_issue (introduced or missed vulnerability), "
                "incomplete_work (didn't finish the task fully), "
                "overcomplicated (over-engineered the solution), "
                "wrong_scope (modified wrong files or missed relevant ones), "
                "misunderstood_requirements (misinterpreted what was needed)."
            ),
        },
        "severity": {
            "type": "integer",
            "enum": [1, 2, 3],
            "description": (
                "Severity of the correction: "
                "1 = low (minor/cosmetic), "
                "2 = medium (significant/functional), "
                "3 = high (critical/blocking)."
            ),
        },
        "scope": {
            "type": "string",
            "enum": CORRECTION_SCOPES,
            "description": (
                "What should be updated to prevent this in the future: "
                "workflow_instructions (update .ambient/ workflow files), "
                "repo_context (update CLAUDE.md or .claude/ context files), "
                "code_patterns (update .claude/patterns/ files), "
                "documentation (update relevant docs)."
            ),
        },
        "description": {
            "type": "string",
            "description": (
                "Detailed description of what went wrong and what the "
                "correct approach should have been."
            ),
        },
        "correction_details": {
            "type": "string",
            "description": (
                "Specific details about the fix applied or the correct "
                "solution. Optional but recommended for actionable feedback."
            ),
        },
    },
    "required": ["category", "severity", "scope", "description"],
}


# ------------------------------------------------------------------
# Tool factory
# ------------------------------------------------------------------


def create_correction_mcp_tool(
    obs: Any,
    session_id: str,
    sdk_tool_decorator,
):
    """Create the log_correction MCP tool.

    Args:
        obs: ObservabilityManager instance for trace ID and Langfuse client.
        session_id: Current session ID.
        sdk_tool_decorator: The ``tool`` decorator from ``claude_agent_sdk``.

    Returns:
        Decorated async tool function.
    """
    _obs = obs
    _session_id = session_id

    @sdk_tool_decorator(
        "log_correction",
        CORRECTION_TOOL_DESCRIPTION,
        CORRECTION_INPUT_SCHEMA,
    )
    async def log_correction_tool(args: dict) -> dict:
        """Log a correction to Langfuse for the feedback loop."""
        category = args.get("category", "")
        severity = args.get("severity", 2)
        scope = args.get("scope", "")
        description = args.get("description", "")
        correction_details = args.get("correction_details")

        success, error = _log_correction_to_langfuse(
            category=category,
            severity=severity,
            scope=scope,
            description=description,
            correction_details=correction_details,
            obs=_obs,
            session_id=_session_id,
        )

        if success:
            return {
                "content": [
                    {
                        "type": "text",
                        "text": (
                            f"Correction logged: category={category}, "
                            f"severity={severity}, scope={scope}. "
                            "This will be reviewed in the next feedback loop cycle."
                        ),
                    }
                ]
            }
        else:
            return {
                "content": [
                    {"type": "text", "text": f"Failed to log correction: {error}"}
                ],
                "isError": True,
            }

    return log_correction_tool


# ------------------------------------------------------------------
# Auto-captured context
# ------------------------------------------------------------------


def _get_session_context() -> dict:
    """Auto-capture session context from environment variables.

    Returns:
        Dict with repo_url, workflow, session_name, and project.
    """
    return {
        "repo_url": os.getenv("ACTIVE_WORKFLOW_GIT_URL", "").strip(),
        "workflow": os.getenv("ACTIVE_WORKFLOW_PATH", "").strip(),
        "session_name": os.getenv("AGENTIC_SESSION_NAME", "").strip(),
        "project": os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip(),
    }


# ------------------------------------------------------------------
# Langfuse logging
# ------------------------------------------------------------------


def _log_correction_to_langfuse(
    category: str,
    severity: int,
    scope: str,
    description: str,
    correction_details: str | None,
    obs: Any,
    session_id: str,
) -> tuple[bool, str | None]:
    """Log a correction score to Langfuse.

    Mirrors the rubric tool's ``_log_to_langfuse`` pattern from tools.py.
    """
    try:
        langfuse_client = getattr(obs, "langfuse_client", None) if obs else None
        using_obs_client = langfuse_client is not None

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

        # Only use trace_id from obs's own client — a fallback ad-hoc client
        # has no knowledge of traces created by the original obs instance.
        # MCP tools run in a different async context so get_current_trace_id()
        # may return None even mid-turn; fall back to last_trace_id which
        # persists across turn boundaries.
        if using_obs_client:
            try:
                trace_id = obs.get_current_trace_id() if obs else None
                if trace_id is None:
                    trace_id = getattr(obs, "last_trace_id", None)
            except Exception:
                trace_id = getattr(obs, "last_trace_id", None)
        else:
            trace_id = None

        context = _get_session_context()

        metadata = {
            "category": category,
            "scope": scope,
            "session_id": session_id,
            "session_name": context["session_name"],
            "project": context["project"],
            "repo_url": context["repo_url"],
            "workflow": context["workflow"],
        }
        if correction_details:
            metadata["correction_details"] = correction_details[:500]

        kwargs: dict = {
            "name": "session-correction",
            "value": severity,
            "data_type": "NUMERIC",
            "comment": description[:500] if description else None,
            "metadata": metadata,
        }
        if trace_id:
            kwargs["trace_id"] = trace_id

        langfuse_client.create_score(**kwargs)
        langfuse_client.flush()

        logger.info(
            f"Correction logged to Langfuse: "
            f"category={category}, severity={severity}, "
            f"scope={scope}, trace_id={trace_id}"
        )
        return True, None

    except ImportError:
        return False, "Langfuse package not installed."
    except Exception as e:
        msg = str(e)
        logger.error(f"Failed to log correction to Langfuse: {msg}")
        return False, msg
