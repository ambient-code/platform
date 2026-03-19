"""
Monkey-patch support for MockClaudeSDKClient with upstream ag-ui-claude-sdk.

When ANTHROPIC_API_KEY="mock-replay-key", this module patches
claude_agent_sdk.ClaudeSDKClient to return MockClaudeSDKClient instead,
allowing tests to run without a real API connection.

Import this module early in tests to activate the patch:
    from ambient_runner.bridges.claude import mock_patch
"""

import logging
import os
import sys

logger = logging.getLogger(__name__)

MOCK_API_KEY = "mock-replay-key"


def _should_use_mock() -> bool:
    """Check if mock client should be activated based on API key."""
    return os.getenv("ANTHROPIC_API_KEY", "").strip() == MOCK_API_KEY


def apply_mock_patch() -> None:
    """Replace ClaudeSDKClient with MockClaudeSDKClient globally.

    This must be called BEFORE ag_ui_claude_sdk.session imports ClaudeSDKClient.
    """
    if not _should_use_mock():
        return

    try:
        # Import our mock client
        from ambient_runner.bridges.claude.mock_client import MockClaudeSDKClient

        # Patch claude_agent_sdk module
        import claude_agent_sdk

        original_client = claude_agent_sdk.ClaudeSDKClient
        claude_agent_sdk.ClaudeSDKClient = MockClaudeSDKClient

        logger.info(
            "MockClaudeSDKClient activated (ANTHROPIC_API_KEY=mock-replay-key)"
        )

        # Store original for potential restoration
        if not hasattr(claude_agent_sdk, "_original_client"):
            claude_agent_sdk._original_client = original_client  # type: ignore

    except ImportError as e:
        logger.warning(f"Could not apply mock patch: {e}")


def restore_original_client() -> None:
    """Restore original ClaudeSDKClient (for test cleanup)."""
    try:
        import claude_agent_sdk

        if hasattr(claude_agent_sdk, "_original_client"):
            claude_agent_sdk.ClaudeSDKClient = claude_agent_sdk._original_client
            delattr(claude_agent_sdk, "_original_client")
            logger.info("Original ClaudeSDKClient restored")
    except ImportError:
        pass


# Auto-apply patch if mock API key is detected
if _should_use_mock():
    apply_mock_patch()
