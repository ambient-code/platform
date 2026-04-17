"""Tests for SDK_OPTIONS env var parsing in ClaudeBridge._ensure_adapter."""

import json
import logging
import os
from typing import Any
from unittest.mock import patch

import pytest

from ambient_runner.bridges.claude.bridge import (
    ClaudeBridge,
    _SDK_OPTIONS_DENYLIST,
    _parse_sdk_options,
)


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------

ENV_KEY = "SDK_OPTIONS"


def _make_bridge(**overrides: Any) -> ClaudeBridge:
    """Create a ClaudeBridge with minimal state so _ensure_adapter() can run."""
    bridge = ClaudeBridge()
    bridge._cwd_path = overrides.get("cwd_path", "/workspace")
    bridge._allowed_tools = overrides.get("allowed_tools", [])
    bridge._mcp_servers = overrides.get("mcp_servers", {})
    bridge._system_prompt = overrides.get(
        "system_prompt", {"type": "preset", "preset": "claude_code", "append": "base"}
    )
    bridge._add_dirs = overrides.get("add_dirs", [])
    bridge._configured_model = overrides.get("configured_model", "")
    return bridge


# ------------------------------------------------------------------
# _parse_sdk_options unit tests
# ------------------------------------------------------------------


class TestParseSdkOptionsValidJson:
    """Valid JSON from SDK_OPTIONS env var is parsed into a dict."""

    def test_valid_json_returns_dict(self):
        raw = json.dumps({"max_tokens": 4096, "temperature": 0.5})
        result = _parse_sdk_options(raw)
        assert result == {"max_tokens": 4096, "temperature": 0.5}

    def test_empty_string_returns_empty_dict(self):
        assert _parse_sdk_options("") == {}

    def test_whitespace_only_returns_empty_dict(self):
        assert _parse_sdk_options("   ") == {}


class TestParseSdkOptionsMalformedJson:
    """Malformed JSON (not valid JSON) logs a warning and returns empty dict."""

    def test_malformed_json_returns_empty(self, caplog):
        with caplog.at_level(logging.WARNING):
            result = _parse_sdk_options("{not valid json")
        assert result == {}
        assert any(
            "SDK_OPTIONS" in r.message and "invalid JSON" in r.message
            for r in caplog.records
        )

    def test_trailing_comma_returns_empty(self, caplog):
        with caplog.at_level(logging.WARNING):
            result = _parse_sdk_options('{"a": 1,}')
        assert result == {}


class TestParseSdkOptionsJsonArray:
    """JSON array (valid JSON but not object) logs a warning and returns empty dict."""

    def test_json_array_returns_empty(self, caplog):
        with caplog.at_level(logging.WARNING):
            result = _parse_sdk_options("[1, 2, 3]")
        assert result == {}
        assert any("must be a JSON object" in r.message for r in caplog.records)

    def test_json_string_returns_empty(self, caplog):
        with caplog.at_level(logging.WARNING):
            result = _parse_sdk_options('"just a string"')
        assert result == {}


class TestParseSdkOptionsDenylist:
    """Denylisted keys are blocked with per-key warning."""

    @pytest.mark.parametrize("key", sorted(_SDK_OPTIONS_DENYLIST))
    def test_denylisted_key_blocked(self, key, caplog):
        raw = json.dumps({key: "some_value"})
        with caplog.at_level(logging.WARNING):
            result = _parse_sdk_options(raw)
        assert key not in result
        assert any(key in r.message and "denied" in r.message for r in caplog.records)

    def test_all_denylisted_keys_present(self):
        """Verify the denylist contains expected keys."""
        expected = {
            "cwd",
            "api_key",
            "mcp_servers",
            "setting_sources",
            "stderr",
            "resume",
            "continue_conversation",
            "add_dirs",
            "cli_path",
            "env",
        }
        assert _SDK_OPTIONS_DENYLIST == expected

    def test_mixed_allowed_and_denied(self, caplog):
        raw = json.dumps({"temperature": 0.5, "cwd": "/bad", "max_tokens": 100})
        with caplog.at_level(logging.WARNING):
            result = _parse_sdk_options(raw)
        assert result == {"temperature": 0.5, "max_tokens": 100}
        assert "cwd" not in result


class TestParseSdkOptionsPassthrough:
    """Non-denylisted keys pass through."""

    def test_allowed_keys_pass_through(self):
        raw = json.dumps(
            {
                "temperature": 0.7,
                "max_tokens": 8192,
                "model": "claude-sonnet-4-20250514",
            }
        )
        result = _parse_sdk_options(raw)
        assert result == {
            "temperature": 0.7,
            "max_tokens": 8192,
            "model": "claude-sonnet-4-20250514",
        }

    def test_none_value_excluded(self):
        raw = json.dumps({"temperature": None})
        result = _parse_sdk_options(raw)
        assert "temperature" not in result

    def test_count_logged_at_info(self, caplog):
        raw = json.dumps({"temperature": 0.5, "max_tokens": 100})
        with caplog.at_level(logging.INFO):
            _parse_sdk_options(raw)
        assert any("2 SDK option(s)" in r.message for r in caplog.records)


class TestParseSdkOptionsSystemPromptString:
    """system_prompt string value is appended to platform prompt under Custom Instructions heading."""

    def test_system_prompt_appended_to_string_prompt(self):
        raw = json.dumps({"system_prompt": "Always respond in French"})
        result = _parse_sdk_options(raw, existing_system_prompt="You are helpful.")
        assert result["system_prompt"] == (
            "You are helpful.\n\n## Custom Instructions\nAlways respond in French"
        )

    def test_system_prompt_appended_to_dict_prompt_with_append_field(self):
        existing = {"type": "preset", "preset": "claude_code", "append": "base prompt"}
        raw = json.dumps({"system_prompt": "Use Python 3.12"})
        result = _parse_sdk_options(raw, existing_system_prompt=existing)
        expected = dict(existing)
        expected["append"] = "base prompt\n\n## Custom Instructions\nUse Python 3.12"
        assert result["system_prompt"] == expected

    def test_system_prompt_dict_with_text_field(self):
        existing = {"text": "You are a code reviewer."}
        raw = json.dumps({"system_prompt": "Focus on security"})
        result = _parse_sdk_options(raw, existing_system_prompt=existing)
        expected = dict(existing)
        expected["text"] = (
            "You are a code reviewer.\n\n## Custom Instructions\nFocus on security"
        )
        assert result["system_prompt"] == expected


class TestParseSdkOptionsSystemPromptIgnored:
    """system_prompt as None/empty is ignored (platform prompt unchanged)."""

    def test_system_prompt_none_ignored(self):
        raw = json.dumps({"system_prompt": None})
        result = _parse_sdk_options(raw, existing_system_prompt="base")
        assert "system_prompt" not in result

    def test_system_prompt_empty_string_ignored(self):
        raw = json.dumps({"system_prompt": ""})
        result = _parse_sdk_options(raw, existing_system_prompt="base")
        assert "system_prompt" not in result

    def test_system_prompt_whitespace_ignored(self):
        raw = json.dumps({"system_prompt": "   "})
        result = _parse_sdk_options(raw, existing_system_prompt="base")
        assert "system_prompt" not in result


# ------------------------------------------------------------------
# Integration: _ensure_adapter applies SDK_OPTIONS
# ------------------------------------------------------------------


class TestEnsureAdapterSdkOptions:
    """Verify _ensure_adapter integrates _parse_sdk_options into the adapter options."""

    def test_sdk_options_applied_to_adapter(self):
        bridge = _make_bridge()
        env = {ENV_KEY: json.dumps({"temperature": 0.3, "max_tokens": 2048})}

        with (
            patch.dict(os.environ, env, clear=False),
            patch(
                "ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter"
            ) as mock_adapter_cls,
        ):
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args[1]
        opts = call_kwargs["options"]
        assert opts["temperature"] == 0.3
        assert opts["max_tokens"] == 2048

    def test_sdk_options_denylisted_key_not_in_adapter(self):
        bridge = _make_bridge()
        env = {ENV_KEY: json.dumps({"cwd": "/evil", "temperature": 0.5})}

        with (
            patch.dict(os.environ, env, clear=False),
            patch(
                "ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter"
            ) as mock_adapter_cls,
        ):
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args[1]
        opts = call_kwargs["options"]
        # cwd should remain the bridge's own value, not overridden
        assert opts["cwd"] == "/workspace"
        assert opts["temperature"] == 0.5

    def test_sdk_options_system_prompt_merged(self):
        bridge = _make_bridge(
            system_prompt={
                "type": "preset",
                "preset": "claude_code",
                "append": "platform base",
            }
        )
        env = {ENV_KEY: json.dumps({"system_prompt": "Be concise"})}

        with (
            patch.dict(os.environ, env, clear=False),
            patch(
                "ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter"
            ) as mock_adapter_cls,
        ):
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args[1]
        opts = call_kwargs["options"]
        assert "## Custom Instructions" in opts["system_prompt"]["append"]
        assert "Be concise" in opts["system_prompt"]["append"]

    def test_no_sdk_options_env_var(self):
        bridge = _make_bridge()
        env = {}

        with (
            patch.dict(os.environ, env, clear=False),
            patch(
                "ambient_runner.bridges.claude.bridge.ClaudeAgentAdapter"
            ) as mock_adapter_cls,
        ):
            # Ensure SDK_OPTIONS is not set
            os.environ.pop(ENV_KEY, None)
            bridge._ensure_adapter()

        call_kwargs = mock_adapter_cls.call_args[1]
        opts = call_kwargs["options"]
        # Should have base options only
        assert opts["cwd"] == "/workspace"
        assert "temperature" not in opts
