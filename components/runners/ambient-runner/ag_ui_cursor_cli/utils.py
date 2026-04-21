# components/runners/ambient-runner/ag_ui_cursor_cli/utils.py
"""Utility functions for Cursor CLI adapter."""

# extract_user_message is identical across adapters — re-export from the
# shared gemini utils rather than duplicating.
from ag_ui_gemini_cli.utils import extract_user_message

__all__ = ["extract_user_message"]
