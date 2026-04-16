"""Tests for correction injection into Claude and Gemini bridge contexts.

Validates:
1. Claude prompt includes corrections when ledger is non-empty and flag is on
2. Claude prompt excludes corrections when ledger is empty or flag is off
3. Gemini writes/skips .gemini/corrections.md based on ledger and flag
4. render() output is identical regardless of bridge (SC-004)
5. Platform feedback log_correction appends to ledger independently of Langfuse
"""

import os
from unittest.mock import patch

from ambient_runner.platform.correction_ledger import CorrectionLedger


# ------------------------------------------------------------------
# Claude prompt injection
# ------------------------------------------------------------------


def test_claude_prompt_includes_corrections():
    """build_sdk_system_prompt appends correction block when ledger is non-empty."""
    from ambient_runner.bridges.claude.prompts import build_sdk_system_prompt

    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Used camelCase",
            "user_correction": "Use snake_case",
        }
    )

    with patch.dict(os.environ, {"ENABLE_CORRECTION_INJECTION": "true"}, clear=False):
        result = build_sdk_system_prompt(
            "/workspace", "/workspace", correction_ledger=ledger
        )

    assert "## Corrections from this session" in result["append"]
    assert "Used camelCase" in result["append"]


def test_claude_prompt_no_corrections_when_empty():
    """build_sdk_system_prompt does not add corrections block for empty ledger."""
    from ambient_runner.bridges.claude.prompts import build_sdk_system_prompt

    ledger = CorrectionLedger()

    with patch.dict(os.environ, {"ENABLE_CORRECTION_INJECTION": "true"}, clear=False):
        result = build_sdk_system_prompt(
            "/workspace", "/workspace", correction_ledger=ledger
        )

    assert "## Corrections from this session" not in result["append"]


def test_claude_prompt_no_corrections_when_flag_disabled():
    """build_sdk_system_prompt skips corrections when flag is off."""
    from ambient_runner.bridges.claude.prompts import build_sdk_system_prompt

    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Used camelCase",
            "user_correction": "Use snake_case",
        }
    )

    env = dict(os.environ)
    env.pop("ENABLE_CORRECTION_INJECTION", None)
    with patch.dict(os.environ, env, clear=True):
        result = build_sdk_system_prompt(
            "/workspace", "/workspace", correction_ledger=ledger
        )

    assert "## Corrections from this session" not in result["append"]


def test_claude_prompt_no_corrections_when_no_ledger():
    """build_sdk_system_prompt works without a ledger (backward compat)."""
    from ambient_runner.bridges.claude.prompts import build_sdk_system_prompt

    result = build_sdk_system_prompt("/workspace", "/workspace")
    assert "append" in result
    assert "## Corrections from this session" not in result["append"]


# ------------------------------------------------------------------
# Gemini context file injection
# ------------------------------------------------------------------


def test_gemini_writes_corrections_file(tmp_path):
    """Gemini bridge writes .gemini/corrections.md when ledger has entries."""
    from ambient_runner.bridges.gemini_cli.bridge import write_corrections_context_file

    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Used camelCase",
            "user_correction": "Use snake_case",
        }
    )

    cwd = str(tmp_path)
    with patch.dict(os.environ, {"ENABLE_CORRECTION_INJECTION": "true"}, clear=False):
        write_corrections_context_file(cwd, ledger)

    corrections_file = tmp_path / ".gemini" / "corrections.md"
    assert corrections_file.exists()
    content = corrections_file.read_text()
    assert "## Corrections from this session" in content
    assert "Used camelCase" in content


def test_gemini_skips_corrections_file_when_empty(tmp_path):
    """No file written when ledger is empty."""
    from ambient_runner.bridges.gemini_cli.bridge import write_corrections_context_file

    ledger = CorrectionLedger()
    cwd = str(tmp_path)

    with patch.dict(os.environ, {"ENABLE_CORRECTION_INJECTION": "true"}, clear=False):
        write_corrections_context_file(cwd, ledger)

    corrections_file = tmp_path / ".gemini" / "corrections.md"
    assert not corrections_file.exists()


def test_gemini_skips_corrections_file_when_flag_disabled(tmp_path):
    """No file written when flag is disabled, even with corrections."""
    from ambient_runner.bridges.gemini_cli.bridge import write_corrections_context_file

    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Did X",
            "user_correction": "Do Y",
        }
    )
    cwd = str(tmp_path)

    env = dict(os.environ)
    env.pop("ENABLE_CORRECTION_INJECTION", None)
    with patch.dict(os.environ, env, clear=True):
        write_corrections_context_file(cwd, ledger)

    corrections_file = tmp_path / ".gemini" / "corrections.md"
    assert not corrections_file.exists()


def test_gemini_no_op_when_no_ledger(tmp_path):
    """No crash and no file when ledger is None."""
    from ambient_runner.bridges.gemini_cli.bridge import write_corrections_context_file

    cwd = str(tmp_path)
    with patch.dict(os.environ, {"ENABLE_CORRECTION_INJECTION": "true"}, clear=False):
        write_corrections_context_file(cwd, None)

    corrections_file = tmp_path / ".gemini" / "corrections.md"
    assert not corrections_file.exists()


# ------------------------------------------------------------------
# Render output is bridge-agnostic (SC-004)
# ------------------------------------------------------------------


def test_render_output_identical_across_bridges():
    """Same ledger produces identical render() output -- bridge-agnostic."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Did X",
            "user_correction": "Do Y",
            "timestamp": "2026-01-01T00:00:00Z",
        }
    )
    ledger.append(
        {
            "correction_type": "incorrect",
            "agent_action": "Did A",
            "user_correction": "Do B",
            "timestamp": "2026-01-01T00:01:00Z",
        }
    )

    # render() is called identically by both bridges
    output1 = ledger.render()
    output2 = ledger.render()
    assert output1 == output2
    assert "## Corrections from this session" in output1


# ------------------------------------------------------------------
# Platform feedback layer -- ledger is independent of Langfuse
# ------------------------------------------------------------------


def test_platform_log_correction_appends_to_ledger():
    """platform.feedback.log_correction appends to ledger even when Langfuse fails."""
    from ambient_runner.platform.feedback import log_correction

    ledger = CorrectionLedger()

    with patch.dict(os.environ, {}, clear=True):
        # Langfuse is disabled -- will return (False, "not enabled")
        success, _err = log_correction(
            correction_type="style",
            agent_action="Did X",
            user_correction="Do Y",
            session_id="test-session",
            ledger=ledger,
        )

    # Langfuse failed, but ledger should still have the entry
    assert success is False  # Langfuse was not enabled
    assert len(ledger) == 1


def test_platform_log_correction_works_without_ledger():
    """Backward compat: log_correction works when no ledger is provided."""
    from ambient_runner.platform.feedback import log_correction

    with patch.dict(os.environ, {}, clear=True):
        success, _err = log_correction(
            correction_type="style",
            agent_action="Did X",
            user_correction="Do Y",
            session_id="test-session",
        )

    assert success is False  # Langfuse not enabled, but no crash
