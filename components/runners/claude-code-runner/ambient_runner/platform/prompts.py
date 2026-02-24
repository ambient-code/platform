"""
System prompt construction and prompt constants for the Ambient Runner SDK.

Provides framework-agnostic workspace context prompts that any bridge can
use. Constants for tool descriptions and prompt fragments are defined here;
framework-specific wrapping (e.g. Claude Code preset format) belongs in the
bridge layer.
"""

import logging
import os
from pathlib import Path

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Prompt constants
# ---------------------------------------------------------------------------

WORKSPACE_STRUCTURE_HEADER = "# Workspace Structure\n\n"

MCP_INTEGRATIONS_PROMPT = (
    "## MCP Integrations\n"
    "If you need Google Drive access: Ask user to go to Integrations page "
    "in Ambient and authenticate with Google Drive.\n"
    "If you need Jira access: Ask user to go to Workspace Settings in Ambient "
    "and configure Jira credentials there.\n\n"
)

GIT_PUSH_INSTRUCTIONS_HEADER = "## Git Push Instructions\n\n"

GIT_PUSH_INSTRUCTIONS_BODY = (
    "The following repositories have auto-push enabled. When you make changes "
    "to these repositories, you MUST commit and push your changes:\n\n"
)

GIT_PUSH_STEPS = (
    "\nAfter making changes to any auto-push repository:\n"
    "1. Use `git add` to stage your changes\n"
    '2. Use `git commit -m "description"` to commit with a descriptive message\n'
    "3. Use `git push -u origin {branch}` to push to the remote repository\n"
    "   (this creates the branch on the remote if it doesn't exist yet)\n"
    "4. Create a pull request using `gh pr create` targeting the default branch\n\n"
    "**IMPORTANT**: NEVER push directly to `main` or `master`. Always work on "
    "the feature branch (`{branch}`). If push fails, do NOT fall back to main.\n\n"
)

RUBRIC_EVALUATION_HEADER = "## Rubric Evaluation\n\n"

RUBRIC_EVALUATION_INTRO = (
    "This workflow includes a scoring rubric for evaluating outputs. "
    "The rubric is located at `.ambient/rubric.md`.\n\n"
)

RUBRIC_EVALUATION_PROCESS = (
    "**Process**:\n"
    "1. Read `.ambient/rubric.md` using the Read tool\n"
    "2. Evaluate the output against each criterion\n"
    "3. Call `evaluate_rubric` (via the rubric MCP server) "
    "with your scores and reasoning\n\n"
    "**Important**: Always read the rubric first before scoring. "
    "Provide honest, calibrated scores with clear reasoning.\n\n"
)

RESTART_TOOL_DESCRIPTION = (
    "Restart the Claude session to recover from issues, clear state, "
    "or get a fresh connection. Use this if you detect you're in a "
    "broken state or need to reset."
)

CORRECTION_DETECTION_INSTRUCTIONS = (
    "## CRITICAL: Automatic Correction Logging\n\n"
    "**BEFORE acting on user feedback that changes something you already did or assumed, ALWAYS ask yourself: \"Is the user steering me away from a previous action or decision?\"**\n\n"
    "If YES → Call `log_correction` FIRST, then fix.\n\n"
    "**Rule: Any message that redirects, corrects, or changes your previous work or assumptions = log it. When in doubt, log it.**\n\n"
    "This includes quality issues, but also: redirections, preference changes, missed requirements, wrong scope, or any context that changes what you should have done. Do NOT treat these as simple new requests.\n"
)


# ---------------------------------------------------------------------------
# Prompt builder
# ---------------------------------------------------------------------------

def build_workspace_context_prompt(
    repos_cfg: list,
    workflow_name: str | None,
    artifacts_path: str,
    ambient_config: dict,
    workspace_path: str,
) -> str:
    """Generate the workspace context prompt appended to the Claude Code preset.

    Args:
        repos_cfg: List of repo config dicts.
        workflow_name: Active workflow name (or None).
        artifacts_path: Relative path for output artifacts.
        ambient_config: Parsed ambient.json dict.
        workspace_path: Absolute workspace root path.

    Returns:
        Formatted prompt string.
    """
    prompt = WORKSPACE_STRUCTURE_HEADER

    # Workflow directory
    if workflow_name:
        prompt += (
            f"**Working Directory**: workflows/{workflow_name}/ "
            "(workflow logic - do not create files here)\n\n"
        )

    # Artifacts
    prompt += f"**Artifacts**: {artifacts_path} (create all output files here)\n\n"

    # Uploaded files
    file_uploads_path = Path(workspace_path) / "file-uploads"
    if file_uploads_path.exists() and file_uploads_path.is_dir():
        try:
            files = sorted(
                [f.name for f in file_uploads_path.iterdir() if f.is_file()]
            )
            if files:
                max_display = 10
                if len(files) <= max_display:
                    prompt += f"**Uploaded Files**: {', '.join(files)}\n\n"
                else:
                    prompt += (
                        f"**Uploaded Files** ({len(files)} total): "
                        f"{', '.join(files[:max_display])}, "
                        f"and {len(files) - max_display} more\n\n"
                    )
        except Exception:
            pass
    else:
        prompt += "**Uploaded Files**: None\n\n"

    # Repositories
    if repos_cfg:
        session_id = os.getenv("AGENTIC_SESSION_NAME", "").strip()
        feature_branch = f"ambient/{session_id}" if session_id else None

        repo_names = [
            repo.get("name", f"repo-{i}") for i, repo in enumerate(repos_cfg)
        ]
        if len(repo_names) <= 5:
            prompt += (
                f"**Repositories**: "
                f"{', '.join([f'repos/{name}/' for name in repo_names])}\n"
            )
        else:
            prompt += (
                f"**Repositories** ({len(repo_names)} total): "
                f"{', '.join([f'repos/{name}/' for name in repo_names[:5]])}, "
                f"and {len(repo_names) - 5} more\n"
            )

        if feature_branch:
            prompt += (
                f"**Working Branch**: `{feature_branch}` "
                "(all repos are on this feature branch)\n\n"
            )
        else:
            prompt += "\n"

        # Git push instructions for auto-push repos
        auto_push_repos = [
            repo for repo in repos_cfg if repo.get("autoPush", False)
        ]
        if auto_push_repos:
            if not feature_branch:
                logger.warning(
                    "AGENTIC_SESSION_NAME not set; git-push prompt will "
                    "use placeholder branch name"
                )
            push_branch = feature_branch or "ambient/<session-name>"
            prompt += GIT_PUSH_INSTRUCTIONS_HEADER
            prompt += GIT_PUSH_INSTRUCTIONS_BODY
            for repo in auto_push_repos:
                repo_name = repo.get("name", "unknown")
                prompt += f"- **repos/{repo_name}/**\n"
            prompt += GIT_PUSH_STEPS.format(branch=push_branch)

    # MCP integration setup instructions
    prompt += MCP_INTEGRATIONS_PROMPT

    # Workflow instructions
    if ambient_config.get("systemPrompt"):
        prompt += (
            f"## Workflow Instructions\n"
            f"{ambient_config['systemPrompt']}\n\n"
        )

    # Rubric evaluation instructions
    prompt += _build_rubric_prompt_section(ambient_config)

    # Corrections feedback instructions (only when Langfuse is configured)
    langfuse_enabled = os.getenv("LANGFUSE_ENABLED", "").strip().lower() in (
        "1", "true", "yes"
    )
    if langfuse_enabled:
        prompt += "## Corrections Feedback\n\n"
        prompt += CORRECTION_DETECTION_INSTRUCTIONS

    return prompt


def _build_rubric_prompt_section(ambient_config: dict) -> str:
    """Build the rubric evaluation section for the system prompt.

    Returns empty string if no rubric config is present.
    """
    rubric_config = ambient_config.get("rubric", {})
    if not rubric_config:
        return ""

    section = RUBRIC_EVALUATION_HEADER
    section += RUBRIC_EVALUATION_INTRO

    activation_prompt = rubric_config.get("activationPrompt", "")
    if activation_prompt:
        section += f"**When to evaluate**: {activation_prompt}\n\n"

    section += RUBRIC_EVALUATION_PROCESS

    return section
