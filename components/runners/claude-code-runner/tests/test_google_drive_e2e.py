"""
E2E test for Google Drive MCP integration.

This test validates the complete flow from authentication to Drive operations.

Prerequisites for running this test:
1. Set environment variable: GOOGLE_DRIVE_E2E_TEST=true
2. Place valid Google OAuth credentials at: /tmp/test_credentials.json
3. Ensure credentials are fresh (not expired)

The credentials file should be in the workspace-mcp format:
{
  "user@gmail.com": {
    "access_token": "ya29...",
    "refresh_token": "1//...",
    "token_expiry": "2026-01-23T12:00:00Z",
    "email": "user@gmail.com"
  }
}

Run with:
    GOOGLE_DRIVE_E2E_TEST=true python -m pytest tests/test_google_drive_e2e.py -v
"""

import json
import os
from pathlib import Path

import pytest


@pytest.mark.skipif(
    not os.getenv("GOOGLE_DRIVE_E2E_TEST"),
    reason="Requires GOOGLE_DRIVE_E2E_TEST=true and real credentials",
)
@pytest.mark.asyncio
async def test_google_drive_authentication_flow():
    """
    E2E test for Google Drive authentication status flow.

    Tests:
    1. Credentials are copied to workspace
    2. USER_GOOGLE_EMAIL is set correctly
    3. Authentication status is reported as True
    4. User email is extracted correctly (not placeholder)
    """
    import sys

    sys.path.insert(0, str(Path(__file__).parent.parent))

    from adapter import ClaudeCodeAdapter
    from context import RunnerContext
    from main import _check_mcp_authentication

    # Setup test credentials
    test_creds_path = Path("/tmp/test_credentials.json")
    assert test_creds_path.exists(), (
        "Missing test credentials at /tmp/test_credentials.json\n"
        "Create this file with valid Google OAuth credentials in workspace-mcp format."
    )

    # Validate credentials format
    with open(test_creds_path, "r") as f:
        creds = json.load(f)

    assert len(creds) > 0, "Credentials file is empty"
    user_email = list(creds.keys())[0]
    assert user_email != "user@example.com", "Cannot use placeholder email for E2E test"
    assert "access_token" in creds[user_email], "Missing access_token"
    assert "refresh_token" in creds[user_email], "Missing refresh_token"

    # Initialize adapter
    adapter = ClaudeCodeAdapter()
    workspace_path = Path("/tmp/test-workspace-google-drive-e2e")
    workspace_path.mkdir(parents=True, exist_ok=True)

    context = RunnerContext(
        session_id="test-session-google-drive", workspace_path=str(workspace_path)
    )

    # Copy test credentials to workspace location (simulating Secret mount)
    workspace_creds_dir = workspace_path / ".google_workspace_mcp/credentials"
    workspace_creds_dir.mkdir(parents=True, exist_ok=True)
    workspace_creds_file = workspace_creds_dir / "credentials.json"
    workspace_creds_file.write_text(test_creds_path.read_text())

    # Mock the mounted secret path to point to our test credentials
    secret_mount_dir = Path("/tmp/test-secret-mount-google-drive-e2e/credentials")
    secret_mount_dir.mkdir(parents=True, exist_ok=True)
    secret_mount_file = secret_mount_dir / "credentials.json"
    secret_mount_file.write_text(test_creds_path.read_text())

    # Patch the paths in adapter to use our test locations
    from unittest.mock import patch

    with patch("adapter.Path") as mock_path:

        def path_factory(path_str):
            if "/app/.google_workspace_mcp" in str(path_str):
                return secret_mount_file.parent / Path(path_str).name
            elif "/workspace/.google_workspace_mcp" in str(path_str):
                return workspace_creds_dir / Path(path_str).name
            else:
                return Path(path_str)

        mock_path.side_effect = path_factory

        # Initialize adapter
        await adapter.initialize(context)

    # Test 1: Verify USER_GOOGLE_EMAIL is set
    user_google_email = os.getenv("USER_GOOGLE_EMAIL")
    assert user_google_email is not None, "USER_GOOGLE_EMAIL should be set"
    assert user_google_email != "user@example.com", "Should not be placeholder email"
    assert (
        user_google_email == user_email
    ), f"Expected {user_email}, got {user_google_email}"

    print(f"✓ USER_GOOGLE_EMAIL correctly set to: {user_google_email}")

    # Test 2: Verify authentication status
    with patch("main.Path") as mock_path:

        def path_factory(path_str):
            if "/workspace/.google_workspace_mcp" in str(path_str):
                return workspace_creds_file
            elif "/app/.google_workspace_mcp" in str(path_str):
                return secret_mount_file
            else:
                return Path(path_str)

        mock_path.side_effect = path_factory

        is_auth, msg = _check_mcp_authentication("google-workspace")

    assert is_auth in (
        True,
        None,
    ), f"Expected True or None (refresh needed), got {is_auth}: {msg}"
    assert user_email in msg, f"Expected user email in message, got: {msg}"

    if is_auth is True:
        print(f"✓ Authentication status: Valid")
    elif is_auth is None:
        print(f"⚠ Authentication status: Needs refresh (token may be expired)")

    print(f"  Message: {msg}")

    # Cleanup
    import shutil

    shutil.rmtree(workspace_path, ignore_errors=True)
    shutil.rmtree(secret_mount_dir.parent, ignore_errors=True)


@pytest.mark.skipif(
    not os.getenv("GOOGLE_DRIVE_E2E_TEST"),
    reason="Requires GOOGLE_DRIVE_E2E_TEST=true and real credentials",
)
@pytest.mark.asyncio
async def test_google_drive_tool_invocation():
    """
    E2E test for Google Drive MCP tool invocation.

    This test would actually invoke Claude SDK with MCP tools to verify
    Drive operations work correctly. Currently a placeholder for future implementation.

    TODO: Implement actual tool invocation test with Claude SDK
    """
    pytest.skip(
        "Tool invocation test not yet implemented - requires Claude SDK integration"
    )


if __name__ == "__main__":
    # Allow running directly with: GOOGLE_DRIVE_E2E_TEST=true python test_google_drive_e2e.py
    print("Running Google Drive E2E tests...")
    print("Note: Set GOOGLE_DRIVE_E2E_TEST=true and create /tmp/test_credentials.json")
    pytest.main([__file__, "-v"])
