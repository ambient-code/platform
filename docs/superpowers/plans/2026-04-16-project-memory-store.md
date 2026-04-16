# Project Memory Store (File-Based) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable the runner to read learned files from `docs/learned/` in a workspace repo at session init and inject them into the agent's system prompt as project memory, gated behind a feature flag and `.ambient/config.json` opt-in.

**Architecture:** The runner reads `.ambient/config.json` for opt-in, then parses markdown files with YAML frontmatter from `docs/learned/corrections/` and `docs/learned/patterns/`. Parsed entries are formatted into a `## Project Memory` prompt section and appended to the system prompt. The backend exposes a read-only `GET /learned` endpoint that reads the same files from the repo via GitHub API. Both paths are gated behind a `learning-agent-loop` feature flag.

**Tech Stack:** Python 3.12+ (runner), Go 1.24 (backend), PyYAML (already a transitive dep), Gin HTTP framework, pytest, Ginkgo/Gomega

---

## File Structure

### Runner (Python)

| File | Action | Responsibility |
|------|--------|----------------|
| `components/runners/ambient-runner/ambient_runner/platform/learned.py` | Create | Parse `.ambient/config.json`, read `docs/learned/**/*.md`, parse YAML frontmatter, format prompt section, enforce token budget |
| `components/runners/ambient-runner/ambient_runner/platform/prompts.py` | Modify | Call `build_project_memory_prompt()` from `learned.py` and append to workspace context prompt |
| `components/runners/ambient-runner/tests/test_learned.py` | Create | Unit tests for all learned.py functions |

### Backend (Go)

| File | Action | Responsibility |
|------|--------|----------------|
| `components/backend/handlers/learned.go` | Create | `ListLearnedEntries` handler — reads `docs/learned/` via GitHub API, parses frontmatter, returns JSON |
| `components/backend/handlers/learned_test.go` | Create | Unit tests for learned handler |
| `components/backend/routes.go` | Modify | Register `GET /projects/:projectName/learned` |

### Feature Flag

| File | Action | Responsibility |
|------|--------|----------------|
| `components/manifests/base/core/flags.json` | Modify | Add `learning-agent-loop` flag with `scope:workspace` tag |

---

### Task 1: Add the feature flag

**Files:**
- Modify: `components/manifests/base/core/flags.json`

- [ ] **Step 1: Add the learning-agent-loop flag to flags.json**

In `components/manifests/base/core/flags.json`, add to the `flags` array:

```json
{
  "name": "learning-agent-loop",
  "description": "Enable the learning agent loop: project memory injection from docs/learned/ files",
  "tags": [
    {
      "type": "scope",
      "value": "workspace"
    }
  ]
}
```

- [ ] **Step 2: Validate JSON syntax**

Run: `python3 -c "import json; json.load(open('components/manifests/base/core/flags.json'))"`
Expected: No output (valid JSON)

- [ ] **Step 3: Commit**

```bash
git add components/manifests/base/core/flags.json
git commit -m "feat(flags): add learning-agent-loop feature flag"
```

---

### Task 2: Runner — learned file parser module

**Files:**
- Create: `components/runners/ambient-runner/ambient_runner/platform/learned.py`
- Create: `components/runners/ambient-runner/tests/test_learned.py`

- [ ] **Step 1: Write tests for `is_learning_enabled()`**

Create `components/runners/ambient-runner/tests/test_learned.py`:

```python
"""Tests for project memory store (learned files)."""

import json
import os
from pathlib import Path

import pytest


class TestIsLearningEnabled:
    """Tests for .ambient/config.json opt-in check."""

    def test_enabled_true(self, tmp_path):
        from ambient_runner.platform.learned import is_learning_enabled

        config_dir = tmp_path / ".ambient"
        config_dir.mkdir()
        (config_dir / "config.json").write_text(
            json.dumps({"learning": {"enabled": True}})
        )
        assert is_learning_enabled(str(tmp_path)) is True

    def test_enabled_false(self, tmp_path):
        from ambient_runner.platform.learned import is_learning_enabled

        config_dir = tmp_path / ".ambient"
        config_dir.mkdir()
        (config_dir / "config.json").write_text(
            json.dumps({"learning": {"enabled": False}})
        )
        assert is_learning_enabled(str(tmp_path)) is False

    def test_missing_config_file(self, tmp_path):
        from ambient_runner.platform.learned import is_learning_enabled

        assert is_learning_enabled(str(tmp_path)) is False

    def test_malformed_json(self, tmp_path):
        from ambient_runner.platform.learned import is_learning_enabled

        config_dir = tmp_path / ".ambient"
        config_dir.mkdir()
        (config_dir / "config.json").write_text("not json {{{")
        assert is_learning_enabled(str(tmp_path)) is False

    def test_missing_learning_key(self, tmp_path):
        from ambient_runner.platform.learned import is_learning_enabled

        config_dir = tmp_path / ".ambient"
        config_dir.mkdir()
        (config_dir / "config.json").write_text(json.dumps({"other": "stuff"}))
        assert is_learning_enabled(str(tmp_path)) is False

    def test_learning_not_dict(self, tmp_path):
        from ambient_runner.platform.learned import is_learning_enabled

        config_dir = tmp_path / ".ambient"
        config_dir.mkdir()
        (config_dir / "config.json").write_text(
            json.dumps({"learning": "yes"})
        )
        assert is_learning_enabled(str(tmp_path)) is False
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py::TestIsLearningEnabled -v`
Expected: FAIL (module not found)

- [ ] **Step 3: Implement `is_learning_enabled()`**

Create `components/runners/ambient-runner/ambient_runner/platform/learned.py`:

```python
"""
Project Memory Store — read learned files from docs/learned/ and format
them for system prompt injection.

Learned files are markdown with YAML frontmatter stored in the workspace
repository under ``docs/learned/corrections/`` and ``docs/learned/patterns/``.
The feature is opt-in via ``.ambient/config.json``.
"""

from __future__ import annotations

import json
import logging
from pathlib import Path

import yaml

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Opt-in check
# ---------------------------------------------------------------------------


def is_learning_enabled(workspace_root: str) -> bool:
    """Check if learning is enabled via .ambient/config.json.

    Returns True only when the file exists, is valid JSON, and contains
    ``{"learning": {"enabled": true}}``. All failures return False
    (opt-in, non-fatal).
    """
    config_path = Path(workspace_root) / ".ambient" / "config.json"
    try:
        if not config_path.is_file():
            return False
        with open(config_path) as f:
            data = json.load(f)
        learning = data.get("learning")
        if not isinstance(learning, dict):
            return False
        return learning.get("enabled") is True
    except (json.JSONDecodeError, OSError, TypeError) as exc:
        logger.warning("Failed to read .ambient/config.json: %s", exc)
        return False
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py::TestIsLearningEnabled -v`
Expected: All 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add components/runners/ambient-runner/ambient_runner/platform/learned.py
git add components/runners/ambient-runner/tests/test_learned.py
git commit -m "feat(runner): add is_learning_enabled() config check"
```

---

### Task 3: Runner — learned file parsing

**Files:**
- Modify: `components/runners/ambient-runner/ambient_runner/platform/learned.py`
- Modify: `components/runners/ambient-runner/tests/test_learned.py`

- [ ] **Step 1: Write tests for `parse_learned_file()` and `read_learned_entries()`**

Append to `tests/test_learned.py`:

```python
class TestParseLearnedFile:
    """Tests for YAML frontmatter + markdown body parsing."""

    def test_valid_correction(self, tmp_path):
        from ambient_runner.platform.learned import parse_learned_file

        md = tmp_path / "test.md"
        md.write_text(
            "---\n"
            "type: correction\n"
            "date: 2026-04-01T14:30:00Z\n"
            "title: Use Pydantic v2\n"
            "session: session-1\n"
            "project: my-project\n"
            "author: Agent\n"
            "---\n"
            "\n"
            "Always use Pydantic v2 BaseModel.\n"
        )
        entry = parse_learned_file(md)
        assert entry is not None
        assert entry["type"] == "correction"
        assert entry["title"] == "Use Pydantic v2"
        assert entry["date"] == "2026-04-01T14:30:00Z"
        assert entry["session"] == "session-1"
        assert entry["project"] == "my-project"
        assert entry["author"] == "Agent"
        assert "Pydantic v2 BaseModel" in entry["content"]
        assert entry["file_path"] == str(md)

    def test_valid_pattern(self, tmp_path):
        from ambient_runner.platform.learned import parse_learned_file

        md = tmp_path / "pattern.md"
        md.write_text(
            "---\n"
            "type: pattern\n"
            "date: 2026-04-02T10:00:00Z\n"
            "title: Error response format\n"
            "---\n"
            "\n"
            "Use gin.H for error responses.\n"
        )
        entry = parse_learned_file(md)
        assert entry is not None
        assert entry["type"] == "pattern"
        assert entry["session"] == ""
        assert entry["project"] == ""
        assert entry["author"] == ""

    def test_missing_frontmatter(self, tmp_path):
        from ambient_runner.platform.learned import parse_learned_file

        md = tmp_path / "no_fm.md"
        md.write_text("Just some text, no frontmatter.\n")
        assert parse_learned_file(md) is None

    def test_missing_required_fields(self, tmp_path):
        from ambient_runner.platform.learned import parse_learned_file

        md = tmp_path / "bad.md"
        md.write_text("---\ntitle: No type field\n---\nBody.\n")
        assert parse_learned_file(md) is None

    def test_invalid_yaml(self, tmp_path):
        from ambient_runner.platform.learned import parse_learned_file

        md = tmp_path / "bad_yaml.md"
        md.write_text("---\n: [invalid yaml\n---\nBody.\n")
        assert parse_learned_file(md) is None


class TestReadLearnedEntries:
    """Tests for recursive directory reading."""

    def _make_file(self, path: Path, entry_type: str, title: str, date: str):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(
            f"---\ntype: {entry_type}\ndate: {date}\ntitle: {title}\n---\n\nBody for {title}.\n"
        )

    def test_reads_corrections_and_patterns(self, tmp_path):
        from ambient_runner.platform.learned import read_learned_entries

        self._make_file(
            tmp_path / "docs" / "learned" / "corrections" / "c1.md",
            "correction", "Fix A", "2026-04-01T00:00:00Z",
        )
        self._make_file(
            tmp_path / "docs" / "learned" / "patterns" / "p1.md",
            "pattern", "Pattern B", "2026-04-02T00:00:00Z",
        )
        entries = read_learned_entries(str(tmp_path))
        assert len(entries) == 2
        types = {e["type"] for e in entries}
        assert types == {"correction", "pattern"}

    def test_empty_directory(self, tmp_path):
        from ambient_runner.platform.learned import read_learned_entries

        (tmp_path / "docs" / "learned").mkdir(parents=True)
        entries = read_learned_entries(str(tmp_path))
        assert entries == []

    def test_no_directory(self, tmp_path):
        from ambient_runner.platform.learned import read_learned_entries

        entries = read_learned_entries(str(tmp_path))
        assert entries == []

    def test_skips_non_md_files(self, tmp_path):
        from ambient_runner.platform.learned import read_learned_entries

        learned = tmp_path / "docs" / "learned" / "corrections"
        learned.mkdir(parents=True)
        (learned / "notes.txt").write_text("not markdown")
        (learned / "image.png").write_bytes(b"\x89PNG")
        self._make_file(learned / "valid.md", "correction", "Valid", "2026-04-01T00:00:00Z")
        entries = read_learned_entries(str(tmp_path))
        assert len(entries) == 1

    def test_sorted_newest_first(self, tmp_path):
        from ambient_runner.platform.learned import read_learned_entries

        self._make_file(
            tmp_path / "docs" / "learned" / "corrections" / "old.md",
            "correction", "Old", "2026-01-01T00:00:00Z",
        )
        self._make_file(
            tmp_path / "docs" / "learned" / "corrections" / "new.md",
            "correction", "New", "2026-04-15T00:00:00Z",
        )
        entries = read_learned_entries(str(tmp_path))
        assert entries[0]["title"] == "New"
        assert entries[1]["title"] == "Old"
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py::TestParseLearnedFile tests/test_learned.py::TestReadLearnedEntries -v`
Expected: FAIL (functions not defined)

- [ ] **Step 3: Implement `parse_learned_file()` and `read_learned_entries()`**

Add to `components/runners/ambient-runner/ambient_runner/platform/learned.py`:

```python
# ---------------------------------------------------------------------------
# File parsing
# ---------------------------------------------------------------------------

_FRONTMATTER_DELIMITER = "---"
_REQUIRED_FIELDS = {"type", "date", "title"}


def parse_learned_file(file_path: Path) -> dict | None:
    """Parse a learned markdown file with YAML frontmatter.

    Returns a dict with frontmatter fields + ``content`` and ``file_path``,
    or None if the file is malformed or missing required fields.
    """
    try:
        text = file_path.read_text(encoding="utf-8")
    except OSError as exc:
        logger.warning("Cannot read learned file %s: %s", file_path, exc)
        return None

    # Split frontmatter from body
    if not text.startswith(_FRONTMATTER_DELIMITER):
        logger.warning("No frontmatter in %s", file_path)
        return None

    parts = text.split(_FRONTMATTER_DELIMITER, maxsplit=2)
    if len(parts) < 3:
        logger.warning("Malformed frontmatter in %s", file_path)
        return None

    yaml_text = parts[1]
    body = parts[2].strip()

    try:
        meta = yaml.safe_load(yaml_text)
    except yaml.YAMLError as exc:
        logger.warning("Invalid YAML frontmatter in %s: %s", file_path, exc)
        return None

    if not isinstance(meta, dict):
        logger.warning("Frontmatter is not a mapping in %s", file_path)
        return None

    # Validate required fields
    missing = _REQUIRED_FIELDS - set(meta.keys())
    if missing:
        logger.warning("Missing required fields %s in %s", missing, file_path)
        return None

    return {
        "type": str(meta.get("type", "")),
        "date": str(meta.get("date", "")),
        "title": str(meta.get("title", "")),
        "session": str(meta.get("session", "") or ""),
        "project": str(meta.get("project", "") or ""),
        "author": str(meta.get("author", "") or ""),
        "content": body,
        "file_path": str(file_path),
    }


def read_learned_entries(workspace_root: str) -> list[dict]:
    """Read all learned markdown files from docs/learned/.

    Recurses into subdirectories (corrections/, patterns/). Parses each
    ``.md`` file, skips invalid ones with warnings. Returns entries sorted
    by date descending (newest first).
    """
    learned_dir = Path(workspace_root) / "docs" / "learned"
    if not learned_dir.is_dir():
        return []

    entries: list[dict] = []
    try:
        for md_file in sorted(learned_dir.rglob("*.md")):
            if not md_file.is_file():
                continue
            entry = parse_learned_file(md_file)
            if entry is not None:
                entries.append(entry)
    except OSError as exc:
        logger.warning("Error reading docs/learned/: %s", exc)

    # Sort newest first by date string (ISO 8601 sorts lexicographically)
    entries.sort(key=lambda e: e.get("date", ""), reverse=True)
    return entries
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py::TestParseLearnedFile tests/test_learned.py::TestReadLearnedEntries -v`
Expected: All 10 tests PASS

- [ ] **Step 5: Commit**

```bash
git add components/runners/ambient-runner/ambient_runner/platform/learned.py
git add components/runners/ambient-runner/tests/test_learned.py
git commit -m "feat(runner): parse learned files from docs/learned/"
```

---

### Task 4: Runner — prompt formatting with token budget

**Files:**
- Modify: `components/runners/ambient-runner/ambient_runner/platform/learned.py`
- Modify: `components/runners/ambient-runner/tests/test_learned.py`

- [ ] **Step 1: Write tests for `build_project_memory_prompt()`**

Append to `tests/test_learned.py`:

```python
class TestBuildProjectMemoryPrompt:
    """Tests for prompt section formatting."""

    def test_corrections_and_patterns(self):
        from ambient_runner.platform.learned import build_project_memory_prompt

        entries = [
            {"type": "correction", "title": "Fix A", "content": "Do X instead of Y.", "date": "2026-04-01T00:00:00Z"},
            {"type": "pattern", "title": "Pattern B", "content": "Always use pattern Z.", "date": "2026-04-02T00:00:00Z"},
        ]
        result = build_project_memory_prompt(entries)
        assert "## Project Memory" in result
        assert "### Corrections" in result
        assert "### Patterns" in result
        assert "Fix A" in result
        assert "Pattern B" in result

    def test_only_corrections(self):
        from ambient_runner.platform.learned import build_project_memory_prompt

        entries = [
            {"type": "correction", "title": "Fix A", "content": "Body.", "date": "2026-04-01T00:00:00Z"},
        ]
        result = build_project_memory_prompt(entries)
        assert "### Corrections" in result
        assert "### Patterns" not in result

    def test_only_patterns(self):
        from ambient_runner.platform.learned import build_project_memory_prompt

        entries = [
            {"type": "pattern", "title": "Pat A", "content": "Body.", "date": "2026-04-01T00:00:00Z"},
        ]
        result = build_project_memory_prompt(entries)
        assert "### Patterns" in result
        assert "### Corrections" not in result

    def test_empty_entries(self):
        from ambient_runner.platform.learned import build_project_memory_prompt

        result = build_project_memory_prompt([])
        assert result == ""

    def test_token_budget_truncation(self):
        from ambient_runner.platform.learned import build_project_memory_prompt

        # Create many entries that exceed the ~4000 token budget
        entries = []
        for i in range(100):
            entries.append({
                "type": "correction",
                "title": f"Correction {i}",
                "content": "A" * 200,  # ~50 tokens each
                "date": f"2026-04-{i % 28 + 1:02d}T00:00:00Z",
            })
        result = build_project_memory_prompt(entries)
        # Should contain truncation note
        assert "additional" in result.lower() or "more" in result.lower()
        # Should be under budget (4000 tokens ~ 16000 chars rough upper bound)
        assert len(result) < 20000

    def test_unknown_type_skipped(self):
        from ambient_runner.platform.learned import build_project_memory_prompt

        entries = [
            {"type": "unknown", "title": "X", "content": "Body.", "date": "2026-04-01T00:00:00Z"},
        ]
        result = build_project_memory_prompt(entries)
        assert result == ""
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py::TestBuildProjectMemoryPrompt -v`
Expected: FAIL (function not defined)

- [ ] **Step 3: Implement `build_project_memory_prompt()`**

Add to `components/runners/ambient-runner/ambient_runner/platform/learned.py`:

```python
# ---------------------------------------------------------------------------
# Prompt formatting
# ---------------------------------------------------------------------------

# Approximate token budget for injected memories.
# ~4 chars per token is a conservative estimate for English text.
_TOKEN_BUDGET = 4000
_CHARS_PER_TOKEN = 4
_CHAR_BUDGET = _TOKEN_BUDGET * _CHARS_PER_TOKEN


def _format_entry(entry: dict) -> str:
    """Format a single learned entry as a markdown block."""
    title = entry.get("title", "Untitled")
    content = entry.get("content", "").strip()
    return f"**{title}**: {content}\n"


def build_project_memory_prompt(entries: list[dict]) -> str:
    """Format learned entries into a ``## Project Memory`` prompt section.

    Groups entries by type (correction, pattern). Enforces a character
    budget (~4000 tokens). Entries are assumed to be pre-sorted by date
    (newest first). Returns empty string if no valid entries.
    """
    if not entries:
        return ""

    corrections = [e for e in entries if e.get("type") == "correction"]
    patterns = [e for e in entries if e.get("type") == "pattern"]

    if not corrections and not patterns:
        return ""

    sections: list[str] = []
    total_chars = 0
    included_count = 0
    total_count = len(corrections) + len(patterns)

    header = "## Project Memory\n\n"
    total_chars += len(header)

    for section_title, section_entries in [
        ("### Corrections", corrections),
        ("### Patterns", patterns),
    ]:
        if not section_entries:
            continue

        section_header = f"{section_title}\n\n"
        section_lines: list[str] = []
        section_chars = len(section_header)

        for entry in section_entries:
            formatted = _format_entry(entry)
            entry_chars = len(formatted)
            if total_chars + section_chars + entry_chars > _CHAR_BUDGET:
                break
            section_lines.append(formatted)
            section_chars += entry_chars
            included_count += 1

        if section_lines:
            sections.append(section_header + "\n".join(section_lines) + "\n")
            total_chars += section_chars

    if not sections:
        return ""

    result = header + "\n".join(sections)

    omitted = total_count - included_count
    if omitted > 0:
        result += f"\n*({omitted} additional entries omitted due to token budget)*\n"

    return result
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py::TestBuildProjectMemoryPrompt -v`
Expected: All 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add components/runners/ambient-runner/ambient_runner/platform/learned.py
git add components/runners/ambient-runner/tests/test_learned.py
git commit -m "feat(runner): format project memory prompt with token budget"
```

---

### Task 5: Runner — integrate into system prompt

**Files:**
- Modify: `components/runners/ambient-runner/ambient_runner/platform/prompts.py`
- Modify: `components/runners/ambient-runner/tests/test_learned.py`

- [ ] **Step 1: Write integration test for prompt injection**

Append to `tests/test_learned.py`:

```python
class TestGetProjectMemoryPrompt:
    """Tests for the top-level entry point with feature flag gating."""

    def _make_file(self, path: Path, entry_type: str, title: str, date: str):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(
            f"---\ntype: {entry_type}\ndate: {date}\ntitle: {title}\n---\n\nBody for {title}.\n"
        )

    def test_returns_prompt_when_enabled(self, tmp_path):
        from ambient_runner.platform.learned import get_project_memory_prompt

        # Enable learning
        config_dir = tmp_path / ".ambient"
        config_dir.mkdir()
        (config_dir / "config.json").write_text(
            json.dumps({"learning": {"enabled": True}})
        )
        self._make_file(
            tmp_path / "docs" / "learned" / "corrections" / "c1.md",
            "correction", "Fix A", "2026-04-01T00:00:00Z",
        )
        result = get_project_memory_prompt(str(tmp_path))
        assert "## Project Memory" in result
        assert "Fix A" in result

    def test_returns_empty_when_disabled(self, tmp_path):
        from ambient_runner.platform.learned import get_project_memory_prompt

        config_dir = tmp_path / ".ambient"
        config_dir.mkdir()
        (config_dir / "config.json").write_text(
            json.dumps({"learning": {"enabled": False}})
        )
        self._make_file(
            tmp_path / "docs" / "learned" / "corrections" / "c1.md",
            "correction", "Fix A", "2026-04-01T00:00:00Z",
        )
        result = get_project_memory_prompt(str(tmp_path))
        assert result == ""

    def test_returns_empty_when_no_config(self, tmp_path):
        from ambient_runner.platform.learned import get_project_memory_prompt

        self._make_file(
            tmp_path / "docs" / "learned" / "corrections" / "c1.md",
            "correction", "Fix A", "2026-04-01T00:00:00Z",
        )
        result = get_project_memory_prompt(str(tmp_path))
        assert result == ""

    def test_non_fatal_on_error(self, tmp_path):
        from ambient_runner.platform.learned import get_project_memory_prompt

        # Enable learning but point at a non-existent workspace
        config_dir = tmp_path / ".ambient"
        config_dir.mkdir()
        (config_dir / "config.json").write_text(
            json.dumps({"learning": {"enabled": True}})
        )
        # No docs/learned/ dir — should return empty, not raise
        result = get_project_memory_prompt(str(tmp_path))
        assert result == ""
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py::TestGetProjectMemoryPrompt -v`
Expected: FAIL (function not defined)

- [ ] **Step 3: Implement `get_project_memory_prompt()`**

Add to `components/runners/ambient-runner/ambient_runner/platform/learned.py`:

```python
# ---------------------------------------------------------------------------
# Top-level entry point
# ---------------------------------------------------------------------------


def get_project_memory_prompt(workspace_root: str) -> str:
    """Build the project memory prompt section if learning is enabled.

    This is the main entry point called from the prompt builder. It:
    1. Checks .ambient/config.json for opt-in
    2. Reads all learned files from docs/learned/
    3. Formats them into a ## Project Memory section

    All failures are non-fatal — returns empty string on any error.
    """
    try:
        if not is_learning_enabled(workspace_root):
            return ""

        entries = read_learned_entries(workspace_root)
        if not entries:
            return ""

        return build_project_memory_prompt(entries)
    except Exception as exc:
        logger.warning("Failed to build project memory prompt: %s", exc)
        return ""
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py::TestGetProjectMemoryPrompt -v`
Expected: All 4 tests PASS

- [ ] **Step 5: Integrate into `build_workspace_context_prompt()`**

In `components/runners/ambient-runner/ambient_runner/platform/prompts.py`, add the project memory section at the end of `build_workspace_context_prompt()`, just before the `return prompt` statement. Find this block near the end of the function:

```python
    # Corrections feedback instructions (only when Langfuse is configured)
    from ambient_runner.observability import is_langfuse_enabled

    if is_langfuse_enabled():
        prompt += "## Corrections Feedback\n\n"
        prompt += CORRECTION_DETECTION_INSTRUCTIONS

    return prompt
```

Replace with:

```python
    # Corrections feedback instructions (only when Langfuse is configured)
    from ambient_runner.observability import is_langfuse_enabled

    if is_langfuse_enabled():
        prompt += "## Corrections Feedback\n\n"
        prompt += CORRECTION_DETECTION_INSTRUCTIONS

    # Project memory injection (learned files from docs/learned/)
    from ambient_runner.platform.learned import get_project_memory_prompt

    memory_prompt = get_project_memory_prompt(workspace_path)
    if memory_prompt:
        prompt += memory_prompt

    return prompt
```

- [ ] **Step 6: Run all learned tests**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py -v`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add components/runners/ambient-runner/ambient_runner/platform/learned.py
git add components/runners/ambient-runner/ambient_runner/platform/prompts.py
git add components/runners/ambient-runner/tests/test_learned.py
git commit -m "feat(runner): inject project memory into system prompt"
```

---

### Task 6: Runner — add pyyaml as explicit dependency

**Files:**
- Modify: `components/runners/ambient-runner/pyproject.toml`

- [ ] **Step 1: Add pyyaml to dependencies**

PyYAML is currently a transitive dependency. Add it as an explicit dependency since the learned module directly imports it. In `pyproject.toml`, find the `dependencies` list:

```toml
dependencies = [
  # Ambient Runner SDK core
  "fastapi>=0.135.1",
  "uvicorn[standard]>=0.41.0",
  "ag-ui-protocol>=0.1.13",
  "pydantic>=2.12.5",
  "aiohttp>=3.13.4",
  "requests>=2.32.5",
  "pyjwt>=2.11.0",
]
```

Add `"pyyaml>=6.0.3",` after `"pyjwt>=2.11.0",`:

```toml
dependencies = [
  # Ambient Runner SDK core
  "fastapi>=0.135.1",
  "uvicorn[standard]>=0.41.0",
  "ag-ui-protocol>=0.1.13",
  "pydantic>=2.12.5",
  "aiohttp>=3.13.4",
  "requests>=2.32.5",
  "pyjwt>=2.11.0",
  "pyyaml>=6.0.3",
]
```

- [ ] **Step 2: Add ambient_runner.platform to setuptools packages if needed**

Check the `packages` list in `pyproject.toml` — `ambient_runner.platform` is already listed. No change needed.

- [ ] **Step 3: Commit**

```bash
git add components/runners/ambient-runner/pyproject.toml
git commit -m "feat(runner): add pyyaml as explicit dependency for learned files"
```

---

### Task 7: Backend — learned entries endpoint

**Files:**
- Create: `components/backend/handlers/learned.go`
- Modify: `components/backend/routes.go`

- [ ] **Step 1: Create the ListLearnedEntries handler**

Create `components/backend/handlers/learned.go`:

```go
package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// LearnedEntry represents a parsed learned file entry
type LearnedEntry struct {
	Type     string `json:"type"`
	Date     string `json:"date"`
	Title    string `json:"title"`
	Session  string `json:"session,omitempty"`
	Project  string `json:"project,omitempty"`
	Author   string `json:"author,omitempty"`
	Content  string `json:"content"`
	FilePath string `json:"filePath"`
}

// GetGitHubTokenLearned is a dependency-injectable function for getting GitHub tokens in learned operations.
// Tests can override this to provide mock implementations.
var GetGitHubTokenLearned func(ctx interface{}, k8s kubernetes.Interface, dyn dynamic.Interface, project, userID string) (string, error)

// parseFrontmatter extracts YAML-like frontmatter key-value pairs from a markdown string.
// Returns the frontmatter map and the body text after the closing "---".
func parseFrontmatter(content string) (map[string]string, string) {
	if !strings.HasPrefix(content, "---") {
		return nil, content
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, content
	}

	fm := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(parts[1]), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes
		val = strings.Trim(val, "\"'")
		if key != "" {
			fm[key] = val
		}
	}

	body := strings.TrimSpace(parts[2])
	return fm, body
}

// ListLearnedEntries handles GET /api/projects/:projectName/learned
// Reads docs/learned/ from the workspace repo via GitHub API and returns parsed entries.
func ListLearnedEntries(c *gin.Context) {
	project := c.Param("projectName")
	repo := c.Query("repo")
	ref := c.Query("ref")
	typeFilter := c.Query("type")

	if repo == "" || ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo and ref query parameters required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)

	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	// Get GitHub token
	var token string
	var err error
	getTokenFn := GetGitHubTokenLearned
	if getTokenFn == nil {
		getTokenFn = func(ctx interface{}, k8s kubernetes.Interface, dyn dynamic.Interface, proj, uid string) (string, error) {
			return GetGitHubTokenRepo(c.Request.Context(), k8s, dyn, proj, uid)
		}
	}
	token, err = getTokenFn(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
	if err != nil {
		log.Printf("Failed to get GitHub token for learned endpoint, project %s: %v", project, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	owner, repoName, err := parseOwnerRepo(repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch the docs/learned/ directory tree from GitHub
	entries, err := fetchLearnedFiles(c, owner, repoName, ref, token)
	if err != nil {
		// If the directory doesn't exist, return empty array (not 404)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			c.JSON(http.StatusOK, gin.H{"entries": []LearnedEntry{}})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to fetch learned files: %v", err)})
		return
	}

	// Apply type filter
	if typeFilter != "" {
		filtered := make([]LearnedEntry, 0, len(entries))
		for _, e := range entries {
			if e.Type == typeFilter {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

// fetchLearnedFiles fetches and parses learned files from the GitHub API.
func fetchLearnedFiles(c *gin.Context, owner, repo, ref, token string) ([]LearnedEntry, error) {
	api := githubAPIBaseURL("github.com")

	// Get directory listing for docs/learned/
	url := fmt.Sprintf("%s/repos/%s/%s/contents/docs/learned?ref=%s", api, owner, repo, ref)
	resp, err := doGitHubRequest(c.Request.Context(), http.MethodGet, url, "Bearer "+token, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("404 Not Found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(b))
	}

	// Parse directory listing — may be array (directory) or object (single file)
	var decoded interface{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	// Collect all .md file paths from the directory tree (recursive)
	mdPaths := collectMDPaths(decoded)

	// Also check subdirectories (corrections/, patterns/)
	for _, subdir := range []string{"corrections", "patterns"} {
		subURL := fmt.Sprintf("%s/repos/%s/%s/contents/docs/learned/%s?ref=%s", api, owner, repo, subdir, ref)
		subResp, subErr := doGitHubRequest(c.Request.Context(), http.MethodGet, subURL, "Bearer "+token, "", nil)
		if subErr != nil {
			continue
		}
		defer subResp.Body.Close()
		if subResp.StatusCode == http.StatusOK {
			var subDecoded interface{}
			if json.NewDecoder(subResp.Body).Decode(&subDecoded) == nil {
				mdPaths = append(mdPaths, collectMDPaths(subDecoded)...)
			}
		}
	}

	// Deduplicate paths
	seen := make(map[string]bool)
	uniquePaths := make([]string, 0, len(mdPaths))
	for _, p := range mdPaths {
		if !seen[p] {
			seen[p] = true
			uniquePaths = append(uniquePaths, p)
		}
	}

	// Fetch and parse each file
	var entries []LearnedEntry
	for _, filePath := range uniquePaths {
		fileURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repo, filePath, ref)
		fileResp, fileErr := doGitHubRequest(c.Request.Context(), http.MethodGet, fileURL, "Bearer "+token, "", nil)
		if fileErr != nil {
			log.Printf("Failed to fetch learned file %s: %v", filePath, fileErr)
			continue
		}
		defer fileResp.Body.Close()

		if fileResp.StatusCode != http.StatusOK {
			continue
		}

		var fileObj map[string]interface{}
		if json.NewDecoder(fileResp.Body).Decode(&fileObj) != nil {
			continue
		}

		rawContent, _ := fileObj["content"].(string)
		encoding, _ := fileObj["encoding"].(string)

		var textContent string
		if strings.ToLower(encoding) == "base64" {
			raw := strings.ReplaceAll(rawContent, "\n", "")
			data, decErr := base64.StdEncoding.DecodeString(raw)
			if decErr != nil {
				continue
			}
			textContent = string(data)
		} else {
			textContent = rawContent
		}

		fm, body := parseFrontmatter(textContent)
		if fm == nil {
			continue
		}

		entryType := fm["type"]
		title := fm["title"]
		date := fm["date"]
		if entryType == "" || title == "" || date == "" {
			continue
		}

		entries = append(entries, LearnedEntry{
			Type:     entryType,
			Date:     date,
			Title:    title,
			Session:  fm["session"],
			Project:  fm["project"],
			Author:   fm["author"],
			Content:  body,
			FilePath: filePath,
		})
	}

	return entries, nil
}

// collectMDPaths extracts .md file paths from a GitHub API directory listing.
func collectMDPaths(decoded interface{}) []string {
	var paths []string

	switch v := decoded.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				name, _ := m["name"].(string)
				path, _ := m["path"].(string)
				typ, _ := m["type"].(string)
				if strings.ToLower(typ) == "file" && strings.HasSuffix(strings.ToLower(name), ".md") {
					paths = append(paths, path)
				}
			}
		}
	case map[string]interface{}:
		name, _ := v["name"].(string)
		path, _ := v["path"].(string)
		typ, _ := v["type"].(string)
		if strings.ToLower(typ) == "file" && strings.HasSuffix(strings.ToLower(name), ".md") {
			paths = append(paths, path)
		}
	}

	return paths
}
```

- [ ] **Step 2: Register the route**

In `components/backend/routes.go`, add inside the `projectGroup` block, after the feature flags block and before the GitLab auth block:

Find:
```go
			// GitLab authentication endpoints (DEPRECATED - moved to cluster-scoped)
```

Add before it:
```go
			// Learned files endpoint (project memory store)
			projectGroup.GET("/learned", handlers.ListLearnedEntries)

```

- [ ] **Step 3: Verify Go compilation**

Run: `cd components/backend && go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add components/backend/handlers/learned.go
git add components/backend/routes.go
git commit -m "feat(backend): add GET /learned endpoint for project memory"
```

---

### Task 8: Backend — learned endpoint tests

**Files:**
- Create: `components/backend/handlers/learned_test.go`

- [ ] **Step 1: Write unit tests for parseFrontmatter**

Create `components/backend/handlers/learned_test.go`:

```go
//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Learned Handler >", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {

	Describe("parseFrontmatter", func() {
		It("parses valid frontmatter with all fields", func() {
			content := "---\ntype: correction\ndate: 2026-04-01T14:30:00Z\ntitle: Use Pydantic v2\nsession: session-1\nproject: my-project\nauthor: Agent\n---\n\nAlways use Pydantic v2 BaseModel."
			fm, body := parseFrontmatter(content)
			Expect(fm).NotTo(BeNil())
			Expect(fm["type"]).To(Equal("correction"))
			Expect(fm["date"]).To(Equal("2026-04-01T14:30:00Z"))
			Expect(fm["title"]).To(Equal("Use Pydantic v2"))
			Expect(fm["session"]).To(Equal("session-1"))
			Expect(fm["project"]).To(Equal("my-project"))
			Expect(fm["author"]).To(Equal("Agent"))
			Expect(body).To(Equal("Always use Pydantic v2 BaseModel."))
		})

		It("handles missing frontmatter", func() {
			content := "Just plain text."
			fm, body := parseFrontmatter(content)
			Expect(fm).To(BeNil())
			Expect(body).To(Equal("Just plain text."))
		})

		It("handles incomplete frontmatter delimiters", func() {
			content := "---\ntype: correction\nNo closing delimiter"
			fm, body := parseFrontmatter(content)
			Expect(fm).To(BeNil())
			Expect(body).To(Equal(content))
		})

		It("strips quotes from values", func() {
			content := "---\ntitle: \"Quoted Title\"\ntype: 'pattern'\ndate: 2026-04-01\n---\n\nBody."
			fm, body := parseFrontmatter(content)
			Expect(fm).NotTo(BeNil())
			Expect(fm["title"]).To(Equal("Quoted Title"))
			Expect(fm["type"]).To(Equal("pattern"))
			Expect(body).To(Equal("Body."))
		})

		It("handles empty body", func() {
			content := "---\ntype: pattern\ntitle: Empty\ndate: 2026-04-01\n---\n"
			fm, body := parseFrontmatter(content)
			Expect(fm).NotTo(BeNil())
			Expect(fm["type"]).To(Equal("pattern"))
			Expect(body).To(Equal(""))
		})
	})

	Describe("collectMDPaths", func() {
		It("collects .md files from array", func() {
			input := []interface{}{
				map[string]interface{}{"name": "fix.md", "path": "docs/learned/corrections/fix.md", "type": "file"},
				map[string]interface{}{"name": "readme.txt", "path": "docs/learned/readme.txt", "type": "file"},
				map[string]interface{}{"name": "corrections", "path": "docs/learned/corrections", "type": "dir"},
			}
			paths := collectMDPaths(input)
			Expect(paths).To(HaveLen(1))
			Expect(paths[0]).To(Equal("docs/learned/corrections/fix.md"))
		})

		It("returns empty for empty array", func() {
			paths := collectMDPaths([]interface{}{})
			Expect(paths).To(BeEmpty())
		})

		It("handles single file object", func() {
			input := map[string]interface{}{"name": "fix.md", "path": "docs/learned/fix.md", "type": "file"}
			paths := collectMDPaths(input)
			Expect(paths).To(HaveLen(1))
		})
	})
})
```

- [ ] **Step 2: Verify Go tests compile**

Run: `cd components/backend && go test -tags test -run "Learned" -count=0 ./handlers/`
Expected: No compilation errors (tests may or may not run depending on test suite setup)

- [ ] **Step 3: Commit**

```bash
git add components/backend/handlers/learned_test.go
git commit -m "test(backend): add learned endpoint unit tests"
```

---

### Task 9: Run all tests and verify

**Files:** (no new files)

- [ ] **Step 1: Run all runner tests**

Run: `cd components/runners/ambient-runner && python -m pytest tests/test_learned.py -v`
Expected: All tests PASS

- [ ] **Step 2: Run Go vet and build**

Run: `cd components/backend && go vet ./... && go build ./...`
Expected: No errors

- [ ] **Step 3: Verify flags.json is valid**

Run: `python3 -c "import json; d=json.load(open('components/manifests/base/core/flags.json')); print(f'{len(d[\"flags\"])} flags'); assert any(f['name']=='learning-agent-loop' for f in d['flags'])"`
Expected: Prints flag count and no assertion error

- [ ] **Step 4: Final commit (if any unstaged changes)**

```bash
git status
# If clean, skip. Otherwise:
git add -A
git commit -m "chore: final cleanup for project memory store"
```
