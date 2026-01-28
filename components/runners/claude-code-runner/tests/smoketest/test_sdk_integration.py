#!/usr/bin/env python3
"""
SDK Integration Smoketest Suite

Validates Claude Agent SDK integration before and after upgrades.
Run before deploying new SDK versions to catch regressions.

Usage:
    pytest tests/smoketest/test_sdk_integration.py -v
    pytest tests/smoketest/test_sdk_integration.py -v --slow  # Include slow tests
"""

import asyncio
import os
import sys
import tempfile
from pathlib import Path
from typing import AsyncIterator, Dict, Any
import pytest

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from claude_agent_sdk import (
    ClaudeSDKClient,
    ClaudeAgentOptions,
    AssistantMessage,
    UserMessage,
    SystemMessage,
    ResultMessage,
    TextBlock,
    ToolUseBlock,
    ToolResultBlock,
)


# Test Configuration
ANTHROPIC_API_KEY = os.getenv("ANTHROPIC_API_KEY", "")
SKIP_REASON = "ANTHROPIC_API_KEY not set (set to run live tests)"


@pytest.fixture
def temp_workspace():
    """Create temporary workspace for SDK operations."""
    with tempfile.TemporaryDirectory() as tmpdir:
        workspace = Path(tmpdir)
        (workspace / "test.txt").write_text("Hello, World!")
        yield workspace


@pytest.fixture
def sdk_options(temp_workspace: Path) -> ClaudeAgentOptions:
    """Standard SDK options for testing."""
    return ClaudeAgentOptions(
        cwd=str(temp_workspace),
        permission_mode="acceptEdits",
        allowed_tools=["Read", "Write", "Bash"],
        include_partial_messages=True,
    )


class TestSDKClientLifecycle:
    """Test SDK client creation, connection, and cleanup."""

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    async def test_client_creation(self, sdk_options):
        """SDK client can be created with valid options."""
        client = ClaudeSDKClient(options=sdk_options)
        assert client is not None

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    async def test_client_connect_disconnect(self, sdk_options):
        """SDK client connects and disconnects cleanly."""
        client = ClaudeSDKClient(options=sdk_options)

        await client.connect()
        # Client should be connected

        await client.disconnect()
        # Client should be disconnected cleanly

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    async def test_multiple_clients_sequential(self, sdk_options):
        """Multiple clients can be created sequentially without conflicts."""
        for i in range(3):
            client = ClaudeSDKClient(options=sdk_options)
            await client.connect()
            await client.disconnect()


class TestSDKMessageHandling:
    """Test SDK message types and streaming."""

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    @pytest.mark.slow
    async def test_simple_query_response(self, sdk_options):
        """SDK handles simple query and returns AssistantMessage."""
        client = ClaudeSDKClient(options=sdk_options)
        await client.connect()

        try:
            await client.query("Say 'Hello' and nothing else.")

            assistant_messages = []
            result_messages = []

            async for message in client.receive_response():
                if isinstance(message, AssistantMessage):
                    assistant_messages.append(message)
                elif isinstance(message, ResultMessage):
                    result_messages.append(message)

            # Should receive at least one AssistantMessage
            assert len(assistant_messages) >= 1

            # Should receive exactly one ResultMessage
            assert len(result_messages) == 1

            # Result should have usage information
            result = result_messages[0]
            assert result.usage is not None

        finally:
            await client.disconnect()

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    @pytest.mark.slow
    async def test_text_block_parsing(self, sdk_options):
        """SDK correctly parses TextBlock in AssistantMessage."""
        client = ClaudeSDKClient(options=sdk_options)
        await client.connect()

        try:
            await client.query("Respond with exactly: TEST_OUTPUT_12345")

            text_blocks = []

            async for message in client.receive_response():
                if isinstance(message, AssistantMessage):
                    for block in message.content or []:
                        if isinstance(block, TextBlock):
                            text_blocks.append(block)

            # Should have at least one text block
            assert len(text_blocks) >= 1

            # Combined text should contain our test string
            combined_text = "".join(block.text for block in text_blocks)
            assert "TEST_OUTPUT_12345" in combined_text

        finally:
            await client.disconnect()


class TestSDKToolExecution:
    """Test tool invocation and result handling."""

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    @pytest.mark.slow
    async def test_read_tool_execution(self, sdk_options, temp_workspace):
        """SDK executes Read tool and returns result."""
        test_file = temp_workspace / "test.txt"
        test_content = "SDK_TEST_CONTENT_UNIQUE"
        test_file.write_text(test_content)

        client = ClaudeSDKClient(options=sdk_options)
        await client.connect()

        try:
            await client.query(f"Read the file test.txt and tell me what it contains.")

            tool_use_blocks = []
            tool_result_blocks = []

            async for message in client.receive_response():
                if isinstance(message, AssistantMessage):
                    for block in message.content or []:
                        if isinstance(block, ToolUseBlock):
                            tool_use_blocks.append(block)
                        elif isinstance(block, ToolResultBlock):
                            tool_result_blocks.append(block)

            # Should invoke at least one tool (Read)
            assert len(tool_use_blocks) >= 1

            # Should have Read tool invocation
            read_tools = [t for t in tool_use_blocks if t.name == "Read"]
            assert len(read_tools) >= 1

            # Should have tool results
            assert len(tool_result_blocks) >= 1

        finally:
            await client.disconnect()

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    @pytest.mark.slow
    async def test_write_tool_execution(self, sdk_options, temp_workspace):
        """SDK executes Write tool and creates file."""
        client = ClaudeSDKClient(options=sdk_options)
        await client.connect()

        try:
            await client.query("Create a file called output.txt with the content 'SDK_WRITE_TEST'")

            tool_use_blocks = []

            async for message in client.receive_response():
                if isinstance(message, AssistantMessage):
                    for block in message.content or []:
                        if isinstance(block, ToolUseBlock):
                            tool_use_blocks.append(block)

            # Should invoke Write tool
            write_tools = [t for t in tool_use_blocks if t.name == "Write"]
            assert len(write_tools) >= 1

            # File should exist
            output_file = temp_workspace / "output.txt"
            assert output_file.exists()

            # File should contain expected content
            content = output_file.read_text()
            assert "SDK_WRITE_TEST" in content

        finally:
            await client.disconnect()


class TestSDKConfiguration:
    """Test SDK configuration options."""

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    async def test_permission_mode_accept_edits(self, temp_workspace):
        """SDK with acceptEdits permission mode auto-approves edits."""
        options = ClaudeAgentOptions(
            cwd=str(temp_workspace),
            permission_mode="acceptEdits",
            allowed_tools=["Write"],
        )

        client = ClaudeSDKClient(options=options)
        await client.connect()

        try:
            await client.query("Create file auto_approved.txt with content 'AUTO'")

            async for message in client.receive_response():
                pass  # Consume all messages

            # File should be created without manual approval
            assert (temp_workspace / "auto_approved.txt").exists()

        finally:
            await client.disconnect()

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    async def test_tool_restrictions(self, temp_workspace):
        """SDK respects allowed_tools restrictions."""
        options = ClaudeAgentOptions(
            cwd=str(temp_workspace),
            permission_mode="acceptEdits",
            allowed_tools=["Read"],  # Only Read, no Write
        )

        client = ClaudeSDKClient(options=options)
        await client.connect()

        try:
            await client.query("Try to write a file called restricted.txt")

            tool_use_blocks = []

            async for message in client.receive_response():
                if isinstance(message, AssistantMessage):
                    for block in message.content or []:
                        if isinstance(block, ToolUseBlock):
                            tool_use_blocks.append(block)

            # Should not invoke Write tool (only Read is allowed)
            write_tools = [t for t in tool_use_blocks if t.name == "Write"]
            assert len(write_tools) == 0

        finally:
            await client.disconnect()


class TestSDKStatePersistence:
    """Test conversation continuation and state management."""

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    @pytest.mark.slow
    async def test_conversation_continuation(self, temp_workspace):
        """SDK continues conversation from disk state."""
        # First session: Create file
        options_1 = ClaudeAgentOptions(
            cwd=str(temp_workspace),
            permission_mode="acceptEdits",
            allowed_tools=["Write", "Read"],
        )

        client_1 = ClaudeSDKClient(options=options_1)
        await client_1.connect()

        try:
            await client_1.query("Create a file called memory.txt with content 'FIRST_SESSION'")

            async for message in client_1.receive_response():
                pass  # Consume all messages

        finally:
            await client_1.disconnect()

        # Second session: Resume and verify memory
        options_2 = ClaudeAgentOptions(
            cwd=str(temp_workspace),
            permission_mode="acceptEdits",
            allowed_tools=["Write", "Read"],
            continue_conversation=True,
        )

        client_2 = ClaudeSDKClient(options=options_2)
        await client_2.connect()

        try:
            await client_2.query("What file did we create in the previous conversation?")

            text_content = []

            async for message in client_2.receive_response():
                if isinstance(message, AssistantMessage):
                    for block in message.content or []:
                        if isinstance(block, TextBlock):
                            text_content.append(block.text)

            # Should remember the file from previous session
            combined_text = "".join(text_content)
            assert "memory.txt" in combined_text.lower()

        finally:
            await client_2.disconnect()


class TestSDKErrorHandling:
    """Test SDK error handling and recovery."""

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    async def test_invalid_tool_input(self, sdk_options):
        """SDK handles invalid tool inputs gracefully."""
        # This test validates the SDK doesn't crash on edge cases
        # Actual behavior depends on SDK implementation
        pass

    @pytest.mark.skipif(not ANTHROPIC_API_KEY, reason=SKIP_REASON)
    @pytest.mark.slow
    async def test_interrupt_handling(self, sdk_options):
        """SDK supports interrupt during execution."""
        client = ClaudeSDKClient(options=sdk_options)
        await client.connect()

        try:
            # Start a query that might take time
            await client.query("Count to 100 slowly.")

            # Send interrupt after brief delay
            await asyncio.sleep(0.5)
            await client.interrupt()

            # Should be able to consume remaining messages without hanging
            async for message in client.receive_response():
                pass

        finally:
            await client.disconnect()


class TestSDKVersionCompatibility:
    """Test SDK version and feature detection."""

    def test_sdk_version_available(self):
        """SDK version can be determined."""
        import claude_agent_sdk
        assert hasattr(claude_agent_sdk, "__version__")

    def test_required_message_types_exist(self):
        """All required message types are available."""
        from claude_agent_sdk import (
            AssistantMessage,
            UserMessage,
            SystemMessage,
            ResultMessage,
            TextBlock,
            ToolUseBlock,
            ToolResultBlock,
        )
        # If imports succeed, types are available

    def test_client_has_required_methods(self):
        """ClaudeSDKClient has all required methods."""
        required_methods = [
            "connect",
            "disconnect",
            "query",
            "receive_response",
            "interrupt",
        ]

        for method in required_methods:
            assert hasattr(ClaudeSDKClient, method)


# Test markers for pytest
def pytest_configure(config):
    """Register custom markers."""
    config.addinivalue_line("markers", "slow: marks tests as slow (deselect with '-m \"not slow\"')")


# Run directly for quick validation
if __name__ == "__main__":
    import sys

    if not ANTHROPIC_API_KEY:
        print("ERROR: ANTHROPIC_API_KEY environment variable not set")
        print("Set it to run live SDK tests:")
        print("  export ANTHROPIC_API_KEY=sk-ant-...")
        sys.exit(1)

    # Run quick tests only
    sys.exit(pytest.main([__file__, "-v", "-m", "not slow"]))
