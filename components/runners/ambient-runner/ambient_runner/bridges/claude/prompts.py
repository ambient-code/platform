"""
Claude-specific system prompt construction.

Wraps the platform workspace context prompt in the Claude Code SDK's
preset format (``type: "preset", preset: "claude_code"``).
"""

from __future__ import annotations

from typing import TYPE_CHECKING

from ambient_runner.platform.correction_ledger import is_correction_injection_enabled
from ambient_runner.platform.prompts import resolve_workspace_prompt

if TYPE_CHECKING:
    from ambient_runner.platform.correction_ledger import CorrectionLedger


def build_sdk_system_prompt(
    workspace_path: str,
    cwd_path: str,
    correction_ledger: CorrectionLedger | None = None,
) -> dict:
    """Build the full system prompt config dict for the Claude SDK.

    Wraps the platform workspace context prompt in the Claude Code preset.
    When a correction ledger is provided and the feature flag is enabled,
    appends the rendered correction block to the prompt.
    """
    prompt = resolve_workspace_prompt(workspace_path, cwd_path)

    if correction_ledger is not None and is_correction_injection_enabled():
        block = correction_ledger.render()
        if block:
            prompt += "\n\n" + block

    return {
        "type": "preset",
        "preset": "claude_code",
        "append": prompt,
    }
