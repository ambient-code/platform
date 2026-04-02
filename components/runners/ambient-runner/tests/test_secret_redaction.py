"""Tests for the secret redaction middleware."""

import os
from unittest.mock import patch

import pytest
from ag_ui.core import (
    CustomEvent,
    EventType,
    RunErrorEvent,
    TextMessageChunkEvent,
    TextMessageContentEvent,
    ToolCallArgsEvent,
    ToolCallChunkEvent,
    ToolCallEndEvent,
    ToolCallResultEvent,
    ToolCallStartEvent,
)

from ambient_runner.middleware.secret_redaction import (
    _collect_secret_values,
    _redact_dict,
    _redact_event,
    _redact_text,
    _redact_value,
    secret_redaction_middleware,
)


def _fake_github_pat(length: int = 36) -> str:
    """Build a GitHub-PAT-shaped token at runtime to avoid secret-scanner flags."""
    return "ghp_" + "a" * length


def _fake_anthropic_key(length: int = 36) -> str:
    """Build an Anthropic-key-shaped token at runtime."""
    return "sk-" + "ant-" + "a" * length


# -- Unit tests for _collect_secret_values --


class TestCollectSecretValues:
    def test_collects_from_env(self):
        with patch.dict(
            os.environ,
            {"GITHUB_TOKEN": _fake_github_pat()},
            clear=False,
        ):
            pairs = _collect_secret_values()
            assert any(var == "GITHUB_TOKEN" for var, _ in pairs)

    def test_skips_short_values(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "short"}, clear=False):
            pairs = _collect_secret_values()
            assert not any(var == "GITHUB_TOKEN" for var, _ in pairs)

    def test_skips_empty_values(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": ""}, clear=False):
            pairs = _collect_secret_values()
            assert not any(var == "GITHUB_TOKEN" for var, _ in pairs)

    def test_sorted_longest_first(self):
        with patch.dict(
            os.environ,
            {
                "GITHUB_TOKEN": "a" * 20,
                "GITLAB_TOKEN": "b" * 40,
            },
            clear=False,
        ):
            pairs = _collect_secret_values()
            gh_idx = next(i for i, (v, _) in enumerate(pairs) if v == "GITHUB_TOKEN")
            gl_idx = next(i for i, (v, _) in enumerate(pairs) if v == "GITLAB_TOKEN")
            assert gl_idx < gh_idx  # longer value comes first


# -- Unit tests for _redact_text --


class TestRedactText:
    def test_value_based_redaction(self):
        secret = "my-secret-api-key-value"
        secrets = [("ANTHROPIC_API_KEY", secret)]
        text = f"Error connecting with key {secret} to API"
        result = _redact_text(text, secrets)
        assert secret not in result
        assert "[REDACTED_ANTHROPIC_API_KEY]" in result

    def test_pattern_based_github_pat(self):
        pat = _fake_github_pat()
        text = f"Found token {pat} in config"
        result = _redact_text(text, [])
        assert "ghp_" not in result or "REDACTED" in result

    def test_pattern_based_anthropic_key(self):
        key = _fake_anthropic_key()
        text = f"Key: {key}"
        result = _redact_text(text, [])
        assert "sk-ant-" not in result or "REDACTED" in result

    def test_no_change_for_clean_text(self):
        text = "Hello, this is a normal message"
        result = _redact_text(text, [])
        assert result == text

    def test_both_approaches_combined(self):
        secret = "my-custom-secret-value"
        secrets = [("BOT_TOKEN", secret)]
        pat = _fake_github_pat()
        text = f"Token {secret} and also {pat}"
        result = _redact_text(text, secrets)
        assert secret not in result
        assert pat not in result


# -- Unit tests for _redact_event --


class TestRedactEvent:
    SECRETS = [("TEST_SECRET", "supersecretvalue123")]

    def test_text_message_content_event(self):
        event = TextMessageContentEvent(
            type=EventType.TEXT_MESSAGE_CONTENT,
            message_id="msg-1",
            delta="The key is supersecretvalue123 here",
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.delta
        assert "[REDACTED_TEST_SECRET]" in result.delta

    def test_text_message_chunk_event(self):
        event = TextMessageChunkEvent(
            type=EventType.TEXT_MESSAGE_CHUNK,
            message_id="msg-1",
            delta="Token: supersecretvalue123",
            role="assistant",
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.delta

    def test_tool_call_args_event(self):
        event = ToolCallArgsEvent(
            type=EventType.TOOL_CALL_ARGS,
            tool_call_id="tc-1",
            delta='{"key": "supersecretvalue123"}',
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.delta

    def test_tool_call_chunk_event(self):
        event = ToolCallChunkEvent(
            type=EventType.TOOL_CALL_CHUNK,
            tool_call_id="tc-1",
            tool_call_name="read_file",
            parent_message_id="msg-1",
            delta="supersecretvalue123",
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.delta

    def test_tool_call_result_event(self):
        event = ToolCallResultEvent(
            type=EventType.TOOL_CALL_RESULT,
            message_id="msg-1",
            tool_call_id="tc-1",
            role="tool",
            content="File contains supersecretvalue123",
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.content

    def test_run_error_event(self):
        event = RunErrorEvent(
            type=EventType.RUN_ERROR,
            message="Auth failed with key supersecretvalue123",
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.message

    def test_custom_event_string_value(self):
        event = CustomEvent(
            type=EventType.CUSTOM,
            name="test",
            value="secret is supersecretvalue123",
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.value

    def test_custom_event_dict_value(self):
        event = CustomEvent(
            type=EventType.CUSTOM,
            name="test",
            value={"nested": {"key": "value supersecretvalue123"}},
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.value["nested"]["key"]

    def test_custom_event_list_value(self):
        event = CustomEvent(
            type=EventType.CUSTOM,
            name="test",
            value=["clean", "supersecretvalue123", {"k": "supersecretvalue123"}],
        )
        result = _redact_event(event, self.SECRETS)
        assert "supersecretvalue123" not in result.value[1]
        assert "supersecretvalue123" not in result.value[2]["k"]

    def test_passthrough_non_text_events(self):
        event = ToolCallStartEvent(
            type=EventType.TOOL_CALL_START,
            tool_call_id="tc-1",
            tool_call_name="read_file",
            parent_message_id="msg-1",
        )
        result = _redact_event(event, self.SECRETS)
        assert result is event  # exact same object, not a copy

    def test_passthrough_clean_text(self):
        event = TextMessageContentEvent(
            type=EventType.TEXT_MESSAGE_CONTENT,
            message_id="msg-1",
            delta="No secrets here",
        )
        result = _redact_event(event, self.SECRETS)
        assert result is event  # unchanged, returns original

    def test_tool_call_end_passthrough(self):
        event = ToolCallEndEvent(
            type=EventType.TOOL_CALL_END,
            tool_call_id="tc-1",
        )
        result = _redact_event(event, self.SECRETS)
        assert result is event


# -- Unit tests for _redact_dict --


class TestRedactDict:
    SECRETS = [("MY_KEY", "secret_value_here")]

    def test_redacts_string_values(self):
        d = {"key": "contains secret_value_here in text"}
        result = _redact_dict(d, self.SECRETS)
        assert "secret_value_here" not in result["key"]

    def test_recursive_redaction(self):
        d = {"outer": {"inner": "secret_value_here"}}
        result = _redact_dict(d, self.SECRETS)
        assert "secret_value_here" not in result["outer"]["inner"]

    def test_returns_original_if_unchanged(self):
        d = {"key": "clean value", "count": 42}
        result = _redact_dict(d, self.SECRETS)
        assert result is d

    def test_preserves_non_string_values(self):
        d = {"count": 42, "flag": True, "items": [1, 2, 3]}
        result = _redact_dict(d, self.SECRETS)
        assert result is d

    def test_redacts_list_values_in_dict(self):
        d = {"items": ["clean", "has secret_value_here", "also clean"]}
        result = _redact_dict(d, self.SECRETS)
        assert "secret_value_here" not in result["items"][1]

    def test_redacts_nested_dict_in_list(self):
        d = {"items": [{"key": "secret_value_here"}]}
        result = _redact_dict(d, self.SECRETS)
        assert "secret_value_here" not in result["items"][0]["key"]

    def test_redacts_dict_keys(self):
        d = {"secret_value_here": "clean_value"}
        result = _redact_dict(d, self.SECRETS)
        assert "secret_value_here" not in result
        assert any("[REDACTED_MY_KEY]" in str(k) for k in result)


# -- Unit tests for _redact_value --


class TestRedactValue:
    SECRETS = [("MY_KEY", "secret_value_here")]

    def test_redacts_string(self):
        result = _redact_value("contains secret_value_here", self.SECRETS)
        assert "secret_value_here" not in result

    def test_redacts_list(self):
        val = ["clean", "secret_value_here", "also clean"]
        result = _redact_value(val, self.SECRETS)
        assert "secret_value_here" not in result[1]
        assert result[0] == "clean"
        assert result[2] == "also clean"

    def test_list_passthrough_if_unchanged(self):
        val = ["clean", "no secrets"]
        result = _redact_value(val, self.SECRETS)
        assert result is val

    def test_passthrough_non_redactable(self):
        assert _redact_value(42, self.SECRETS) == 42
        assert _redact_value(True, self.SECRETS) is True
        assert _redact_value(None, self.SECRETS) is None

    def test_nested_list_in_list(self):
        val = [["secret_value_here"]]
        result = _redact_value(val, self.SECRETS)
        assert "secret_value_here" not in result[0][0]


# -- Integration test for the full middleware --


class TestSecretRedactionMiddleware:
    @pytest.mark.asyncio
    async def test_full_middleware_pipeline(self):
        """End-to-end test: secrets in env are redacted from events."""
        secret_token = _fake_github_pat()

        events = [
            TextMessageContentEvent(
                type=EventType.TEXT_MESSAGE_CONTENT,
                message_id="msg-1",
                delta=f"I found a token: {secret_token}",
            ),
            ToolCallStartEvent(
                type=EventType.TOOL_CALL_START,
                tool_call_id="tc-1",
                tool_call_name="read",
                parent_message_id="msg-1",
            ),
            ToolCallResultEvent(
                type=EventType.TOOL_CALL_RESULT,
                message_id="msg-2",
                tool_call_id="tc-1",
                role="tool",
                content=f"GITHUB_TOKEN={secret_token}",
            ),
        ]

        async def fake_stream():
            for e in events:
                yield e

        with patch.dict(os.environ, {"GITHUB_TOKEN": secret_token}, clear=False):
            results = []
            async for event in secret_redaction_middleware(fake_stream()):
                results.append(event)

        assert len(results) == 3

        # TextMessageContentEvent should be redacted
        assert secret_token not in results[0].delta

        # ToolCallStartEvent passes through unchanged
        assert results[1].tool_call_name == "read"

        # ToolCallResultEvent should be redacted
        assert secret_token not in results[2].content

    @pytest.mark.asyncio
    async def test_empty_stream(self):
        """Middleware handles empty streams gracefully."""

        async def empty():
            return
            yield  # make it an async generator

        results = []
        async for event in secret_redaction_middleware(empty()):
            results.append(event)
        assert results == []

    @pytest.mark.asyncio
    async def test_no_secrets_in_env(self):
        """When no secrets are set, pattern-based redaction still works."""
        token = _fake_github_pat()
        events = [
            TextMessageContentEvent(
                type=EventType.TEXT_MESSAGE_CONTENT,
                message_id="msg-1",
                delta=f"token: {token}",
            ),
        ]

        async def fake_stream():
            for e in events:
                yield e

        # Clear all secret env vars
        clear_env = {
            var: ""
            for var in (
                "ANTHROPIC_API_KEY",
                "BOT_TOKEN",
                "GITHUB_TOKEN",
                "GITLAB_TOKEN",
                "JIRA_API_TOKEN",
                "GEMINI_API_KEY",
                "GOOGLE_API_KEY",
                "GOOGLE_OAUTH_CLIENT_SECRET",
                "LANGFUSE_SECRET_KEY",
                "LANGFUSE_PUBLIC_KEY",
                "LANGSMITH_API_KEY",
            )
        }
        with patch.dict(os.environ, clear_env, clear=False):
            results = []
            async for event in secret_redaction_middleware(fake_stream()):
                results.append(event)

        assert token not in results[0].delta
