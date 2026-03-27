"""
MockClaudeSDKClient — drop-in replacement for ClaudeSDKClient in tests.

Activated when ANTHROPIC_API_KEY=mock-replay-key. Replays pre-recorded
SDK messages from JSONL fixture files, allowing the full ClaudeBridge
stack (SessionWorker, ClaudeAgentAdapter, etc.) to run without a real
Anthropic API connection.

Fixtures are captured using scripts/capture-fixtures.py with a real key,
then committed to ambient_runner/bridges/claude/fixtures/.
"""

import asyncio
import json
import logging
import re
from pathlib import Path
from typing import Any, AsyncIterator, Optional

from claude_agent_sdk import (
    AssistantMessage,
    ResultMessage,
    SystemMessage,
    UserMessage,
    ThinkingBlock,
    ToolResultBlock,
    ToolUseBlock,
    TextBlock,
)
from claude_agent_sdk.types import StreamEvent

logger = logging.getLogger(__name__)

MOCK_API_KEY = "mock-replay-key"

_FIXTURES_DIR = Path(__file__).parent / "fixtures"

_SCENARIO_PATTERNS: list[tuple[re.Pattern, str]] = [
    (re.compile(r"\bhello\b", re.IGNORECASE), "hello"),
    (re.compile(r"\bhi\b", re.IGNORECASE), "hello"),
    (re.compile(r"\bhey\b", re.IGNORECASE), "hello"),
    (re.compile(r"\bfix a bug\b", re.IGNORECASE), "workflow"),
    (re.compile(r"\bworkflow\b", re.IGNORECASE), "workflow"),
    (re.compile(r"\banalyze\b", re.IGNORECASE), "workflow"),
    (re.compile(r"\bcomprehensive\b", re.IGNORECASE), "comprehensive"),
]

_SDK_TYPE_MAP: dict[str, type] = {
    "StreamEvent": StreamEvent,
    "AssistantMessage": AssistantMessage,
    "UserMessage": UserMessage,
    "SystemMessage": SystemMessage,
    "ResultMessage": ResultMessage,
}

_BLOCK_TYPE_MAP: dict[str, type] = {
    "TextBlock": TextBlock,
    "ToolUseBlock": ToolUseBlock,
    "ThinkingBlock": ThinkingBlock,
    "ToolResultBlock": ToolResultBlock,
}


class MockClaudeSDKClient:
    """Mimics ClaudeSDKClient but replays fixture messages instead of calling Claude."""

    def __init__(self, options: Any = None) -> None:
        self._prompt = ""
        self._interrupted = False

    async def connect(self) -> None:
        logger.info("[MockSDK] connect() — no-op")

    async def query(self, prompt: str, session_id: Optional[str] = None) -> None:
        logger.info("[MockSDK] query: %r", prompt[:80])
        self._prompt = prompt
        self._interrupted = False

    async def receive_response(self) -> AsyncIterator[Any]:
        scenario = _match_scenario(self._prompt)
        fixture_path = _FIXTURES_DIR / f"{scenario}.jsonl"

        if not fixture_path.exists():
            fixture_path = _FIXTURES_DIR / "default.jsonl"

        if not fixture_path.exists():
            logger.error("[MockSDK] No fixture found in %s", _FIXTURES_DIR)
            yield ResultMessage(
                subtype="result",
                duration_ms=0,
                duration_api_ms=0,
                is_error=True,
                num_turns=0,
                session_id="mock",
            )
            return

        logger.info("[MockSDK] Replaying fixture: %s", fixture_path.name)

        for msg in _load_fixture(fixture_path):
            if self._interrupted:
                return
            yield msg

    async def receive_messages(self) -> AsyncIterator[Any]:
        """Persistent message stream — replays fixture then blocks forever."""
        async for msg in self.receive_response():
            yield msg
        # Simulate persistent stream: block until cancelled
        await asyncio.Event().wait()

    async def interrupt(self) -> None:
        logger.info("[MockSDK] interrupt()")
        self._interrupted = True


# ------------------------------------------------------------------
# Fixture loading
# ------------------------------------------------------------------


def _load_fixture(path: Path) -> list[Any]:
    messages = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            data = json.loads(line)
            msg = _deserialize(data)
            if msg is not None:
                messages.append(msg)
    return messages


def _deserialize(data: dict) -> Optional[Any]:
    type_name = data.pop("_type", "")
    data.pop("delay_ms", None)

    cls = _SDK_TYPE_MAP.get(type_name)
    if cls is None:
        logger.warning("[MockSDK] Unknown type in fixture: %s", type_name)
        return None

    if cls in (AssistantMessage, UserMessage) and "content" in data:
        content = data["content"]
        if isinstance(content, list):
            data["content"] = [b for b in (_deserialize_block(b) for b in content) if b]

    try:
        return cls(**data)
    except Exception as e:
        logger.warning("[MockSDK] Failed to construct %s: %s", type_name, e)
        return None


def _deserialize_block(data: dict) -> Optional[Any]:
    type_name = data.pop("_type", "")
    cls = _BLOCK_TYPE_MAP.get(type_name)
    if cls is None:
        return None
    try:
        return cls(**data)
    except Exception as e:
        logger.warning("[MockSDK] Failed to construct block %s: %s", type_name, e)
        return None


def _match_scenario(prompt: str) -> str:
    for pattern, scenario in _SCENARIO_PATTERNS:
        if pattern.search(prompt):
            return scenario
    return "default"
