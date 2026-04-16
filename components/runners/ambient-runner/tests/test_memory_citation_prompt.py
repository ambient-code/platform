#!/usr/bin/env python3
"""
Test memory citation prompt injection.

Validates:
1. Memory entries are tagged with unique PM-XXX IDs
2. Citation instruction block is included when memories exist
3. Citation instruction block is omitted when no memories exist
4. Memory metadata (author, date, correction ID) is included
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

from ambient_runner.platform.prompts import (
    build_memory_citation_prompt,
    format_memory_entry,
    MEMORY_CITATION_INSTRUCTIONS,
)


# ------------------------------------------------------------------
# format_memory_entry
# ------------------------------------------------------------------


def test_format_memory_entry_basic():
    """Memory entry is formatted with ID, content, and metadata."""
    entry = {
        "id": "PM-001",
        "content": "Always use gofmt before committing Go code.",
        "author": "jeder",
        "created_at": "2026-04-10",
        "correction_id": "corr-42",
    }
    result = format_memory_entry(entry)
    assert "[PM-001]" in result
    assert "Always use gofmt before committing Go code." in result
    assert "jeder" in result
    assert "2026-04-10" in result
    assert "corr-42" in result


def test_format_memory_entry_missing_optional_fields():
    """Memory entry renders gracefully when optional fields are missing."""
    entry = {
        "id": "PM-005",
        "content": "Use snake_case for Python variables.",
    }
    result = format_memory_entry(entry)
    assert "[PM-005]" in result
    assert "Use snake_case for Python variables." in result
    # Should not crash on missing author/date/correction_id


# ------------------------------------------------------------------
# build_memory_citation_prompt
# ------------------------------------------------------------------


def test_build_memory_citation_prompt_with_memories():
    """Citation prompt includes memory entries and instruction block."""
    memories = [
        {
            "id": "PM-001",
            "content": "Always use gofmt.",
            "author": "jeder",
            "created_at": "2026-04-10",
            "correction_id": "corr-42",
        },
        {
            "id": "PM-002",
            "content": "Run tests before pushing.",
            "author": "alice",
            "created_at": "2026-04-11",
        },
    ]
    result = build_memory_citation_prompt(memories)

    # Both entries present
    assert "[PM-001]" in result
    assert "[PM-002]" in result
    assert "Always use gofmt." in result
    assert "Run tests before pushing." in result

    # Citation instruction block present
    assert "[memory:PM-" in result
    assert MEMORY_CITATION_INSTRUCTIONS in result


def test_build_memory_citation_prompt_empty_list():
    """No prompt returned when memory list is empty."""
    result = build_memory_citation_prompt([])
    assert result == ""


def test_build_memory_citation_prompt_none():
    """No prompt returned when memories is None."""
    result = build_memory_citation_prompt(None)
    assert result == ""


def test_build_memory_citation_prompt_preserves_order():
    """Memory entries appear in the order provided."""
    memories = [
        {"id": "PM-003", "content": "Third entry."},
        {"id": "PM-001", "content": "First entry."},
        {"id": "PM-002", "content": "Second entry."},
    ]
    result = build_memory_citation_prompt(memories)
    pos_3 = result.index("[PM-003]")
    pos_1 = result.index("[PM-001]")
    pos_2 = result.index("[PM-002]")
    assert pos_3 < pos_1 < pos_2
