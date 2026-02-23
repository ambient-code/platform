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

CORRECTION_SOURCES = ["human", "rubric"]

CORRECTION_TOOL_DESCRIPTION_BASE = (
    "Log a correction whenever the user redirects, corrects, or changes what "
    "you did or assumed. Call this BEFORE fixing the issue.\n\n"
    "Use broad judgment — if the user is steering you away from something you "
    "already did or decided, that is a correction. This includes: pointing out "
    "errors or bugs, asking you to redo work, clarifying what they actually "
    "wanted, saying you missed something, telling you the approach was wrong, "
    "or providing any context that changes what you should have done. When in "
    "doubt, log it.\n\n"
    "## Targeting\n\n"
    "Pick which target this correction applies to using the `target` field. "
    "This determines WHERE the downstream improvement session will make fixes "
    "— either in the workflow instructions or in the repo's context files.\n\n"
    "If BOTH a workflow AND a repo need fixing, call this tool TWICE — "
    "once for each target.\n\n"
    "Fields:\n"
    "- correction_type: pick the best fit — "
    "incomplete (missed something), "
    "incorrect (did the wrong thing), "
    "out_of_scope (wrong files or area), "
    "style (right result but wrong approach or pattern)\n"
    "- agent_action: what you did or assumed (be honest and specific)\n"
    "- user_correction: exactly what the user said should have happened instead\n"
    "- target: which target this correction applies to"
)

RUBRIC_CORRECTION_ADDENDUM = (
    "\n\n## Post-Rubric Corrections\n\n"
    "This workflow has a rubric. After calling `evaluate_rubric`, also call "
    "this tool for each dimension that scored below the midpoint of its scale. "
    "For each weak dimension:\n"
    "- correction_type: 'style'\n"
    "- agent_action: what the output did for that dimension\n"
    "- user_correction: what the rubric says it should have done\n"
    "- source: 'rubric'\n\n"
    "This feeds rubric feedback into the improvement loop."
)

BASE_CORRECTION_PROPERTIES: dict = {
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
    "source": {
        "type": "string",
        "enum": CORRECTION_SOURCES,
        "description": (
            "Where this correction came from: "
            "'human' (default) for user-provided corrections, "
            "'rubric' when logging weak dimensions after a rubric evaluation."
        ),
    },
}

BASE_REQUIRED_FIELDS = ["correction_type", "agent_action", "user_correction"]


# ------------------------------------------------------------------
# Target map
# ------------------------------------------------------------------


def _repo_name(url: str) -> str:
    """Extract a short name from a repo URL for use as a target label."""
    name = url.rstrip("/").split("/")[-1]
    if name.endswith(".git"):
        name = name[:-4]
    return name


def build_target_map(context: dict) -> dict[str, dict]:
    """Build a mapping from friendly label to full target details.

    Each entry contains ``target_type``, ``target_repo_url``,
    ``target_branch``, and ``target_path``.
    """
    targets: dict[str, dict] = {}

    wf = context.get("workflow") or {}
    wf_url = wf.get("repo_url", "")
    wf_branch = wf.get("branch", "")
    wf_path = wf.get("path", "")

    if wf_url:
        label = wf_path.rstrip("/").split("/")[-1] if wf_path else _repo_name(wf_url)
        targets[label] = {
            "target_type": "workflow",
            "target_repo_url": wf_url,
            "target_branch": wf_branch,
            "target_path": wf_path,
        }

    for repo in context.get("repos") or []:
        url = repo.get("url", "")
        if not url:
            continue
        label = _repo_name(url)
        if label in targets:
            label = f"{label}-repo"
        targets[label] = {
            "target_type": "repo",
            "target_repo_url": url,
            "target_branch": repo.get("branch", ""),
            "target_path": "",
        }

    return targets


def build_correction_schema(target_labels: list[str]) -> dict:
    """Build the full input schema with a dynamic ``target`` enum.

    When only one target is available, ``target`` is optional (auto-filled).
    When no targets are available, ``target`` is omitted entirely.
    """
    import copy

    properties = copy.deepcopy(BASE_CORRECTION_PROPERTIES)
    required = list(BASE_REQUIRED_FIELDS)

    if target_labels:
        properties["target"] = {
            "type": "string",
            "enum": target_labels,
            "description": (
                "Which target this correction applies to. "
                "Pick from the available targets listed above."
            ),
        }
        if len(target_labels) > 1:
            required.append("target")

    return {
        "type": "object",
        "properties": properties,
        "required": required,
    }


def _build_tool_description(
    target_map: dict[str, dict],
    has_rubric: bool = False,
) -> str:
    """Build the tool description including available targets."""
    desc = CORRECTION_TOOL_DESCRIPTION_BASE

    if target_map:
        desc += "\n\n## Available Targets\n\n"
        for label, target in target_map.items():
            t_type = target["target_type"]
            url = target["target_repo_url"]
            short_url = _repo_name(url) if url else "unknown"
            if t_type == "workflow":
                path = target.get("target_path", "")
                desc += f"- `{label}` (workflow): {path or short_url}\n"
            else:
                desc += f"- `{label}` (repo): {short_url}\n"

    if has_rubric:
        desc += RUBRIC_CORRECTION_ADDENDUM

    return desc


# ------------------------------------------------------------------
# Tool factory
# ------------------------------------------------------------------


def create_correction_mcp_tool(
    obs: Any,
    session_id: str,
    sdk_tool_decorator,
    has_rubric: bool = False,
):
    """Create the log_correction MCP tool.

    Args:
        obs: ObservabilityManager instance for trace ID and Langfuse client.
        session_id: Current session ID.
        sdk_tool_decorator: The ``tool`` decorator from ``claude_agent_sdk``.
        has_rubric: Whether a rubric evaluation tool is also available.
            When True the tool description is extended to instruct the
            agent to log corrections after rubric evaluations.

    Returns:
        Decorated async tool function.
    """
    _obs = obs
    _session_id = session_id

    context = _get_session_context()
    _target_map = build_target_map(context)

    description = _build_tool_description(_target_map, has_rubric=has_rubric)
    schema = build_correction_schema(list(_target_map.keys()))

    @sdk_tool_decorator(
        "log_correction",
        description,
        schema,
    )
    async def log_correction_tool(args: dict) -> dict:
        """Log a correction to Langfuse for the feedback loop."""
        correction_type = args.get("correction_type", "")
        agent_action = args.get("agent_action", "")
        user_correction = args.get("user_correction", "")
        target_label = args.get("target", "")
        source = args.get("source", "human")

        success, error = _log_correction_to_langfuse(
            correction_type=correction_type,
            agent_action=agent_action,
            user_correction=user_correction,
            target_label=target_label,
            target_map=_target_map,
            obs=_obs,
            session_id=_session_id,
            source=source,
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

    Handles both operator-injected format (top-level url/branch) and
    runtime-added format (url/branch nested under ``input``).

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
            if not isinstance(r, dict):
                continue
            url = r.get("url", "")
            branch = r.get("branch", "")
            if not url:
                inp = r.get("input") or {}
                if isinstance(inp, dict):
                    url = inp.get("url", "")
                    branch = branch or inp.get("branch", "")
            if url:
                result.append({"url": url, "branch": branch})
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
# Target resolution
# ------------------------------------------------------------------


def _resolve_target(target_label: str, target_map: dict[str, dict]) -> dict:
    """Resolve a target label to full target details via the target map.

    Falls back to the first (or only) entry when the label is empty.
    Returns empty target dict when the map is empty.
    """
    if target_label and target_label in target_map:
        return target_map[target_label]

    if len(target_map) == 1:
        return next(iter(target_map.values()))

    if target_map:
        return next(iter(target_map.values()))

    return {
        "target_type": "",
        "target_repo_url": "",
        "target_branch": "",
        "target_path": "",
    }


# ------------------------------------------------------------------
# Langfuse logging
# ------------------------------------------------------------------


def _log_correction_to_langfuse(
    correction_type: str,
    agent_action: str,
    user_correction: str,
    target_label: str,
    target_map: dict[str, dict],
    obs: Any,
    session_id: str,
    source: str = "human",
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
        target = _resolve_target(target_label, target_map)

        comment = (
            f"Agent did: {agent_action[:500]}\n"
            f"User corrected to: {user_correction[:500]}"
        )

        metadata = {
            "correction_type": correction_type,
            "source": source,
            "agent_action": agent_action[:500],
            "user_correction": user_correction[:500],
            "target_type": target["target_type"],
            "target_repo_url": target["target_repo_url"],
            "target_branch": target["target_branch"],
            "target_path": target["target_path"],
            "session_id": session_id,
            "session_name": context["session_name"],
            "project": context["project"],
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
            f"type={correction_type}, target={target['target_type']}:"
            f"{target['target_repo_url']}, trace_id={trace_id}"
        )
        return True, None

    except ImportError:
        return False, "Langfuse package not installed."
    except Exception as e:
        msg = str(e)
        logger.error(f"Failed to log correction to Langfuse: {msg}")
        return False, msg
