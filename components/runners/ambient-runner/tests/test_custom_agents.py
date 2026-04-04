"""Unit tests for custom agent loading from .claude/agents/*.md files."""

import textwrap
from pathlib import Path
from unittest.mock import patch

import pytest

from ambient_runner.bridges.claude.agents import (
    _parse_agent_file,
    _parse_string_list,
    load_agents_from_directory,
)


# ------------------------------------------------------------------
# _parse_agent_file
# ------------------------------------------------------------------


class TestParseAgentFile:
    """Verify frontmatter + body extraction from markdown files."""

    def test_valid_frontmatter_and_body(self, tmp_path: Path):
        md = tmp_path / "agent.md"
        md.write_text(
            textwrap.dedent("""\
                ---
                name: test-agent
                description: A test agent
                tools: Read, Write, Bash
                model: sonnet
                ---

                You are a helpful test agent.
            """)
        )

        metadata, body = _parse_agent_file(md)

        assert metadata["name"] == "test-agent"
        assert metadata["description"] == "A test agent"
        assert metadata["tools"] == "Read, Write, Bash"
        assert metadata["model"] == "sonnet"
        assert body == "You are a helpful test agent."

    def test_no_frontmatter(self, tmp_path: Path):
        md = tmp_path / "plain.md"
        md.write_text("Just a plain markdown file.\n")

        metadata, body = _parse_agent_file(md)

        assert metadata == {}
        assert body == "Just a plain markdown file."

    def test_empty_file(self, tmp_path: Path):
        md = tmp_path / "empty.md"
        md.write_text("")

        metadata, body = _parse_agent_file(md)

        assert metadata == {}
        assert body == ""

    def test_frontmatter_only_no_body(self, tmp_path: Path):
        md = tmp_path / "no-body.md"
        md.write_text("---\nname: solo\n---\n")

        metadata, body = _parse_agent_file(md)

        assert metadata["name"] == "solo"
        assert body == ""

    def test_missing_file(self, tmp_path: Path):
        md = tmp_path / "nonexistent.md"

        metadata, body = _parse_agent_file(md)

        assert metadata == {}
        assert body == ""

    def test_quoted_values_stripped(self, tmp_path: Path):
        md = tmp_path / "quoted.md"
        md.write_text('---\nname: "my-agent"\ndescription: \'does things\'\n---\nBody.\n')

        metadata, body = _parse_agent_file(md)

        assert metadata["name"] == "my-agent"
        assert metadata["description"] == "does things"

    def test_multiline_body(self, tmp_path: Path):
        md = tmp_path / "multi.md"
        md.write_text("---\nname: multi\n---\nLine one.\n\nLine two.\n")

        metadata, body = _parse_agent_file(md)

        assert metadata["name"] == "multi"
        assert "Line one." in body
        assert "Line two." in body


# ------------------------------------------------------------------
# _parse_string_list
# ------------------------------------------------------------------


class TestParseStringList:
    """Verify comma-separated string → list conversion."""

    def test_comma_separated(self):
        assert _parse_string_list("Read, Write, Bash") == ["Read", "Write", "Bash"]

    def test_single_item(self):
        assert _parse_string_list("Read") == ["Read"]

    def test_none_returns_none(self):
        assert _parse_string_list(None) is None

    def test_empty_string_returns_none(self):
        assert _parse_string_list("") is None

    def test_list_passthrough(self):
        assert _parse_string_list(["Read", "Write"]) == ["Read", "Write"]

    def test_whitespace_trimmed(self):
        assert _parse_string_list("  Read ,  Write  ") == ["Read", "Write"]

    def test_empty_items_filtered(self):
        assert _parse_string_list("Read,,Write,") == ["Read", "Write"]


# ------------------------------------------------------------------
# load_agents_from_directory
# ------------------------------------------------------------------


class TestLoadAgentsFromDirectory:
    """Verify end-to-end agent loading from a directory of .md files."""

    def _write_agent(self, agents_dir: Path, filename: str, content: str) -> None:
        agents_dir.mkdir(parents=True, exist_ok=True)
        (agents_dir / filename).write_text(textwrap.dedent(content))

    @patch("ambient_runner.bridges.claude.agents.AgentDefinition")
    def test_loads_multiple_agents(self, mock_agent_def, tmp_path: Path):
        agents_dir = tmp_path / "agents"
        self._write_agent(
            agents_dir,
            "researcher.md",
            """\
            ---
            name: researcher
            description: Research agent for web lookups
            tools: Read, WebSearch, WebFetch
            model: sonnet
            ---

            You are a research assistant.
            """,
        )
        self._write_agent(
            agents_dir,
            "writer.md",
            """\
            ---
            name: docs-writer
            description: Technical documentation writer
            tools: Read, Write, Edit
            skills: vale-lint
            ---

            You write documentation.
            """,
        )

        result = load_agents_from_directory(str(agents_dir))

        assert mock_agent_def.call_count == 2

        # Verify researcher call
        researcher_call = mock_agent_def.call_args_list[0]
        assert researcher_call.kwargs["description"] == "Research agent for web lookups"
        assert researcher_call.kwargs["prompt"] == "You are a research assistant."
        assert researcher_call.kwargs["tools"] == ["Read", "WebSearch", "WebFetch"]
        assert researcher_call.kwargs["model"] == "sonnet"
        assert researcher_call.kwargs["skills"] is None

        # Verify writer call
        writer_call = mock_agent_def.call_args_list[1]
        assert writer_call.kwargs["description"] == "Technical documentation writer"
        assert writer_call.kwargs["prompt"] == "You write documentation."
        assert writer_call.kwargs["tools"] == ["Read", "Write", "Edit"]
        assert writer_call.kwargs["skills"] == ["vale-lint"]

        assert "researcher" in result
        assert "docs-writer" in result

    @patch("ambient_runner.bridges.claude.agents.AgentDefinition")
    def test_fallback_name_from_filename(self, mock_agent_def, tmp_path: Path):
        agents_dir = tmp_path / "agents"
        self._write_agent(
            agents_dir,
            "my-agent.md",
            """\
            ---
            description: An agent without a name field
            ---

            Do things.
            """,
        )

        result = load_agents_from_directory(str(agents_dir))

        assert "my-agent" in result
        assert mock_agent_def.call_args.kwargs["description"] == "An agent without a name field"

    @patch("ambient_runner.bridges.claude.agents.AgentDefinition")
    def test_fallback_description(self, mock_agent_def, tmp_path: Path):
        agents_dir = tmp_path / "agents"
        self._write_agent(
            agents_dir,
            "bare.md",
            """\
            ---
            name: bare-agent
            ---

            Minimal agent.
            """,
        )

        result = load_agents_from_directory(str(agents_dir))

        assert mock_agent_def.call_args.kwargs["description"] == "Agent: bare-agent"

    def test_nonexistent_directory_returns_empty(self, tmp_path: Path):
        result = load_agents_from_directory(str(tmp_path / "does-not-exist"))
        assert result == {}

    def test_empty_directory_returns_empty(self, tmp_path: Path):
        agents_dir = tmp_path / "agents"
        agents_dir.mkdir()

        result = load_agents_from_directory(str(agents_dir))
        assert result == {}

    @patch("ambient_runner.bridges.claude.agents.AgentDefinition")
    def test_ignores_non_md_files(self, mock_agent_def, tmp_path: Path):
        agents_dir = tmp_path / "agents"
        agents_dir.mkdir()
        (agents_dir / "readme.txt").write_text("Not an agent.")
        self._write_agent(
            agents_dir,
            "real.md",
            """\
            ---
            name: real
            description: A real agent
            ---

            Hello.
            """,
        )

        result = load_agents_from_directory(str(agents_dir))

        assert len(result) == 1
        assert "real" in result

    @patch("ambient_runner.bridges.claude.agents.AgentDefinition", side_effect=Exception("boom"))
    def test_bad_file_logged_and_skipped(self, mock_agent_def, tmp_path: Path):
        agents_dir = tmp_path / "agents"
        self._write_agent(
            agents_dir,
            "bad.md",
            """\
            ---
            name: bad-agent
            ---

            Will fail.
            """,
        )

        result = load_agents_from_directory(str(agents_dir))
        assert result == {}
