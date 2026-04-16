"""Tests for CorrectionLedger -- the per-session in-memory correction store.

Validates:
1. Empty ledger renders empty string (SC-006)
2. Single correction renders correctly (SC-006)
3. 20+ corrections trigger cap and summary (SC-002, SC-006)
4. Field truncation at 500 chars (SC-006)
5. render() output is identical regardless of bridge (SC-004)
6. Edge cases: empty fields, timestamp generation, singular/plural summary
"""

import re

from ambient_runner.platform.correction_ledger import CorrectionLedger


# ------------------------------------------------------------------
# Empty ledger
# ------------------------------------------------------------------


def test_empty_ledger_renders_empty_string():
    """An empty ledger renders nothing -- no header, no content."""
    ledger = CorrectionLedger()
    assert ledger.render() == ""


def test_empty_ledger_len_is_zero():
    """Empty ledger has length 0."""
    ledger = CorrectionLedger()
    assert len(ledger) == 0


# ------------------------------------------------------------------
# Single correction
# ------------------------------------------------------------------


def test_single_correction_renders_block():
    """A single correction produces a ## header and one entry."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Used camelCase",
            "user_correction": "Use snake_case",
        }
    )
    output = ledger.render()
    assert output.startswith("## Corrections from this session")
    assert "style" in output
    assert "Used camelCase" in output
    assert "Use snake_case" in output


def test_single_correction_len_is_one():
    """Ledger length is 1 after one append."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Did X",
            "user_correction": "Do Y",
        }
    )
    assert len(ledger) == 1


# ------------------------------------------------------------------
# Ordering
# ------------------------------------------------------------------


def test_corrections_in_reverse_chronological_order():
    """Most recent correction appears first in rendered output."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "incorrect",
            "agent_action": "First action",
            "user_correction": "First fix",
        }
    )
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Second action",
            "user_correction": "Second fix",
        }
    )
    output = ledger.render()
    second_pos = output.index("Second action")
    first_pos = output.index("First action")
    assert second_pos < first_pos, "Most recent correction should appear first"


# ------------------------------------------------------------------
# Cap at 20
# ------------------------------------------------------------------


def test_cap_at_20_with_summary():
    """When >20 corrections exist, only 20 render with a summary of omitted."""
    ledger = CorrectionLedger()
    for i in range(25):
        ledger.append(
            {
                "correction_type": "style",
                "agent_action": f"Action {i}",
                "user_correction": f"Fix {i}",
            }
        )
    output = ledger.render()
    # Most recent 20 should be present (indices 24 down to 5)
    assert "Action 24" in output
    assert "Action 5" in output
    # Oldest 5 should NOT be individually listed
    assert "Action 4" not in output
    assert "Action 0" not in output
    # Summary line for omitted entries
    assert "5 earlier corrections omitted" in output


def test_exactly_20_no_summary():
    """With exactly 20 corrections, all render and no summary line appears."""
    ledger = CorrectionLedger()
    for i in range(20):
        ledger.append(
            {
                "correction_type": "incomplete",
                "agent_action": f"Action {i}",
                "user_correction": f"Fix {i}",
            }
        )
    output = ledger.render()
    assert "Action 0" in output
    assert "Action 19" in output
    assert "omitted" not in output


def test_21_corrections_summary_singular():
    """21 corrections => 20 shown + '1 earlier correction omitted' (singular)."""
    ledger = CorrectionLedger()
    for i in range(21):
        ledger.append(
            {
                "correction_type": "style",
                "agent_action": f"Action {i}",
                "user_correction": f"Fix {i}",
            }
        )
    output = ledger.render()
    assert "1 earlier correction omitted" in output
    # Plural "corrections" should NOT appear for the singular case
    assert "1 earlier corrections" not in output


# ------------------------------------------------------------------
# Field truncation
# ------------------------------------------------------------------


def test_field_truncation_at_500_chars():
    """agent_action and user_correction are truncated to 500 characters."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "incorrect",
            "agent_action": "A" * 1000,
            "user_correction": "B" * 1000,
        }
    )
    output = ledger.render()
    # The full 1000-char strings should NOT appear
    assert "A" * 1000 not in output
    assert "B" * 1000 not in output
    # But the truncated 500-char versions should
    assert "A" * 500 in output
    assert "B" * 500 in output


# ------------------------------------------------------------------
# Empty fields
# ------------------------------------------------------------------


def test_empty_fields_render_placeholder():
    """Empty agent_action or user_correction renders '(not specified)'."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "",
            "user_correction": "",
        }
    )
    output = ledger.render()
    assert "(not specified)" in output


# ------------------------------------------------------------------
# Timestamp
# ------------------------------------------------------------------


def test_timestamp_auto_generated():
    """Each entry includes an auto-generated ISO-format timestamp."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Did X",
            "user_correction": "Do Y",
        }
    )
    output = ledger.render()
    assert re.search(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", output)


def test_custom_timestamp_preserved():
    """If a timestamp is provided in the entry, it is used as-is."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Did X",
            "user_correction": "Do Y",
            "timestamp": "2026-01-01T00:00:00Z",
        }
    )
    output = ledger.render()
    assert "2026-01-01T00:00:00Z" in output


# ------------------------------------------------------------------
# Size constraint
# ------------------------------------------------------------------


def test_render_output_under_4000_chars_for_20_entries():
    """SC-003: 20 entries with reasonable text stays under 4000 chars."""
    ledger = CorrectionLedger()
    for i in range(20):
        ledger.append(
            {
                "correction_type": "style",
                "agent_action": f"Used wrong pattern in function_{i} -- applied X instead",
                "user_correction": f"Should use pattern Y for function_{i} per conventions",
            }
        )
    output = ledger.render()
    assert len(output) < 4000, f"Rendered output is {len(output)} chars, exceeds 4000"


# ------------------------------------------------------------------
# Determinism
# ------------------------------------------------------------------


def test_render_is_deterministic():
    """Same ledger state always produces identical output."""
    ledger = CorrectionLedger()
    ledger.append(
        {
            "correction_type": "style",
            "agent_action": "Did X",
            "user_correction": "Do Y",
            "timestamp": "2026-01-01T00:00:00Z",
        }
    )
    output1 = ledger.render()
    output2 = ledger.render()
    assert output1 == output2
