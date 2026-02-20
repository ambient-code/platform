"""
Corrections feedback MCP tool for capturing human corrections.

When a user corrects the agent's work during a session, this tool
logs the correction to Langfuse as a categorical score capturing what
the agent did and what the user corrected it to. A downstream feedback
loop (GitHub Action) periodically queries these scores and creates
improvement sessions to update workflow instructions and repo context.
"""

import json
import logging
import os
from typing import Any

logger = logging.getLogger(__name__)


# ------------------------------------------------------------------
# Constants
# ------------------------------------------------------------------

CORRECTION_TYPES = [
    "incomplete",    # missed something that should have been done
    "incorrect",     # did the wrong thing
    "out_of_scope",  # worked on wrong files / area
    "style",         # right result, wrong approach or pattern
]

CORRECTION_TOOL_DESCRIPTION = (
    "Log a correction whenever the user redirects, corrects, or changes what "
    "you did or assumed. Call this BEFORE fixing the issue.\n\n"
    "Use broad judgment — if the user is steering you away from something you "
    "already did or decided, that is a correction. This includes: pointing out "
    "errors or bugs, asking you to redo work, clarifying what they actually "
    "wanted, saying you missed something, telling you the approach was wrong, "
    "or providing any context that changes what you should have done. When in "
    "doubt, log it.\n\n"
    "Fields:\n"
    "- agent_action: what you did or assumed (be honest and specific)\n"
    "- user_correction: exactly what the user said should have happened instead\n"
    "- correction_type: pick the best fit — "
    "incomplete (missed something), "
    "incorrect (did the wrong thing), "
    "out_of_scope (wrong files or area), "
    "style (right result but wrong approach or pattern)"
)

CORRECTION_INPUT_SCHEMA: dict = {
    "type": "object",
    "properties": {
        "correction_type": {
            "type": "string",
            "enum": CORRECTION_TYPES,
            "description": (
                "The type of correction: "
                "incomplete (missed something that should have been done), "
                "incorrect (did the wrong thing), "
                "out_of_scope (worked on wrong files or area), "
                "style (right result but wrong approach or pattern)."
            ),
        },
        "agent_action": {
            "type": "string",
            "description": (
                "What the agent did or assumed. Be honest and specific about "
                "the action taken or assumption made before the correction."
            ),
        },
        "user_correction": {
            "type": "string",
            "description": (
                "What the user said should have happened instead. Capture "
                "their correction as accurately as possible."
            ),
        },
    },
    "required": ["correction_type", "agent_action", "user_correction"],
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
        correction_type = args.get("correction_type", "")
        agent_action = args.get("agent_action", "")
        user_correction = args.get("user_correction", "")

        success, error = _log_correction_to_langfuse(
            correction_type=correction_type,
            agent_action=agent_action,
            user_correction=user_correction,
            obs=_obs,
            session_id=_session_id,
        )

        if success:
            return {
                "content": [
                    {
                        "type": "text",
                        "text": (
                            f"Correction logged: type={correction_type}. "
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


def _parse_repos_json() -> list:
    """Parse REPOS_JSON env var into a list of repo dicts.

    Returns:
        List of dicts with 'url' and 'branch' keys, or empty list.
    """
    raw = os.getenv("REPOS_JSON", "").strip()
    if not raw:
        return []
    try:
        repos = json.loads(raw)
        if not isinstance(repos, list):
            return []
        result = []
        for r in repos:
            if isinstance(r, dict) and r.get("url"):
                result.append({
                    "url": r.get("url", ""),
                    "branch": r.get("branch", ""),
                })
        return result
    except Exception:
        return []


def _get_session_context() -> dict:
    """Auto-capture session context from environment variables.

    Returns:
        Dict with workflow (repo_url, branch, path), repos list,
        session_name, and project.
    """
    return {
        "workflow": {
            "repo_url": os.getenv("ACTIVE_WORKFLOW_GIT_URL", "").strip(),
            "branch": os.getenv("ACTIVE_WORKFLOW_BRANCH", "").strip(),
            "path": os.getenv("ACTIVE_WORKFLOW_PATH", "").strip(),
        },
        "repos": _parse_repos_json(),
        "session_name": os.getenv("AGENTIC_SESSION_NAME", "").strip(),
        "project": os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip(),
    }


# ------------------------------------------------------------------
# Langfuse logging
# ------------------------------------------------------------------


def _log_correction_to_langfuse(
    correction_type: str,
    agent_action: str,
    user_correction: str,
    obs: Any,
    session_id: str,
) -> tuple[bool, str | None]:
    """Log a correction score to Langfuse."""
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

        comment = (
            f"Agent did: {agent_action[:500]}\n"
            f"User corrected to: {user_correction[:500]}"
        )

        repos = context["repos"]
        metadata = {
            "correction_type": correction_type,
            "agent_action": agent_action[:500],
            "user_correction": user_correction[:500],
            "session_id": session_id,
            "session_name": context["session_name"],
            "project": context["project"],
            "workflow_repo_url": context["workflow"]["repo_url"],
            "workflow_branch": context["workflow"]["branch"],
            "workflow_path": context["workflow"]["path"],
            "repos": json.dumps(repos) if repos else "",
        }

        kwargs: dict = {
            "name": "session-correction",
            "value": correction_type,
            "data_type": "CATEGORICAL",
            "comment": comment,
            "metadata": metadata,
        }
        if trace_id:
            kwargs["trace_id"] = trace_id

        langfuse_client.create_score(**kwargs)
        langfuse_client.flush()

        logger.info(
            f"Correction logged to Langfuse: "
            f"type={correction_type}, trace_id={trace_id}"
        )
        return True, None

    except ImportError:
        return False, "Langfuse package not installed."
    except Exception as e:
        msg = str(e)
        logger.error(f"Failed to log correction to Langfuse: {msg}")
        return False, msg
