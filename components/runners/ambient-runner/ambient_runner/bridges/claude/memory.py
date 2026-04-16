"""
suggest_memory MCP tool for proposing new learned files.

When an agent discovers something worth remembering, it calls this tool
to write a markdown file to docs/learned/<type>s/<date>-<slug>.md on a
new branch and open a draft PR with the continuous-learning label.

The tool checks .ambient/config.json for learning.enabled before proceeding.
Git/GitHub failures are non-fatal -- they return an error message to the
agent but do not crash the session.
"""

import asyncio
import json
import logging
import os
import re
from datetime import datetime, timezone

logger = logging.getLogger(__name__)


# ------------------------------------------------------------------
# Constants
# ------------------------------------------------------------------

MEMORY_TYPES = ["correction", "pattern"]

SUGGEST_MEMORY_TOOL_DESCRIPTION = (
    "Suggest a new project memory to be reviewed and merged. Use this when you "
    "discover something worth remembering for future sessions: a non-obvious "
    "environment detail, a correction from the user, or an effective pattern.\n\n"
    "The memory is written as a markdown file and submitted as a draft PR for "
    "human review. It will be injected into future sessions once merged.\n\n"
    "Fields:\n"
    "- title: Short descriptive title for the memory\n"
    "- type: 'correction' (something that was wrong and should be avoided) or "
    "'pattern' (an effective approach worth repeating)\n"
    "- content: Detailed description of what was learned"
)


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------


def _slugify(text: str) -> str:
    """Convert text to a URL-safe slug."""
    slug = text.lower().strip()
    slug = re.sub(r"[^a-z0-9\s-]", "", slug)
    slug = re.sub(r"[\s]+", "-", slug)
    slug = re.sub(r"-+", "-", slug)
    slug = slug.strip("-")
    if not slug:
        return "untitled"
    return slug[:60]


def _generate_file_path(memory_type: str, title: str) -> str:
    """Generate the file path for a learned file."""
    date_str = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    slug = _slugify(title)
    type_dir = f"{memory_type}s"
    return f"docs/learned/{type_dir}/{date_str}-{slug}.md"


def _generate_frontmatter(
    title: str,
    memory_type: str,
    session_name: str,
) -> str:
    """Generate markdown frontmatter for a learned file."""
    date_str = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    return (
        "---\n"
        f"title: {title}\n"
        f"type: {memory_type}\n"
        f"date: {date_str}\n"
        f"source: agent\n"
        f"session: {session_name}\n"
        "---\n"
    )


def _check_cl_enabled(cwd_path: str) -> bool:
    """Check if continuous learning is enabled in .ambient/config.json."""
    config_path = os.path.join(cwd_path, ".ambient", "config.json")
    try:
        with open(config_path, "r") as f:
            config = json.load(f)
        learning = config.get("learning", {})
        return learning.get("enabled", False) is True
    except (FileNotFoundError, json.JSONDecodeError, OSError):
        return False


async def _run_git_command(
    args: list[str],
    cwd: str,
) -> tuple[bool, str]:
    """Run a git command asynchronously.

    Uses create_subprocess_exec (not shell) to avoid injection risks.
    Returns (success, output_or_error).
    """
    try:
        proc = await asyncio.create_subprocess_exec(
            "git",
            *args,
            cwd=cwd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            error = stderr.decode().strip() or stdout.decode().strip()
            return False, error
        return True, stdout.decode().strip()
    except Exception as e:
        return False, str(e)


async def _create_github_pr(
    cwd: str,
    branch_name: str,
    title: str,
    body: str,
) -> tuple[bool, str]:
    """Create a draft PR using the gh CLI.

    Uses create_subprocess_exec (not shell) to avoid injection risks.
    Returns (success, pr_url_or_error).
    """
    try:
        proc = await asyncio.create_subprocess_exec(
            "gh",
            "pr",
            "create",
            "--draft",
            "--title",
            title,
            "--body",
            body,
            "--label",
            "continuous-learning",
            "--head",
            branch_name,
            cwd=cwd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            error = stderr.decode().strip() or stdout.decode().strip()
            return False, error
        pr_url = stdout.decode().strip()
        return True, pr_url
    except Exception as e:
        return False, str(e)


# ------------------------------------------------------------------
# Tool factory
# ------------------------------------------------------------------


def create_suggest_memory_tool(
    sdk_tool_decorator,
    cwd_path: str,
    session_name: str,
):
    """Create the suggest_memory MCP tool.

    Args:
        sdk_tool_decorator: The ``tool`` decorator from ``claude_agent_sdk``.
        cwd_path: Working directory (repo root with .ambient/config.json).
        session_name: Current session name for PR attribution.

    Returns:
        Decorated async tool function.
    """

    input_schema: dict = {
        "type": "object",
        "properties": {
            "content": {
                "type": "string",
                "description": (
                    "Detailed description of what was learned. "
                    "Be specific and actionable."
                ),
            },
            "type": {
                "type": "string",
                "enum": MEMORY_TYPES,
                "description": (
                    "Type of memory: 'correction' (something wrong to avoid) "
                    "or 'pattern' (an effective approach to repeat)."
                ),
            },
            "title": {
                "type": "string",
                "description": "Short descriptive title for the memory.",
            },
        },
        "required": ["content", "type", "title"],
    }

    _cwd = cwd_path
    _session_name = session_name

    @sdk_tool_decorator(
        "suggest_memory",
        SUGGEST_MEMORY_TOOL_DESCRIPTION,
        input_schema,
    )
    async def suggest_memory_tool(args: dict) -> dict:
        """Suggest a new project memory via draft PR."""
        content = args.get("content", "").strip()
        memory_type = args.get("type", "").strip()
        title = args.get("title", "").strip()

        # --- Input validation ---

        if memory_type not in MEMORY_TYPES:
            return {
                "content": [
                    {
                        "type": "text",
                        "text": (
                            f"Invalid type '{memory_type}'. "
                            f"Valid types are: {', '.join(MEMORY_TYPES)}"
                        ),
                    }
                ],
                "isError": True,
            }

        if not content:
            return {
                "content": [{"type": "text", "text": "Content cannot be empty."}],
                "isError": True,
            }

        if not title:
            return {
                "content": [{"type": "text", "text": "Title cannot be empty."}],
                "isError": True,
            }

        # --- CL enabled check ---

        if not _check_cl_enabled(_cwd):
            return {
                "content": [
                    {
                        "type": "text",
                        "text": (
                            "Continuous learning is not enabled for this workspace. "
                            "Set learning.enabled=true in .ambient/config.json."
                        ),
                    }
                ],
                "isError": True,
            }

        # --- Generate file content ---

        file_path = _generate_file_path(memory_type, title)
        frontmatter = _generate_frontmatter(title, memory_type, _session_name)
        file_content = f"{frontmatter}\n{content}\n"

        date_str = datetime.now(timezone.utc).strftime("%Y-%m-%d")
        slug = _slugify(title)
        branch_name = f"learned/{memory_type}-{date_str}-{slug}"

        # --- Git operations ---

        original_branch = "main"
        try:
            # Save current branch to restore later
            ok, result = await _run_git_command(
                ["rev-parse", "--abbrev-ref", "HEAD"], _cwd
            )
            if ok and result:
                original_branch = result

            # Create and switch to new branch
            ok, err = await _run_git_command(["checkout", "-b", branch_name], _cwd)
            if not ok:
                return {
                    "content": [
                        {
                            "type": "text",
                            "text": f"Failed to create branch '{branch_name}': {err}",
                        }
                    ],
                    "isError": True,
                }

            # Create directories and write file
            full_path = os.path.join(_cwd, file_path)
            os.makedirs(os.path.dirname(full_path), exist_ok=True)
            with open(full_path, "w", encoding="utf-8") as f:
                f.write(file_content)

            # Stage and commit
            ok, err = await _run_git_command(["add", file_path], _cwd)
            if not ok:
                await _run_git_command(["checkout", original_branch], _cwd)
                return {
                    "content": [
                        {"type": "text", "text": f"Failed to stage file: {err}"}
                    ],
                    "isError": True,
                }

            commit_msg = f"learned({memory_type}): {title}"
            ok, err = await _run_git_command(["commit", "-m", commit_msg], _cwd)
            if not ok:
                await _run_git_command(["checkout", original_branch], _cwd)
                return {
                    "content": [{"type": "text", "text": f"Failed to commit: {err}"}],
                    "isError": True,
                }

            # Push branch
            ok, err = await _run_git_command(
                ["push", "-u", "origin", branch_name], _cwd
            )
            if not ok:
                await _run_git_command(["checkout", original_branch], _cwd)
                return {
                    "content": [
                        {"type": "text", "text": f"Failed to push branch: {err}"}
                    ],
                    "isError": True,
                }

            # Switch back to original branch
            await _run_git_command(["checkout", original_branch], _cwd)

            # Create draft PR
            pr_body = (
                f"## Suggested Memory\n\n"
                f"**Type:** {memory_type}\n"
                f"**Source:** Agent (session: {_session_name})\n\n"
                f"---\n\n"
                f"{content}\n"
            )
            ok, pr_result = await _create_github_pr(
                _cwd, branch_name, f"learned: {title}", pr_body
            )
            if not ok:
                logger.warning(f"PR creation failed (branch pushed): {pr_result}")
                return {
                    "content": [
                        {
                            "type": "text",
                            "text": (
                                f"Branch '{branch_name}' pushed successfully but "
                                f"draft PR creation failed: {pr_result}. "
                                "You can create the PR manually."
                            ),
                        }
                    ],
                }

            logger.info(f"Memory suggested: {memory_type}/{title} -> {pr_result}")
            return {
                "content": [
                    {
                        "type": "text",
                        "text": (
                            f"Memory suggested successfully! "
                            f"Draft PR created: {pr_result}"
                        ),
                    }
                ],
            }

        except Exception as e:
            logger.error(f"suggest_memory failed: {e}", exc_info=True)
            try:
                await _run_git_command(["checkout", original_branch], _cwd)
            except Exception:
                pass
            return {
                "content": [
                    {
                        "type": "text",
                        "text": f"Failed to suggest memory: {e}",
                    }
                ],
                "isError": True,
            }

    return suggest_memory_tool
