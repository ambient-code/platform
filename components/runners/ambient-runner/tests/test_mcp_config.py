"""Tests for MCP config loading with custom/project/disabled server merging."""

import json
from pathlib import Path


from ambient_runner.platform.config import load_mcp_config
from ambient_runner.platform.context import RunnerContext


def _make_context(env: dict[str, str] | None = None) -> RunnerContext:
    """Create a minimal RunnerContext with optional env overrides."""
    return RunnerContext(
        session_id="test-session",
        workspace_path="/workspace",
        environment=env or {},
    )


class TestLoadMcpConfig:
    """Tests for load_mcp_config with custom/project/disabled server support."""

    def test_loads_from_default_file(self, tmp_path: Path):
        """Should load servers from .mcp.json file."""
        mcp_file = tmp_path / ".mcp.json"
        mcp_file.write_text(
            json.dumps(
                {
                    "mcpServers": {
                        "context7": {
                            "type": "http",
                            "url": "https://mcp.context7.com/mcp",
                        },
                        "webfetch": {"command": "uvx", "args": ["mcp-server-fetch"]},
                    }
                }
            )
        )
        ctx = _make_context({"MCP_CONFIG_FILE": str(mcp_file)})
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is not None
        assert "context7" in result
        assert "webfetch" in result
        assert result["context7"]["url"] == "https://mcp.context7.com/mcp"

    def test_merges_custom_session_servers(self, tmp_path: Path):
        """Session-level custom servers should be merged with defaults."""
        mcp_file = tmp_path / ".mcp.json"
        mcp_file.write_text(
            json.dumps(
                {
                    "mcpServers": {
                        "context7": {
                            "type": "http",
                            "url": "https://mcp.context7.com/mcp",
                        },
                    }
                }
            )
        )
        custom = {
            "custom": {
                "my-server": {"type": "http", "url": "https://example.com/mcp"},
            }
        }
        ctx = _make_context(
            {
                "MCP_CONFIG_FILE": str(mcp_file),
                "CUSTOM_MCP_SERVERS": json.dumps(custom),
            }
        )
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is not None
        assert "context7" in result
        assert "my-server" in result
        assert result["my-server"]["url"] == "https://example.com/mcp"

    def test_merges_project_level_servers(self, tmp_path: Path):
        """Project-level custom servers should be merged with defaults."""
        mcp_file = tmp_path / ".mcp.json"
        mcp_file.write_text(
            json.dumps(
                {
                    "mcpServers": {
                        "default-server": {"type": "http", "url": "https://default.com"}
                    }
                }
            )
        )

        project_mcp = {
            "custom": {
                "project-server": {"type": "http", "url": "https://project.com/mcp"},
            }
        }
        ctx = _make_context(
            {
                "MCP_CONFIG_FILE": str(mcp_file),
                "PROJECT_MCP_SERVERS": json.dumps(project_mcp),
            }
        )
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is not None
        assert "default-server" in result
        assert "project-server" in result

    def test_session_overrides_project(self, tmp_path: Path):
        """Session-level config should override project-level config for same server name."""
        mcp_file = tmp_path / ".mcp.json"
        mcp_file.write_text(json.dumps({"mcpServers": {}}))

        project_mcp = {
            "custom": {
                "shared": {"type": "http", "url": "https://project.com"},
            }
        }
        session_mcp = {
            "custom": {
                "shared": {"type": "http", "url": "https://session.com"},
            }
        }
        ctx = _make_context(
            {
                "MCP_CONFIG_FILE": str(mcp_file),
                "PROJECT_MCP_SERVERS": json.dumps(project_mcp),
                "CUSTOM_MCP_SERVERS": json.dumps(session_mcp),
            }
        )
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is not None
        assert result["shared"]["url"] == "https://session.com"

    def test_disables_default_servers(self, tmp_path: Path):
        """Disabled servers should be removed from the final config."""
        mcp_file = tmp_path / ".mcp.json"
        mcp_file.write_text(
            json.dumps(
                {
                    "mcpServers": {
                        "context7": {
                            "type": "http",
                            "url": "https://mcp.context7.com/mcp",
                        },
                        "deepwiki": {
                            "type": "http",
                            "url": "https://mcp.deepwiki.com/mcp",
                        },
                        "webfetch": {"command": "uvx", "args": ["mcp-server-fetch"]},
                    }
                }
            )
        )
        custom = {"disabled": ["context7", "deepwiki"]}
        ctx = _make_context(
            {
                "MCP_CONFIG_FILE": str(mcp_file),
                "CUSTOM_MCP_SERVERS": json.dumps(custom),
            }
        )
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is not None
        assert "context7" not in result
        assert "deepwiki" not in result
        assert "webfetch" in result

    def test_project_and_session_disabled_merge(self, tmp_path: Path):
        """Disabled lists from both project and session should be combined."""
        mcp_file = tmp_path / ".mcp.json"
        mcp_file.write_text(
            json.dumps(
                {
                    "mcpServers": {
                        "server-a": {"type": "http", "url": "https://a.com"},
                        "server-b": {"type": "http", "url": "https://b.com"},
                        "server-c": {"type": "http", "url": "https://c.com"},
                    }
                }
            )
        )
        project_mcp = {"disabled": ["server-a"]}
        session_mcp = {"disabled": ["server-b"]}
        ctx = _make_context(
            {
                "MCP_CONFIG_FILE": str(mcp_file),
                "PROJECT_MCP_SERVERS": json.dumps(project_mcp),
                "CUSTOM_MCP_SERVERS": json.dumps(session_mcp),
            }
        )
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is not None
        assert "server-a" not in result
        assert "server-b" not in result
        assert "server-c" in result

    def test_returns_none_when_all_disabled(self, tmp_path: Path):
        """Should return None when all servers are disabled."""
        mcp_file = tmp_path / ".mcp.json"
        mcp_file.write_text(
            json.dumps(
                {
                    "mcpServers": {
                        "only-server": {"type": "http", "url": "https://only.com"}
                    }
                }
            )
        )
        custom = {"disabled": ["only-server"]}
        ctx = _make_context(
            {
                "MCP_CONFIG_FILE": str(mcp_file),
                "CUSTOM_MCP_SERVERS": json.dumps(custom),
            }
        )
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is None

    def test_handles_invalid_custom_json(self, tmp_path: Path):
        """Should handle invalid JSON in CUSTOM_MCP_SERVERS gracefully."""
        mcp_file = tmp_path / ".mcp.json"
        mcp_file.write_text(
            json.dumps(
                {"mcpServers": {"s1": {"type": "http", "url": "https://s1.com"}}}
            )
        )
        ctx = _make_context(
            {
                "MCP_CONFIG_FILE": str(mcp_file),
                "CUSTOM_MCP_SERVERS": "not-json",
            }
        )
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is not None
        assert "s1" in result

    def test_no_mcp_file_with_custom(self, tmp_path: Path):
        """Should work with only custom servers when no default file exists."""
        custom = {
            "custom": {
                "custom-only": {"type": "http", "url": "https://custom.com"},
            }
        }
        ctx = _make_context(
            {
                "MCP_CONFIG_FILE": str(tmp_path / "nonexistent.json"),
                "CUSTOM_MCP_SERVERS": json.dumps(custom),
            }
        )
        result = load_mcp_config(ctx, str(tmp_path))
        assert result is not None
        assert "custom-only" in result
