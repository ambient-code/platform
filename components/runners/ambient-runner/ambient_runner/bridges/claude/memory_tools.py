"""Memory MCP tools for storing and querying repo intelligence.

Provides three tools that give agents persistent project knowledge:
- memory_query: Search intelligence and findings by repo, file, category
- memory_store: Save file-level findings for future sessions
- memory_warn: Check if a file has known issues before modifying it
"""

import json
import logging
import os
from typing import Any, Callable, Optional

from ambient_runner.tools.intelligence_api import IntelligenceAPIClient

logger = logging.getLogger(__name__)


def create_memory_mcp_tools(
    sdk_tool_decorator: Callable,
    client: Optional[IntelligenceAPIClient] = None,
) -> list[Any]:
    """Create memory tools for the Claude Agent SDK.

    Args:
        sdk_tool_decorator: The claude_agent_sdk.tool decorator
        client: Optional IntelligenceAPIClient instance

    Returns:
        List of SDK tool functions
    """
    api_client = client or _create_default_client()
    if api_client is None:
        logger.warning(
            "Intelligence API client not available - memory tools will be skipped"
        )
        return []

    tools = []
    session_id = os.getenv("AGENTIC_SESSION_NAME", "")

    def _tool_response(data: dict) -> dict:
        """Helper to format successful tool response."""
        return {"content": [{"type": "text", "text": json.dumps(data, indent=2)}]}

    def _tool_error(error: Exception) -> dict:
        """Helper to format error tool response."""
        return {
            "content": [
                {
                    "type": "text",
                    "text": json.dumps(
                        {"success": False, "error": str(error)}, indent=2
                    ),
                }
            ],
            "isError": True,
        }

    # ── Tool 1: memory_query ──────────────────────────────────────────

    @sdk_tool_decorator(
        "memory_query",
        (
            "Search project memory for repo intelligence and findings. "
            "Use this when starting work on a repo or file you haven't seen before, "
            "or to check what previous sessions discovered. "
            "Returns repo-level architecture summaries and file-level findings."
        ),
        {
            "type": "object",
            "properties": {
                "repo_url": {
                    "type": "string",
                    "description": "Repository URL to query intelligence for",
                },
                "file_path": {
                    "type": "string",
                    "description": "Optional file path to filter findings (partial match)",
                },
                "category": {
                    "type": "string",
                    "description": "Filter findings by category",
                    "enum": ["investigation", "caveat", "review", "convention"],
                },
            },
            "required": ["repo_url"],
        },
    )
    async def memory_query(args: dict) -> dict:
        try:
            repo_url = args["repo_url"]
            intel = api_client.lookup_intelligence(repo_url)
            if not intel:
                return _tool_response(
                    {"found": False, "message": f"No intelligence stored for {repo_url}"}
                )

            result: dict[str, Any] = {"found": True, "intelligence": intel}

            file_path = args.get("file_path")
            category = args.get("category")
            if file_path or category:
                findings = api_client.list_findings(
                    intel["id"], file_path=file_path, category=category
                )
                result["findings"] = findings.get("items", [])
                result["findings_count"] = findings.get("total", len(result["findings"]))

            return _tool_response(result)
        except Exception as e:
            return _tool_error(e)

    tools.append(memory_query)

    # ── Tool 2: memory_store ──────────────────────────────────────────

    @sdk_tool_decorator(
        "memory_store",
        (
            "Store a finding about a file in project memory for future sessions. "
            "Use this when you discover something important: bug root causes, "
            "code caveats, conventions, or investigation findings. "
            "Future sessions touching this file will be warned."
        ),
        {
            "type": "object",
            "properties": {
                "repo_url": {
                    "type": "string",
                    "description": "Repository URL this finding belongs to",
                },
                "file_path": {
                    "type": "string",
                    "description": (
                        "File path within the repo "
                        "(e.g. 'components/backend/handlers/sessions.go')"
                    ),
                },
                "category": {
                    "type": "string",
                    "description": "Type of finding",
                    "enum": ["investigation", "caveat", "review", "convention"],
                },
                "title": {
                    "type": "string",
                    "description": "One-line summary of the finding",
                },
                "body": {
                    "type": "string",
                    "description": "Detailed description in markdown",
                },
                "severity": {
                    "type": "string",
                    "description": "Severity level (default: info)",
                    "enum": ["info", "warning", "critical"],
                },
                "source_ref": {
                    "type": "string",
                    "description": "Reference (e.g. 'pr:456', 'issue:PROJ-789')",
                },
                "confidence": {
                    "type": "number",
                    "description": "Confidence score 0.0-1.0",
                    "minimum": 0.0,
                    "maximum": 1.0,
                },
            },
            "required": ["repo_url", "file_path", "category", "title", "body"],
        },
    )
    async def memory_store(args: dict) -> dict:
        try:
            repo_url = args["repo_url"]

            intel = api_client.lookup_intelligence(repo_url)
            if not intel:
                return _tool_response(
                    {
                        "stored": False,
                        "message": (
                            f"No intelligence record for {repo_url}. "
                            "Create one first with memory_query or wait for auto-analysis."
                        ),
                    }
                )

            finding_data: dict[str, Any] = {
                "intelligence_id": intel["id"],
                "file_path": args["file_path"],
                "category": args["category"],
                "title": args["title"],
                "body": args["body"],
                "severity": args.get("severity", "info"),
                "source_type": "agent_analysis",
                "source_ref": args.get("source_ref"),
                "confidence": args.get("confidence"),
            }
            if session_id:
                finding_data["session_id"] = session_id

            finding = api_client.create_finding(finding_data)
            return _tool_response(
                {
                    "stored": True,
                    "finding_id": finding.get("id"),
                    "message": (
                        f"Finding stored for {args['file_path']}. "
                        "Future sessions will see this."
                    ),
                }
            )
        except Exception as e:
            return _tool_error(e)

    tools.append(memory_store)

    # ── Tool 3: memory_warn ───────────────────────────────────────────

    @sdk_tool_decorator(
        "memory_warn",
        (
            "Check if a file has any known findings from previous sessions. "
            "Use this before modifying a file, especially during bug fixes "
            "or reviews. Returns active warnings, investigation findings, "
            "and caveats."
        ),
        {
            "type": "object",
            "properties": {
                "file_path": {
                    "type": "string",
                    "description": (
                        "File path to check "
                        "(e.g. 'components/backend/handlers/sessions.go')"
                    ),
                },
                "repo_url": {
                    "type": "string",
                    "description": (
                        "Repository URL "
                        "(optional — uses first repo in project if omitted)"
                    ),
                },
            },
            "required": ["file_path"],
        },
    )
    async def memory_warn(args: dict) -> dict:
        try:
            file_path = args["file_path"]
            repo_url = args.get("repo_url")

            if not repo_url:
                return _tool_response(
                    {
                        "warnings": [],
                        "message": (
                            "No repo_url provided. Specify a repo_url to "
                            "check for known findings."
                        ),
                    }
                )

            intel = api_client.lookup_intelligence(repo_url)
            if not intel:
                return _tool_response(
                    {"warnings": [], "message": f"No intelligence for {repo_url}"}
                )

            findings = api_client.list_findings(intel["id"], file_path=file_path)
            items = findings.get("items", [])

            if not items:
                return _tool_response(
                    {
                        "warnings": [],
                        "message": f"No known findings for {file_path}",
                    }
                )

            return _tool_response(
                {
                    "warnings": items,
                    "count": len(items),
                    "message": f"{len(items)} known finding(s) for {file_path}",
                }
            )
        except Exception as e:
            return _tool_error(e)

    tools.append(memory_warn)

    return tools


def _create_default_client() -> Optional[IntelligenceAPIClient]:
    """Create a default IntelligenceAPIClient from environment variables.

    Returns:
        IntelligenceAPIClient instance, or None if required env vars are missing
    """
    api_url = (
        os.getenv("API_SERVER_URL", "")
        or os.getenv("BACKEND_API_URL", "")
    ).strip()

    if not api_url:
        logger.debug(
            "Intelligence API client cannot be created: "
            "API_SERVER_URL or BACKEND_API_URL not set"
        )
        return None

    try:
        return IntelligenceAPIClient(api_server_url=api_url)
    except ValueError as e:
        logger.warning(f"Failed to create intelligence API client: {e}")
        return None
