#!/usr/bin/env python3
"""
Test platform command injection logic.

Validates:
1. Commands are copied from source to target directory
2. Target directory is created when missing
3. No-op when source directory is absent
4. Only files (not subdirectories) are copied
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

from ambient_runner.platform.commands import inject_platform_commands


def test_injects_command_files(tmp_path):
    """Command .md files are copied from source to target."""
    source = tmp_path / "commands"
    source.mkdir()
    (source / "feedback.md").write_text("# feedback command")
    (source / "help.md").write_text("# help command")

    target = tmp_path / ".claude" / "commands"

    count = inject_platform_commands(source=source, target=target)

    assert count == 2
    assert (target / "feedback.md").exists()
    assert (target / "help.md").exists()
    assert (target / "feedback.md").read_text() == "# feedback command"


def test_creates_target_directory(tmp_path):
    """Target directory is created if it doesn't exist."""
    source = tmp_path / "commands"
    source.mkdir()
    (source / "feedback.md").write_text("content")

    target = tmp_path / "deep" / "nested" / "commands"
    assert not target.exists()

    inject_platform_commands(source=source, target=target)

    assert target.is_dir()
    assert (target / "feedback.md").exists()


def test_noop_when_source_missing(tmp_path):
    """Returns 0 when source directory doesn't exist."""
    source = tmp_path / "nonexistent"
    target = tmp_path / "target"

    count = inject_platform_commands(source=source, target=target)

    assert count == 0
    assert not target.exists()


def test_skips_subdirectories(tmp_path):
    """Only files are copied, subdirectories are skipped."""
    source = tmp_path / "commands"
    source.mkdir()
    (source / "feedback.md").write_text("command")
    (source / "subdir").mkdir()

    target = tmp_path / "target"

    count = inject_platform_commands(source=source, target=target)

    assert count == 1
    assert not (target / "subdir").exists()


def test_overwrites_existing_files(tmp_path):
    """Existing files in target are overwritten with source content."""
    source = tmp_path / "commands"
    source.mkdir()
    (source / "feedback.md").write_text("new content")

    target = tmp_path / "target"
    target.mkdir(parents=True)
    (target / "feedback.md").write_text("old content")

    inject_platform_commands(source=source, target=target)

    assert (target / "feedback.md").read_text() == "new content"
