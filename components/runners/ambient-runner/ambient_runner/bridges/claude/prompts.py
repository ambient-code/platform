"""
Claude-specific system prompt construction.

Wraps the platform workspace context prompt in the Claude Code SDK's
preset format (``type: "preset", preset: "claude_code"``).
"""

from ambient_runner.platform.prompts import (
    DEFAULT_AGENT_PREAMBLE,
    resolve_workspace_prompt,
)


def build_sdk_system_prompt(workspace_path: str, cwd_path: str) -> dict:
    """Build the full system prompt config dict for the Claude SDK.

    Wraps the platform workspace context prompt in the Claude Code preset.
    The DEFAULT_AGENT_PREAMBLE (overridable via AGENT_PREAMBLE env var) is
    prepended so it applies to every session regardless of workflow or prompt.
    """
    workspace_context = resolve_workspace_prompt(workspace_path, cwd_path)
    append_content = f"{DEFAULT_AGENT_PREAMBLE}\n\n{workspace_context}"
    return {
        "type": "preset",
        "preset": "claude_code",
        "append": append_content,
    }
