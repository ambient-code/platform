"""
Claude-specific system prompt construction.

Wraps the platform workspace context prompt in the Claude Code SDK's
preset format (``type: "preset", preset: "claude_code"``).
"""

import logging
import os

from ambient_runner.platform.config import get_repos_config, load_ambient_config
from ambient_runner.platform.prompts import build_workspace_context_prompt

logger = logging.getLogger(__name__)


def build_sdk_system_prompt(workspace_path: str, cwd_path: str) -> dict:
    """Build the full system prompt config dict for the Claude SDK.

    Wraps the platform workspace context prompt in the Claude Code preset.
    """
    repos_cfg = get_repos_config()
    active_workflow_url = (os.getenv("ACTIVE_WORKFLOW_GIT_URL") or "").strip()
    ambient_config = (
        load_ambient_config(cwd_path) if active_workflow_url else {}
    )

    derived_name = None
    if active_workflow_url:
        derived_name = active_workflow_url.split("/")[-1].removesuffix(".git")

    workspace_prompt = build_workspace_context_prompt(
        repos_cfg=repos_cfg,
        workflow_name=derived_name if active_workflow_url else None,
        artifacts_path="artifacts",
        ambient_config=ambient_config,
        workspace_path=workspace_path,
    )

    return {
        "type": "preset",
        "preset": "claude_code",
        "append": workspace_prompt,
    }
