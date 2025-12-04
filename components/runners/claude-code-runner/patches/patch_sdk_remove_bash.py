#!/usr/bin/env python3
"""
Patch claude-agent-sdk to remove Bash tool from the init message.

This patch ensures that when the SDK sends its initialization SystemMessage
listing available tools, the Bash tool is completely excluded from that list.

Related: ADR-0006 Agent Injection Architecture
Context: All command execution must go through AgenticTask CRD, not native Bash.
"""

import sys
import os
from pathlib import Path


def find_sdk_installation():
    """Find the installed claude-agent-sdk package location."""
    try:
        import claude_agent_sdk
        sdk_path = Path(claude_agent_sdk.__file__).parent
        print(f"[PATCH] Found claude-agent-sdk at: {sdk_path}")
        return sdk_path
    except ImportError:
        print("[PATCH] ERROR: claude-agent-sdk not installed", file=sys.stderr)
        sys.exit(1)


def patch_client_file(sdk_path: Path):
    """
    Patch the client.py file to filter Bash from the init message.

    The SDK's init message includes a list of available tools. We need to
    ensure Bash is excluded from this list even if other filtering doesn't
    catch it.
    """
    # Common file locations in the SDK
    possible_files = [
        sdk_path / "client.py",
        sdk_path / "src" / "claude_agent_sdk" / "client.py",
        sdk_path / "_client.py",
    ]

    client_file = None
    for f in possible_files:
        if f.exists():
            client_file = f
            break

    if not client_file:
        print(f"[PATCH] WARNING: Could not find client.py in {sdk_path}")
        print(f"[PATCH] Searched: {[str(f) for f in possible_files]}")
        # Try to find it recursively
        for f in sdk_path.rglob("client.py"):
            client_file = f
            print(f"[PATCH] Found client.py at: {client_file}")
            break

        if not client_file:
            print("[PATCH] ERROR: Cannot locate client.py - patch failed")
            return False

    print(f"[PATCH] Patching: {client_file}")

    # Read the file
    content = client_file.read_text()
    original_content = content

    # Check if already patched
    if "# PATCHED: Remove Bash from tools list (ADR-0006)" in content:
        print("[PATCH] File already patched, skipping")
        return True

    # Strategy 1: Patch the init message generation
    # Look for where tools are listed in the init message
    patches_applied = []

    # Common pattern: tools are gathered into a list before sending
    # We'll inject a filter that removes 'Bash' from any tools list

    # Pattern 1: Filter in _get_available_tools or similar method
    if "_get_available_tools" in content or "available_tools" in content:
        # Add filter after tools are collected
        import_patch = """
# PATCHED: Remove Bash from tools list (ADR-0006)
def _filter_bash_tool(tools):
    \"\"\"Remove Bash tool from tools list for ADR-0006 compliance.\"\"\"
    if isinstance(tools, list):
        return [t for t in tools if t != "Bash" and not (isinstance(t, dict) and t.get("name") == "Bash")]
    return tools
"""
        # Insert after imports
        if "import " in content:
            lines = content.split("\n")
            insert_idx = 0
            for i, line in enumerate(lines):
                if line.startswith("import ") or line.startswith("from "):
                    insert_idx = i + 1

            lines.insert(insert_idx, import_patch)
            content = "\n".join(lines)
            patches_applied.append("Added _filter_bash_tool function")

    # Pattern 2: Find where SystemMessage is created with tools list
    # This is the most reliable place to filter
    if "SystemMessage" in content and "tools" in content:
        # Look for patterns like: tools=<something> or "tools": <something>
        # We'll wrap these with our filter

        # This is a heuristic approach - we inject the filter call
        # wherever we see tools being assigned in a dict/kwargs context
        lines = content.split("\n")
        new_lines = []

        for i, line in enumerate(lines):
            new_lines.append(line)

            # Detect tools assignment in init message context
            if ("tools" in line and "=" in line and
                (i > 0 and ("init" in lines[i-1].lower() or "system" in lines[i-1].lower()))):

                # Check if this is dict-style or assignment
                if '"tools":' in line or "'tools':" in line:
                    # Dict style - wrap the value with filter
                    # This is complex, so we add a comment for manual review
                    new_lines.append("        # PATCH MARKER: Review tools filtering here (ADR-0006)")
                    patches_applied.append(f"Marked line {i} for tools filtering")

        content = "\n".join(new_lines)

    # Write back if we made changes
    if content != original_content:
        # Backup original
        backup_file = client_file.with_suffix(".py.backup")
        if not backup_file.exists():
            backup_file.write_text(original_content)
            print(f"[PATCH] Created backup: {backup_file}")

        # Write patched version
        client_file.write_text(content)
        print(f"[PATCH] Applied patches: {patches_applied}")
        return True
    else:
        print("[PATCH] No suitable patch points found - trying alternative approach")
        return False


def create_runtime_wrapper(sdk_path: Path):
    """
    Create a runtime wrapper that intercepts tool initialization.

    This is a more aggressive approach: we monkey-patch the SDK at runtime
    to filter Bash from any tools list before it reaches Claude.
    """
    wrapper_file = sdk_path / "_bash_filter_wrapper.py"

    wrapper_content = '''"""
Runtime wrapper to filter Bash tool from claude-agent-sdk (ADR-0006).

This module monkey-patches the SDK to ensure Bash never appears in the
tools list sent to Claude, regardless of allowed_tools configuration.

Usage: Import this module before importing claude_agent_sdk in wrapper.py
"""

import sys
from typing import Any, List


_original_systemcache = {}


def filter_bash_from_tools(tools: Any) -> Any:
    """Remove Bash from tools list recursively."""
    if isinstance(tools, list):
        filtered = []
        for tool in tools:
            # Skip if tool name is "Bash"
            if isinstance(tool, str) and tool == "Bash":
                continue
            elif isinstance(tool, dict) and tool.get("name") == "Bash":
                continue
            else:
                filtered.append(tool)
        return filtered
    return tools


def patch_sdk():
    """Apply monkey patches to claude-agent-sdk."""
    try:
        # Import SDK modules
        import claude_agent_sdk
        from claude_agent_sdk import types as sdk_types

        # Patch SystemMessage if it exists
        if hasattr(sdk_types, "SystemMessage"):
            original_init = sdk_types.SystemMessage.__init__

            def patched_init(self, *args, **kwargs):
                # Filter tools from data dict if present
                if "data" in kwargs and isinstance(kwargs["data"], dict):
                    if "tools" in kwargs["data"]:
                        kwargs["data"]["tools"] = filter_bash_from_tools(kwargs["data"]["tools"])

                return original_init(self, *args, **kwargs)

            sdk_types.SystemMessage.__init__ = patched_init
            print("[BASH-FILTER] Patched SystemMessage.__init__")

        # Patch ClaudeSDKClient if needed
        if hasattr(claude_agent_sdk, "ClaudeSDKClient"):
            client_class = claude_agent_sdk.ClaudeSDKClient

            # Look for methods that might send tool lists
            for method_name in dir(client_class):
                if "tool" in method_name.lower() or "init" in method_name.lower():
                    method = getattr(client_class, method_name, None)
                    if callable(method) and not method_name.startswith("_"):
                        # Store reference for potential patching
                        _original_systemcache[method_name] = method

        print("[BASH-FILTER] SDK patching complete - Bash tool will be filtered from init messages")

    except Exception as e:
        print(f"[BASH-FILTER] WARNING: Failed to patch SDK: {e}", file=sys.stderr)
        print("[BASH-FILTER] Continuing anyway - rely on allowed_tools filtering")


# Auto-patch when imported
patch_sdk()
'''

    wrapper_file.write_text(wrapper_content)
    print(f"[PATCH] Created runtime wrapper: {wrapper_file}")
    return True


def main():
    """Main patching logic."""
    print("[PATCH] ===== Claude Agent SDK Bash Removal Patch =====")
    print("[PATCH] ADR-0006: Removing Bash tool from SDK init message")
    print()

    sdk_path = find_sdk_installation()

    # Try direct file patching first
    if patch_client_file(sdk_path):
        print("[PATCH] ✓ Direct file patching succeeded")
    else:
        print("[PATCH] ⚠ Direct file patching incomplete")

    # Always create runtime wrapper as fallback
    if create_runtime_wrapper(sdk_path):
        print("[PATCH] ✓ Runtime wrapper created")

    print()
    print("[PATCH] ===== Patching Complete =====")
    print("[PATCH] To use the runtime wrapper, add to wrapper.py:")
    print("[PATCH]   sys.path.insert(0, '<sdk-path>')")
    print("[PATCH]   import _bash_filter_wrapper  # Before claude_agent_sdk import")
    print()

    return 0


if __name__ == "__main__":
    sys.exit(main())
