"""
Claude-specific system prompt construction.

Wraps the platform workspace context prompt in the Claude Code SDK's
preset format (``type: "preset", preset: "claude_code"``).
"""

from ambient_runner.platform.prompts import resolve_workspace_prompt


def build_sdk_system_prompt(workspace_path: str, cwd_path: str) -> dict:
    """Build the full system prompt config dict for the Claude SDK.

    Wraps the platform workspace context prompt in the Claude Code preset.
    """
    return {
        "type": "preset",
        "preset": "claude_code",
        "append": resolve_workspace_prompt(workspace_path, cwd_path),
    }
