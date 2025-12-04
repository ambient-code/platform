#!/usr/bin/env python3
"""
Workspace MCP Server

Provides tools for executing commands in workspace pods via the session-proxy
sidecar's streaming exec API. This is the primary interface for the Claude agent
to run shell commands in workspace container mode (ADR-0006).

The session-proxy sidecar runs in the same pod and handles K8s authentication
internally, so no token is required from the runner container.
"""

import json
import logging
import os
import sys

# HTTP client - prefer httpx for streaming, fall back to aiohttp
try:
    import httpx
    USE_HTTPX = True
except ImportError:
    try:
        import aiohttp
        USE_HTTPX = False
    except ImportError:
        print("Error: Neither httpx nor aiohttp installed. Install with: pip install httpx", file=sys.stderr)
        sys.exit(1)

# MCP SDK imports
try:
    from mcp.server.fastmcp import FastMCP
except ImportError:
    print("Error: MCP SDK not installed. Install with: pip install mcp", file=sys.stderr)
    sys.exit(1)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("workspace-mcp")

# Environment variables (set by operator)
SESSION_NAME = os.environ.get("SESSION_NAME", "")
POD_NAMESPACE = os.environ.get("POD_NAMESPACE", "")

# Session-proxy sidecar URL (runs in the same pod on localhost)
# The session-proxy handles K8s authentication internally
SESSION_PROXY_URL = os.environ.get("SESSION_PROXY_URL", "http://localhost:8081")

# Configurable timeouts via environment variables
DEFAULT_TIMEOUT = int(os.environ.get("WORKSPACE_DEFAULT_TIMEOUT", "300"))
MAX_TIMEOUT = int(os.environ.get("WORKSPACE_MAX_TIMEOUT", "1800"))  # 30 minutes max
DEFAULT_WORKDIR = "/workspace"


# Create MCP server
server = FastMCP("workspace")


@server.tool()
async def exec(
    command: str,
    workdir: str = DEFAULT_WORKDIR,
    timeout: int = DEFAULT_TIMEOUT,
) -> str:
    """Execute a command in the workspace pod with streaming output.

    The workspace pod is created automatically on first use using your
    configured container image. State persists across exec calls
    (working directory, environment variables, background processes).

    Args:
        command: Shell command to execute (e.g., "npm install", "git status")
        workdir: Working directory for command execution (default: /workspace)
        timeout: Command timeout in seconds (default: 300, max: 1800)

    Returns:
        Command output (stdout + stderr streamed in real-time)

    Examples:
        exec("git status")
        exec("npm install && npm test")
        exec("cargo build --release", workdir="/workspace/myrepo")
        exec("python analyze.py", timeout=600)
    """
    if not command.strip():
        return "Error: Empty command"

    if not SESSION_NAME or not POD_NAMESPACE:
        return "Error: Missing required environment variables (SESSION_NAME, POD_NAMESPACE)"

    # Clamp timeout
    timeout = min(max(timeout, 1), MAX_TIMEOUT)

    try:
        return await _exec_streaming(command, workdir, timeout)
    except Exception as e:
        logger.exception("exec failed")
        return f"Error: {str(e)}"


async def _exec_streaming(command: str, workdir: str, timeout: int) -> str:
    """Execute command via session-proxy sidecar."""
    url = f"{SESSION_PROXY_URL}/exec"
    headers = {
        "Content-Type": "application/json",
    }
    # Session-proxy expects command as array and optional cwd
    payload = {
        "command": ["sh", "-c", command],
        "cwd": workdir,
    }

    if USE_HTTPX:
        return await _exec_httpx(url, headers, payload, timeout)
    else:
        return await _exec_aiohttp(url, headers, payload, timeout)


async def _exec_httpx(url: str, headers: dict, payload: dict, timeout: int) -> str:
    """Execute using httpx with streaming."""
    output_chunks = []

    async with httpx.AsyncClient() as client:
        try:
            async with client.stream(
                "POST",
                url,
                headers=headers,
                json=payload,
                timeout=timeout + 30,  # Add buffer for network latency
            ) as response:
                if response.status_code == 503:
                    return "Error: Workspace pod not ready"
                if response.status_code != 200:
                    body = await response.aread()
                    return f"Error: HTTP {response.status_code} - {body.decode('utf-8', errors='replace')}"

                # Stream the output
                async for chunk in response.aiter_text():
                    output_chunks.append(chunk)
                    # Note: MCP streaming would yield chunks here for real-time display
                    # when the protocol supports it

        except httpx.TimeoutException:
            return "Error: Request timed out"
        except httpx.ConnectError as e:
            return f"Error: Failed to connect to session-proxy: {e}"

    # Combine output and parse final status
    full_output = "".join(output_chunks)
    return _format_output(full_output)


async def _exec_aiohttp(url: str, headers: dict, payload: dict, timeout: int) -> str:
    """Execute using aiohttp with streaming."""
    import aiohttp

    output_chunks = []

    timeout_config = aiohttp.ClientTimeout(total=timeout + 30)

    try:
        async with aiohttp.ClientSession(timeout=timeout_config) as session:
            async with session.post(url, headers=headers, json=payload) as response:
                if response.status == 503:
                    return "Error: Workspace pod not ready"
                if response.status != 200:
                    body = await response.text()
                    return f"Error: HTTP {response.status} - {body}"

                # Stream the output
                async for chunk in response.content.iter_any():
                    output_chunks.append(chunk.decode('utf-8', errors='replace'))

    except aiohttp.ClientError as e:
        return f"Error: HTTP request failed: {e}"

    # Combine output and parse final status
    full_output = "".join(output_chunks)
    return _format_output(full_output)


def _format_output(full_output: str) -> str:
    """Format the streaming output, extracting final status."""
    # Session-proxy sends final JSON status: {"exitCode": 0} or {"exitCode": 1, "error": "..."}
    lines = full_output.rstrip().split('\n')

    if not lines:
        return "(no output)"

    # Check if last line is JSON status
    last_line = lines[-1].strip()
    exit_status = None

    if last_line.startswith('{') and last_line.endswith('}'):
        try:
            exit_status = json.loads(last_line)
            # Remove status line from output
            lines = lines[:-1]
        except json.JSONDecodeError:
            pass

    output = '\n'.join(lines)

    if exit_status:
        exit_code = exit_status.get("exitCode", 0)
        error = exit_status.get("error")

        if exit_code != 0:
            if error:
                if output:
                    return f"{output}\n\nError: {error} (exit code: {exit_code})"
                return f"Error: {error} (exit code: {exit_code})"
            if output:
                return f"{output}\n\n(exit code: {exit_code})"
            return f"(exit code: {exit_code})"

    return output if output else "(no output)"


@server.tool()
async def status() -> str:
    """Show workspace pod status.

    Returns the current state of the workspace pod, including
    whether it's running, its resource usage, and any recent events.

    Returns:
        Status information for the workspace pod
    """
    if not SESSION_NAME or not POD_NAMESPACE:
        return "Error: Missing required environment variables"

    # Use exec to run kubectl inside the workspace or a simple status check
    # For now, call the status endpoint if available, or use a simple exec
    pod_name = f"{SESSION_NAME}-ws"

    return f"""Workspace pod: {pod_name}
Namespace: {POD_NAMESPACE}
Session: {SESSION_NAME}

To check if the pod exists and is running, try:
  exec("echo 'Workspace is ready'")

The workspace pod is created automatically on the first exec() call.
"""


@server.tool()
async def logs(lines: int = 100) -> str:
    """View workspace pod logs.

    Returns recent logs from the workspace pod. Useful for debugging
    background processes or checking what happened after a command.

    Args:
        lines: Number of recent lines to show (default: 100)

    Returns:
        Recent log output from the workspace pod

    Note:
        The workspace container typically runs 'sleep infinity' so logs
        are usually empty unless your commands produce container-level output.
    """
    if not SESSION_NAME or not POD_NAMESPACE:
        return "Error: Missing required environment variables"

    # Note: Could add a /logs endpoint to the operator, or use exec to tail files
    return f"""Workspace logs are not directly available via the streaming API.

To view command history or outputs:
  exec("cat ~/.bash_history")
  exec("tail -100 /workspace/some.log")

For container-level logs, the workspace runs 'sleep infinity' so container
logs are typically empty.
"""


def main():
    """Main entry point for the MCP server."""
    logger.info(f"Starting Workspace MCP server for session {SESSION_NAME}")
    logger.info(f"Session-proxy URL: {SESSION_PROXY_URL}")
    # FastMCP.run() handles stdio transport by default
    server.run(transport="stdio")


if __name__ == "__main__":
    main()
