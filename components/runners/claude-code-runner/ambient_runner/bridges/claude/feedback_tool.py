"""
Global /feedback SDK tool for capturing user satisfaction during sessions.

When a user expresses satisfaction, dissatisfaction, or provides qualitative
feedback about the session or agent output, this tool records it. When
Langfuse is configured, feedback is logged as a scored event; otherwise it
falls back to stdout (pod logs) so feedback is never lost.

Available in every session regardless of workflow configuration.
"""

import logging
import os
from typing import Any

logger = logging.getLogger(__name__)


# ------------------------------------------------------------------
# Constants
# ------------------------------------------------------------------

FEEDBACK_RATINGS = ["positive", "negative"]

FEEDBACK_TOOL_DESCRIPTION = (
    "Submit user feedback about the session or agent output. Call this when "
    "the user explicitly rates the session, expresses satisfaction or "
    "dissatisfaction, or provides qualitative feedback about quality.\n\n"
    "## When to call\n\n"
    "- User says the output is good, great, perfect, or similar praise\n"
    "- User says the output is bad, wrong, unhelpful, or similar criticism\n"
    "- User explicitly asks to submit feedback or rate the session\n"
    "- User gives a thumbs up / thumbs down\n\n"
    "## Fields\n\n"
    "- `rating`: 'positive' for praise/satisfaction, 'negative' for "
    "criticism/dissatisfaction\n"
    "- `comment`: the user's exact words or a brief summary of their feedback\n"
)

FEEDBACK_INPUT_SCHEMA: dict = {
    "type": "object",
    "properties": {
        "rating": {
            "type": "string",
            "enum": FEEDBACK_RATINGS,
            "description": (
                "User sentiment: 'positive' for satisfaction/praise, "
                "'negative' for dissatisfaction/criticism."
            ),
        },
        "comment": {
            "type": "string",
            "description": (
                "The user's feedback comment. Capture their exact words "
                "or a concise summary of what they said."
            ),
        },
    },
    "required": ["rating", "comment"],
}


# ------------------------------------------------------------------
# Tool factory
# ------------------------------------------------------------------


def create_feedback_mcp_tool(
    obs: Any,
    session_id: str,
    sdk_tool_decorator,
):
    """Create the submit_feedback MCP tool.

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
        "submit_feedback",
        FEEDBACK_TOOL_DESCRIPTION,
        FEEDBACK_INPUT_SCHEMA,
    )
    async def submit_feedback_tool(args: dict) -> dict:
        """Log user feedback to Langfuse."""
        rating = args.get("rating", "")
        comment = args.get("comment", "")

        success, error = _log_feedback_to_langfuse(
            rating=rating,
            comment=comment,
            obs=_obs,
            session_id=_session_id,
        )

        if success:
            return {
                "content": [
                    {
                        "type": "text",
                        "text": (
                            f"Feedback recorded (rating={rating}). "
                            "Thank you for helping improve the platform."
                        ),
                    }
                ]
            }
        else:
            return {
                "content": [
                    {
                        "type": "text",
                        "text": f"Feedback noted but could not be recorded: {error}",
                    }
                ],
                "isError": True,
            }

    return submit_feedback_tool


# ------------------------------------------------------------------
# Langfuse logging (with stdout fallback)
# ------------------------------------------------------------------


def _log_feedback_fallback(
    reason: str, rating: str, comment: str, session_id: str
) -> tuple[bool, None]:
    """Log feedback to stdout when Langfuse is unavailable."""
    logger.info(
        f"Feedback ({reason}): rating={rating}, "
        f"comment={comment[:500] if comment else ''}, "
        f"session_id={session_id}"
    )
    return True, None


def _log_feedback_to_langfuse(
    rating: str,
    comment: str,
    obs: Any,
    session_id: str,
) -> tuple[bool, str | None]:
    """Log a user feedback score to Langfuse."""
    try:
        langfuse_client = getattr(obs, "langfuse_client", None) if obs else None
        using_obs_client = langfuse_client is not None

        if not langfuse_client:
            langfuse_enabled = os.getenv("LANGFUSE_ENABLED", "").strip().lower() in (
                "1",
                "true",
                "yes",
            )
            if not langfuse_enabled:
                return _log_feedback_fallback(
                    "no Langfuse", rating, comment, session_id
                )

            from langfuse import Langfuse

            public_key = os.getenv("LANGFUSE_PUBLIC_KEY", "").strip()
            secret_key = os.getenv("LANGFUSE_SECRET_KEY", "").strip()
            host = os.getenv("LANGFUSE_HOST", "").strip()

            if not (public_key and secret_key and host):
                return _log_feedback_fallback(
                    "Langfuse creds missing", rating, comment, session_id
                )

            langfuse_client = Langfuse(
                public_key=public_key,
                secret_key=secret_key,
                host=host,
            )

        # Prefer obs-owned trace ID; fall back to last_trace_id across turns.
        if using_obs_client:
            try:
                trace_id = obs.get_current_trace_id() if obs else None
                if trace_id is None:
                    trace_id = getattr(obs, "last_trace_id", None)
            except Exception:
                trace_id = getattr(obs, "last_trace_id", None)
        else:
            trace_id = None

        value = rating == "positive"

        session_name = os.getenv("AGENTIC_SESSION_NAME", "").strip()
        project = os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()

        metadata: dict = {
            "rating": rating,
            "session_id": session_id,
            "session_name": session_name,
            "project": project,
        }

        kwargs: dict = {
            "name": "session-feedback",
            "value": value,
            "data_type": "BOOLEAN",
            "comment": comment[:500] if comment else None,
            "metadata": metadata,
        }
        if trace_id:
            kwargs["trace_id"] = trace_id

        langfuse_client.create_score(**kwargs)
        langfuse_client.flush()

        logger.info(
            f"Feedback logged to Langfuse: rating={rating}, trace_id={trace_id}"
        )
        return True, None

    except ImportError:
        return _log_feedback_fallback(
            "langfuse not installed", rating, comment, session_id
        )
    except Exception as e:
        msg = str(e)
        logger.error(f"Failed to log feedback to Langfuse: {msg}")
        return False, msg
