"""Write a Gemini CLI system prompt that appends to the default.

Sets GEMINI_SYSTEM_MD=true and writes .gemini/system.md.

Since GEMINI_SYSTEM_MD is a full replacement, we use the documented variable
substitutions to pull in Gemini's dynamically-built default content, then
append platform additions after it:

    ${SubAgents}      — injects the current available sub-agents section
    ${AgentSkills}    — injects the current available agent skills section
    ${AvailableTools} — injects the current enabled tools list

This gives true "append" semantics without hardcoding any static default text
that could drift with Gemini CLI version updates.
"""

import logging
import os
from pathlib import Path

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Platform-specific prompt fragments
# ---------------------------------------------------------------------------

_GEMINI_COMMAND_HINTS = """\
## Ambient Platform Commands

Use these slash commands when applicable:

- `/ambient:evaluate-rubric` — when a workflow rubric exists, run this after
  completing the task to evaluate your output against the criteria and log the
  score to Langfuse.
- `/ambient:log-correction` — when the user redirects or corrects something
  you already did or assumed, run this BEFORE fixing the issue to log the
  correction to the improvement feedback loop.

"""

_CORRECTION_DETECTION = """\
## CRITICAL: Correction Logging

**BEFORE acting on user feedback that changes something you already did or
assumed, ALWAYS ask yourself: "Is the user steering me away from a previous
action or decision?"**

If YES → run `/ambient:log-correction` FIRST, then fix.

**Rule**: Any message that redirects, corrects, or changes your previous work
or assumptions = log it. When in doubt, log it. This includes quality issues,
redirections, preference changes, missed requirements, wrong scope, or any
context that changes what you should have done.

"""

_RUBRIC_INSTRUCTIONS_TEMPLATE = """\
## Rubric Evaluation

This workflow includes a scoring rubric for evaluating outputs.{activation}

**Process**:
1. Complete the requested task
2. Run `/ambient:evaluate-rubric` — it will read the rubric and guide you
   through scoring your output against each criterion
3. Be honest and calibrated; provide clear reasoning for each dimension

"""


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def write_gemini_system_prompt(cwd_path: str) -> str:
    """Write .gemini/system.md and return its absolute path.

    The file starts with Gemini's variable substitutions (${SubAgents},
    ${AgentSkills}, ${AvailableTools}) to pull in the dynamically-built
    default content, then appends platform-specific context.

    Also sets os.environ["GEMINI_SYSTEM_MD"] = "true" so every Gemini CLI
    subprocess spawned in this session inherits it.

    Args:
        cwd_path: The session working directory (Gemini project root).

    Returns:
        Absolute path to the written system.md, or empty string on failure.
    """
    content = _build_system_prompt(cwd_path)

    gemini_dir = Path(cwd_path) / ".gemini"
    gemini_dir.mkdir(parents=True, exist_ok=True)
    system_md = gemini_dir / "system.md"

    try:
        system_md.write_text(content, encoding="utf-8")
        system_md.chmod(0o644)
    except OSError as exc:
        logger.warning("Could not write .gemini/system.md: %s", exc)
        return ""

    abs_path = str(system_md.resolve())
    os.environ["GEMINI_SYSTEM_MD"] = "true"
    logger.info("Wrote Gemini system prompt to %s", abs_path)
    return abs_path


# ---------------------------------------------------------------------------
# Content builder
# ---------------------------------------------------------------------------


def _build_system_prompt(cwd_path: str) -> str:
    """Build the full system.md content string."""
    from ambient_runner.platform.config import get_repos_config, load_ambient_config
    from ambient_runner.platform.prompts import (
        GITHUB_TOKEN_PROMPT,
        GITLAB_TOKEN_PROMPT,
        GIT_PUSH_INSTRUCTIONS_BODY,
        GIT_PUSH_INSTRUCTIONS_HEADER,
        GIT_PUSH_STEPS,
        MCP_INTEGRATIONS_PROMPT,
        WORKSPACE_FIXED_PATHS_PROMPT,
    )
    from ambient_runner.platform.utils import derive_workflow_name

    # Pull in Gemini's dynamically-built default sections via variable substitution.
    # These are expanded at runtime by the CLI — no static text to maintain.
    sections = [
        "${SubAgents}",
        "",
        "${AgentSkills}",
        "",
        "${AvailableTools}",
        "",
        "---",
        "",
        "# Ambient Code Platform",
        "",
    ]

    # ---- Workspace paths ----
    sections.append("## Workspace Structure\n")
    sections.append(WORKSPACE_FIXED_PATHS_PROMPT)

    # ---- Repos + git push instructions ----
    repos_cfg = get_repos_config()
    active_workflow_url = (os.getenv("ACTIVE_WORKFLOW_GIT_URL") or "").strip()
    session_id = os.getenv("AGENTIC_SESSION_NAME", "").strip()
    feature_branch = f"ambient/{session_id}" if session_id else None

    if repos_cfg:
        repo_names = [r.get("name", f"repo-{i}") for i, r in enumerate(repos_cfg)]
        display = [f"repos/{n}/" for n in repo_names]
        if len(display) <= 5:
            sections.append(f"**Repositories**: {', '.join(display)}\n")
        else:
            sections.append(
                f"**Repositories** ({len(display)} total): "
                f"{', '.join(display[:5])}, and {len(display) - 5} more\n"
            )
        if feature_branch:
            sections.append(f"**Working Branch**: `{feature_branch}`\n")
        sections.append("")

        auto_push = [r for r in repos_cfg if r.get("autoPush", False)]
        if auto_push:
            branch = feature_branch or "ambient/<session-name>"
            sections.append(GIT_PUSH_INSTRUCTIONS_HEADER)
            sections.append(GIT_PUSH_INSTRUCTIONS_BODY)
            for r in auto_push:
                sections.append(f"- **repos/{r.get('name', 'unknown')}/**")
            sections.append(GIT_PUSH_STEPS.format(branch=branch))

    # ---- Workflow directory ----
    if active_workflow_url:
        workflow_name = derive_workflow_name(active_workflow_url)
        if workflow_name:
            sections.append(
                f"**Working Directory**: `workflows/{workflow_name}/` "
                "(workflow logic — do not create files here)\n"
            )

    # ---- File uploads ----
    uploads = Path(os.getenv("WORKSPACE_PATH", "/workspace")) / "file-uploads"
    if uploads.exists():
        try:
            files = sorted(f.name for f in uploads.iterdir() if f.is_file())
            if files:
                if len(files) <= 10:
                    sections.append(f"**Uploaded Files**: {', '.join(files)}\n")
                else:
                    sections.append(
                        f"**Uploaded Files** ({len(files)} total): "
                        f"{', '.join(files[:10])}, and {len(files) - 10} more\n"
                    )
        except Exception as exc:
            logger.warning("Could not list uploaded files in %s: %s", uploads, exc)

    # ---- MCP integration hints ----
    sections.append(MCP_INTEGRATIONS_PROMPT)

    # ---- Token visibility ----
    if os.getenv("GITHUB_TOKEN"):
        sections.append(GITHUB_TOKEN_PROMPT)
    if os.getenv("GITLAB_TOKEN"):
        sections.append(GITLAB_TOKEN_PROMPT)

    # ---- Workflow custom instructions ----
    ambient_config: dict = {}
    if active_workflow_url:
        ambient_config = load_ambient_config(cwd_path) or {}
    if ambient_config.get("systemPrompt"):
        sections.append(f"## Workflow Instructions\n\n{ambient_config['systemPrompt']}\n")

    # ---- Rubric instructions (when rubric config exists) ----
    rubric_config = ambient_config.get("rubric", {})
    if rubric_config:
        activation = rubric_config.get("activationPrompt", "")
        activation_str = f" {activation}" if activation else ""
        sections.append(_RUBRIC_INSTRUCTIONS_TEMPLATE.format(activation=activation_str))

    # ---- Custom command hints (always included) ----
    sections.append(_GEMINI_COMMAND_HINTS)

    # ---- Corrections (when Langfuse enabled) ----
    from ambient_runner.observability import is_langfuse_enabled

    if is_langfuse_enabled():
        sections.append(_CORRECTION_DETECTION)

    return "\n".join(sections)
