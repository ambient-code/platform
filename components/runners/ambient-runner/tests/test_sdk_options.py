"""Unit tests for SDK_OPTIONS env var parsing in ClaudeBridge."""

import json
import os
from unittest.mock import patch, MagicMock


class TestSdkOptionsEnvVar:
    """Test SDK_OPTIONS env var parsing in _ensure_adapter."""

    def _make_bridge(self):
        """Create a ClaudeBridge with minimal platform state."""
        from ambient_runner.bridges.claude.bridge import ClaudeBridge

        bridge = ClaudeBridge()
        # Set minimal required state so _ensure_adapter can run
        bridge._cwd_path = "/tmp/test"
        bridge._allowed_tools = ["Read", "Write"]
        bridge._mcp_servers = {}
        bridge._system_prompt = {"type": "text", "text": "base prompt"}
        bridge._add_dirs = []
        bridge._configured_model = "claude-sonnet-4-5"
        return bridge

    @patch("ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter")
    def test_no_sdk_options(self, mock_adapter_cls):
        """When SDK_OPTIONS is not set, adapter uses defaults."""
        mock_adapter_cls.return_value = MagicMock()
        bridge = self._make_bridge()

        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop("SDK_OPTIONS", None)
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args
        options = call_kwargs.kwargs.get("options", call_kwargs[1].get("options", {}))
        assert options["permission_mode"] == "acceptEdits"
        assert options["allowed_tools"] == ["Read", "Write"]

    @patch("ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter")
    def test_sdk_options_override(self, mock_adapter_cls):
        """SDK_OPTIONS values override defaults."""
        mock_adapter_cls.return_value = MagicMock()
        bridge = self._make_bridge()

        sdk_opts = {
            "permission_mode": "bypassPermissions",
            "max_turns": 50,
            "max_budget_usd": 5.0,
            "temperature": 0.3,
        }

        with patch.dict(os.environ, {"SDK_OPTIONS": json.dumps(sdk_opts)}):
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args
        options = call_kwargs.kwargs.get("options", call_kwargs[1].get("options", {}))
        assert options["permission_mode"] == "bypassPermissions"
        assert options["max_turns"] == 50
        assert options["max_budget_usd"] == 5.0
        assert options["temperature"] == 0.3

    @patch("ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter")
    def test_sdk_options_system_prompt_appended(self, mock_adapter_cls):
        """Custom system_prompt is appended, not replaced."""
        mock_adapter_cls.return_value = MagicMock()
        bridge = self._make_bridge()

        sdk_opts = {"system_prompt": "Always respond in French."}

        with patch.dict(os.environ, {"SDK_OPTIONS": json.dumps(sdk_opts)}):
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args
        options = call_kwargs.kwargs.get("options", call_kwargs[1].get("options", {}))
        prompt = options["system_prompt"]
        assert "base prompt" in prompt["text"]
        assert "Always respond in French." in prompt["text"]
        assert "Custom Instructions" in prompt["text"]

    @patch("ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter")
    def test_sdk_options_invalid_json(self, mock_adapter_cls):
        """Invalid JSON in SDK_OPTIONS is silently ignored."""
        mock_adapter_cls.return_value = MagicMock()
        bridge = self._make_bridge()

        with patch.dict(os.environ, {"SDK_OPTIONS": "not-valid-json"}):
            bridge._ensure_adapter()

        # Should still create the adapter with defaults
        assert mock_adapter_cls.called

    @patch("ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter")
    def test_sdk_options_denylist_filtered(self, mock_adapter_cls):
        """Denied keys in SDK_OPTIONS (e.g. cwd, resume) are filtered out."""
        mock_adapter_cls.return_value = MagicMock()
        bridge = self._make_bridge()

        sdk_opts = {
            "cwd": "/tmp/malicious",
            "resume": "some-session-id",
            "mcp_servers": {"evil": {}},
            "max_turns": 25,
        }

        with patch.dict(os.environ, {"SDK_OPTIONS": json.dumps(sdk_opts)}):
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args
        options = call_kwargs.kwargs.get("options", call_kwargs[1].get("options", {}))
        # Denied keys should not appear in options (overriding platform defaults)
        assert options.get("cwd") != "/tmp/malicious"
        assert "resume" not in options
        assert options.get("mcp_servers") == {}  # Should keep bridge default, not SDK_OPTIONS value
        # Allowed keys should pass through
        assert options["max_turns"] == 25

    @patch("ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter")
    def test_sdk_options_null_values_ignored(self, mock_adapter_cls):
        """None/null values in SDK_OPTIONS don't overwrite defaults."""
        mock_adapter_cls.return_value = MagicMock()
        bridge = self._make_bridge()

        sdk_opts = {"permission_mode": None, "max_turns": 10}

        with patch.dict(os.environ, {"SDK_OPTIONS": json.dumps(sdk_opts)}):
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args
        options = call_kwargs.kwargs.get("options", call_kwargs[1].get("options", {}))
        # permission_mode should remain default since SDK_OPTIONS had null
        assert options["permission_mode"] == "acceptEdits"
        assert options["max_turns"] == 10
